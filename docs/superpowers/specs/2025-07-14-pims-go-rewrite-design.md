# PIMS Go Rewrite — Design Spec

**Date:** 2025-07-14
**Goal:** Rewrite PIMS (ProcurePilot Inventory Management System) from Google Apps Script to Go, deploy to VPS with PostgreSQL, achieve full feature parity with original GAS version (excluding PDF generation).

---

## 1. Overview

PIMS is an inventory management SPA for Starlight Veterinary Hospital. The current version runs on Google Apps Script with Google Sheets as the database. The rewrite moves to Go + PostgreSQL on a VPS behind Caddy reverse proxy.

**In scope:** All 10 modules at feature parity
**Out of scope:** PDF generation (GRN PDF, PRF PDF, Analysis PDF) — deferred
**Out of scope:** New features — deferred until after parity ships

---

## 2. Stack

| Layer | Technology |
|-------|------------|
| Backend | Go (stdlib `net/http` + minimal deps) |
| Database | PostgreSQL |
| Auth | bcrypt passwords + session cookies (`users` table) |
| Frontend | Vanilla SPA (HTML/CSS/JS ported from GAS, embedded via `embed.FS`) |
| AI OCR | OpenRouter API (primary) + Gemini API (fallback) |
| CI/CD | GitHub Actions (build, lint, test) |
| Deploy | Single Go binary on VPS port `8083`, Caddy reverse proxy at `/pims` |

---

## 3. Architecture

```
Caddy (80/443)
  ├─ /          → /var/www/landing (static landing page)
  └─ /pims/*    → localhost:8083 (Go binary)

Go binary (localhost:8083)
  ├─ GET  /pims/           → SPA shell (Index.html)
  ├─ GET  /pims/api/...    → JSON API endpoints
  ├─ POST /pims/api/...    → JSON API endpoints (mutations)
  └─ /pims/static/         → embedded CSS/JS/fonts/assets
```

### Frontend SPA

Same architecture as GAS version:
- `Index.html` — shell with sidebar, header, overlay
- 10 module HTML files loaded inline via server-side includes
- Client-side JS: `switchTab()` SPA navigation, mobile sidebar toggle, module-specific JS
- All CSS inlined in `<style>` blocks (as in original)

Server renders the full SPA shell with all modules embedded on first load. Module switching is purely client-side (hide/show divs). Each module makes its own `google.script.run`-style calls ported to `fetch()` against `/pims/api/...` endpoints.

---

## 4. Database Schema

### `users` (new)
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',  -- 'user', 'indent_approver', 'spec_approver', 'admin'
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `master_items`
```sql
CREATE TABLE master_items (
    stock_id TEXT PRIMARY KEY,
    item_name TEXT NOT NULL,
    uom TEXT DEFAULT '',
    item_group TEXT DEFAULT '',
    cost NUMERIC(12,2) DEFAULT 0,
    last_supplier TEXT DEFAULT '',
    product_status TEXT DEFAULT 'Available',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `inventory`
```sql
CREATE TABLE inventory (
    stock_id TEXT PRIMARY KEY REFERENCES master_items(stock_id),
    item_name TEXT NOT NULL,
    current_stock NUMERIC(12,2) DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `indents`
```sql
CREATE TABLE indents (
    id SERIAL PRIMARY KEY,
    indent_id TEXT NOT NULL,        -- "REQ-DDMM-HHmm"
    request_date TIMESTAMPTZ DEFAULT NOW(),
    requester TEXT NOT NULL,
    status TEXT DEFAULT 'Pending',  -- 'Pending', 'Approved', 'Rejected'
    item_name TEXT NOT NULL,
    stock_id TEXT NOT NULL,
    uom TEXT DEFAULT '',
    requested_qty NUMERIC(12,2) DEFAULT 0,
    action_log TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_indents_status ON indents(status);
CREATE INDEX idx_indents_indent_id ON indents(indent_id);
```

### `expiry_tracking`
```sql
CREATE TABLE expiry_tracking (
    id SERIAL PRIMARY KEY,
    stock_id TEXT NOT NULL,
    item_name TEXT NOT NULL,
    batch_no TEXT NOT NULL,
    expiry_date DATE NOT NULL,
    uom TEXT DEFAULT '',
    remarks TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(stock_id, batch_no)
);

CREATE TABLE expiry_monthly_qty (
    id SERIAL PRIMARY KEY,
    expiry_tracking_id INTEGER REFERENCES expiry_tracking(id),
    month_key TEXT NOT NULL,        -- "MMM-YYYY"
    qty NUMERIC(12,2) DEFAULT 0,
    UNIQUE(expiry_tracking_id, month_key)
);
```

### `grn_logs`
```sql
CREATE TABLE grn_logs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    grn_no TEXT UNIQUE NOT NULL,
    supplier TEXT NOT NULL,
    do_date TEXT DEFAULT '',
    invoice_no TEXT DEFAULT '',
    po_no TEXT DEFAULT '',
    created_by TEXT DEFAULT '',
    submission_token TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE grn_items (
    id SERIAL PRIMARY KEY,
    grn_log_id INTEGER REFERENCES grn_logs(id),
    item_name TEXT NOT NULL,
    qty_po NUMERIC(12,2) DEFAULT 0,
    qty_do NUMERIC(12,2) DEFAULT 0,
    qty_inv NUMERIC(12,2) DEFAULT 0,
    uom TEXT DEFAULT '',
    batch TEXT DEFAULT '',
    status TEXT DEFAULT '',
    remarks TEXT DEFAULT ''
);
```

### `disposal_logs`
```sql
CREATE TABLE disposal_logs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    stock_id TEXT NOT NULL,
    item_name TEXT NOT NULL,
    batch_no TEXT DEFAULT '',
    qty_disposed NUMERIC(12,2) DEFAULT 0,
    unit_cost NUMERIC(12,2) DEFAULT 0,
    total_loss NUMERIC(12,2) DEFAULT 0,
    reason TEXT DEFAULT '',
    remarks TEXT DEFAULT '',
    user_email TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `new_item_requests` (specs)
```sql
CREATE TABLE new_item_requests (
    id SERIAL PRIMARY KEY,
    req_id TEXT UNIQUE NOT NULL,
    requester TEXT NOT NULL,
    request_date TIMESTAMPTZ DEFAULT NOW(),
    item_name TEXT NOT NULL,
    item_group TEXT DEFAULT '',
    uom TEXT DEFAULT '',
    cost NUMERIC(12,2) DEFAULT 0,
    justification TEXT DEFAULT '',
    status TEXT DEFAULT 'Pending Review',  -- 'Pending Review', 'Approved', 'Rejected'
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `stock_takes`
```sql
CREATE TABLE stock_takes (
    id SERIAL PRIMARY KEY,
    take_date DATE NOT NULL DEFAULT CURRENT_DATE,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    location TEXT DEFAULT '',
    stock_id TEXT NOT NULL,
    item_name TEXT DEFAULT '',
    uom TEXT DEFAULT '',
    item_group TEXT DEFAULT '',
    cost NUMERIC(12,2) DEFAULT 0,
    physical_qty NUMERIC(12,2) DEFAULT 0,
    batch_no TEXT DEFAULT '',
    expiry_date TEXT DEFAULT '',
    user_email TEXT DEFAULT ''
);
CREATE INDEX idx_stock_takes_date ON stock_takes(take_date);
```

### `sessions`
```sql
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `id_counters`
```sql
CREATE TABLE id_counters (
    key TEXT PRIMARY KEY,   -- "GRN_20250714", "PRF_20250714", "REQ"
    counter INTEGER DEFAULT 1
);
```

---

## 5. API Endpoints

All endpoints return JSON. Auth-protected endpoints require `Cookie: session=<token>`.

### Auth
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/pims/api/auth/login` | No | `{email, password}` → session cookie |
| POST | `/pims/api/auth/logout` | Yes | Clear session |
| GET | `/pims/api/auth/me` | Yes | Current user info + role |

### Master Items
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/master/chunk?page=&pageSize=` | Yes | Paginated master data |
| GET | `/pims/api/master/search?q=` | Yes | Search by stock_id or name |
| POST | `/pims/api/master/replace` | Admin | Replace all master data |
| GET | `/pims/api/master/all` | Yes | All items with cost, supplier, stock (for orders) |

### Inventory
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/inventory/chunk?page=&pageSize=` | Yes | Paginated inventory |
| POST | `/pims/api/inventory/replace` | Admin | Replace all inventory data |

### Indents
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/indent/master-data` | Yes | All items with stock (for search) |
| POST | `/pims/api/indent/submit` | Yes | Submit indent request |
| POST | `/pims/api/indent/approve` | Approver | Approve + deduct stock |
| POST | `/pims/api/indent/reject` | Approver | Reject indent line |

### GRN
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/grn/master-data` | Yes | Items + suppliers |
| POST | `/pims/api/grn/submit` | Yes | Log GRN (no PDF) |

### Stock Take
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/stocktake/today` | Yes | Today's entries |
| POST | `/pims/api/stocktake/submit` | Yes | Log stock take item |
| POST | `/pims/api/stocktake/analyze-image` | Yes | AI OCR (OpenRouter → Gemini fallback) |

### Disposal
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/disposal/search?q=` | Yes | Search batches |
| POST | `/pims/api/disposal/submit` | Yes | Log disposal + deduct inventory |

### Analysis
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/pims/api/analysis/run` | Yes | Run variance analysis |
| GET | `/pims/api/analysis/today` | Yes | Get today's analysis data |

### Expiry
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/expiry/list?page=&pageSize=` | Yes | Paginated expiry items |
| POST | `/pims/api/expiry/update-remark` | Yes | Update remarks on a row |

### Specs (New Items)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/pims/api/spec/submit` | Yes | Submit new item request |
| POST | `/pims/api/spec/approve` | Spec Approver | Approve + add to master |
| POST | `/pims/api/spec/reject` | Spec Approver | Reject |

### Dashboard
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/dashboard/summary` | Yes | All dashboard stats + pending lists |

### Order (PRF)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/pims/api/order/prf-number` | Yes | Get next PRF number (no PDF) |

---

## 6. Go Package Structure

```
pims/
├── main.go                 # Entry point, server setup, routes
├── go.mod
├── go.sum
├── static/                 # embed.FS: all frontend assets
│   ├── index.html          # SPA shell
│   ├── dashboard.html
│   ├── lab_order.html
│   ├── pharmacy_order.html
│   ├── indent_form.html
│   ├── specification_form.html
│   ├── grn.html
│   ├── stock_take.html
│   ├── stock_disposal.html
│   ├── stock_analysis.html
│   ├── expiry_tracking.html
│   └── logo.png            # Starlight Vet logo
├── internal/
│   ├── config/
│   │   └── config.go       # Env vars, constants, role lists
│   ├── db/
│   │   ├── db.go           # Connection pool, migrations
│   │   ├── users.go        # User queries
│   │   ├── master.go       # Master items queries
│   │   ├── inventory.go    # Inventory queries
│   │   ├── indents.go      # Indent queries
│   │   ├── grn.go          # GRN queries
│   │   ├── stocktake.go    # Stock take queries
│   │   ├── disposal.go     # Disposal queries
│   │   ├── analysis.go     # Analysis queries
│   │   ├── expiry.go       # Expiry queries
│   │   ├── specs.go        # New item request queries
│   │   ├── dashboard.go    # Dashboard aggregation queries
│   │   └── counters.go     # ID generation (GRN, PRF, REQ)
│   ├── auth/
│   │   ├── auth.go         # Login, logout, session middleware
│   │   └── roles.go        # Role-check middleware
│   ├── handler/
│   │   ├── handler.go      # Shared helpers (JSON responses, error handling)
│   │   ├── auth_handler.go
│   │   ├── master_handler.go
│   │   ├── inventory_handler.go
│   │   ├── indent_handler.go
│   │   ├── grn_handler.go
│   │   ├── stocktake_handler.go
│   │   ├── disposal_handler.go
│   │   ├── analysis_handler.go
│   │   ├── expiry_handler.go
│   │   ├── spec_handler.go
│   │   ├── dashboard_handler.go
│   │   ├── order_handler.go
│   │   └── spa_handler.go   # Serves Index + static files
│   └── ocr/
│       └── ocr.go           # OpenRouter + Gemini image analysis
├── migrations/
│   └── 001_init.sql         # Full schema DDL
└── .github/
    └── workflows/
        └── ci.yml           # Build, lint, test on push
```

---

## 7. Authentication & Authorization

### Login flow
1. User POSTs `{email, password}` to `/pims/api/auth/login`
2. Server bcrypt-compares, generates random session token
3. Stores token in `sessions` table with expiry (24h)
4. Returns `Set-Cookie` header with session token
5. All subsequent requests validate session via middleware

### Roles (matching GAS Config.gs)
| Role | Access |
|------|--------|
| `admin` | Master data replace, inventory replace |
| `indent_approver` | Approve/reject indents |
| `spec_approver` | Approve/reject new item requests |
| `user` | Everything else (submit forms, view data) |

Admin emails and approver lists are configurable via environment variables (not hardcoded in source).

---

## 8. ID Generation

Matches GAS format:
- **GRN:** `GRN-YYYYMMDD-NNN` (counter per day, 3-digit pad)
- **PRF:** `PRF-YYYYMMDD-NNN` (counter per day, 3-digit pad)
- **Indent:** `REQ-DDMM-HHmm` (timestamp-based)

Implemented via `id_counters` table with row-level locks (equivalent to `LockService` in GAS).

---

## 9. Concurrency

GAS uses `LockService.getScriptLock()` for writes. Go equivalent:
- PostgreSQL row-level locks (`SELECT ... FOR UPDATE`)
- Transactions for multi-step operations (approve indent = deduct stock + update status)
- Unique constraints to prevent duplicates (submission_token, stock_id+batch in expiry)

---

## 10. AI Image Analysis (Stock Take)

Same dual-provider flow as GAS:
1. Try OpenRouter first (`google/gemma-4-31b-it:free` or configured model)
2. On failure, fall back to Gemini (`gemini-2.5-flash-lite`)
3. Returns `{productName, batchNumber, expiryDate, _model}`

API keys come from environment variables.

---

## 11. SPA Frontend Port

### Strategy
- Copy all HTML module files from GAS into `static/`
- Replace `google.script.run.withSuccessHandler(fn).serverMethod()` with `fetch('/pims/api/...').then(r => r.json()).then(fn)`
- Replace `<?!= include('ModuleName'); ?>` with Go `{{ template "module" }}` (server-side includes at render time)
- Font Awesome and jsPDF remain CDN-loaded (same as GAS)
- All CSS stays inline (same as GAS)
- Mobile responsive behavior unchanged

### JS changes per module
Every module has its own inline `<script>` with `google.script.run` calls. Each needs:

```js
// Before (GAS):
google.script.run.withSuccessHandler(onSuccess).serverMethod(args);

// After (Go):
fetch('/pims/api/server-method', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(args)
}).then(r => r.json()).then(onSuccess);
```

---

## 12. Deployment

### VPS Setup
```bash
# PostgreSQL (if not already installed)
apt install postgresql
sudo -u postgres createuser pims
sudo -u postgres createdb pims

# Go binary
# Build for linux/amd64, scp to VPS, run as systemd service
```

### systemd unit
```
[Unit]
Description=PIMS Inventory Management
After=network.target postgresql.service

[Service]
Type=simple
User=pims
WorkingDirectory=/opt/pims
EnvironmentFile=/opt/pims/pims.env
ExecStart=/opt/pims/pims
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Caddy config addition
```
handle_path /pims/* {
    reverse_proxy localhost:8083
}
```

### Environment variables (`pims.env`)
```
DATABASE_URL=postgres://pims:password@localhost:5432/pims?sslmode=disable
PORT=8083
SESSION_SECRET=<random-64-char>
OPENROUTER_API_KEY=sk-or-v1-...
OPENROUTER_MODEL=google/gemma-4-31b-it:free
GEMINI_API_KEY=...
INDENT_APPROVERS=kisame350@gmail.com,anushambigai@starlight-vet.com.my,...
SPEC_APPROVERS=kisame350@gmail.com,...
MASTER_ADMINS=kisame350@gmail.com,anushambigai@starlight-vet.com.my,...
```

---

## 13. GitHub Actions CI

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: pims
          POSTGRES_PASSWORD: pims
          POSTGRES_DB: pims_test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go vet ./...
      - run: go test ./... -v
```

---

## 14. What's NOT in scope (deferred)

- PDF generation (GRN PDF, PRF PDF, Analysis PDF) — the frontend buttons remain, but backend returns "not yet implemented"
- Google Sheets import/export
- Email notifications
- Audit trail beyond existing logs
- Multi-tenancy
- Rate limiting
- Any new features

---

## 15. Migration Path

1. Deploy Go app alongside existing GAS app (different URL)
2. Run schema migrations on PostgreSQL
3. Optionally migrate existing Google Sheets data → PostgreSQL (one-time script)
4. Switch users to new URL
5. GAS app remains as read-only fallback during transition
