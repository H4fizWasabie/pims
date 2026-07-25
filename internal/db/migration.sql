CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS master_items (
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

CREATE TABLE IF NOT EXISTS inventory (
    stock_id TEXT PRIMARY KEY REFERENCES master_items(stock_id) ON DELETE CASCADE,
    item_name TEXT NOT NULL,
    current_stock NUMERIC(12,2) DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS indents (
    id SERIAL PRIMARY KEY,
    indent_id TEXT NOT NULL,
    request_date TIMESTAMPTZ DEFAULT NOW(),
    requester TEXT NOT NULL,
    status TEXT DEFAULT 'Pending',
    item_name TEXT NOT NULL,
    stock_id TEXT NOT NULL,
    uom TEXT DEFAULT '',
    requested_qty NUMERIC(12,2) DEFAULT 0,
    action_log TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_indents_status ON indents(status);

CREATE TABLE IF NOT EXISTS expiry_tracking (
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

CREATE TABLE IF NOT EXISTS expiry_monthly_qty (
    id SERIAL PRIMARY KEY,
    expiry_tracking_id INTEGER REFERENCES expiry_tracking(id) ON DELETE CASCADE,
    month_key TEXT NOT NULL,
    qty NUMERIC(12,2) DEFAULT 0,
    UNIQUE(expiry_tracking_id, month_key)
);

CREATE TABLE IF NOT EXISTS grn_logs (
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

CREATE TABLE IF NOT EXISTS grn_items (
    id SERIAL PRIMARY KEY,
    grn_log_id INTEGER REFERENCES grn_logs(id) ON DELETE CASCADE,
    item_name TEXT NOT NULL,
    qty_po NUMERIC(12,2) DEFAULT 0,
    qty_do NUMERIC(12,2) DEFAULT 0,
    qty_inv NUMERIC(12,2) DEFAULT 0,
    uom TEXT DEFAULT '',
    batch TEXT DEFAULT '',
    status TEXT DEFAULT '',
    remarks TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS disposal_logs (
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

CREATE TABLE IF NOT EXISTS new_item_requests (
    id SERIAL PRIMARY KEY,
    req_id TEXT UNIQUE NOT NULL,
    requester TEXT NOT NULL,
    request_date TIMESTAMPTZ DEFAULT NOW(),
    item_name TEXT NOT NULL,
    item_group TEXT DEFAULT '',
    uom TEXT DEFAULT '',
    cost NUMERIC(12,2) DEFAULT 0,
    justification TEXT DEFAULT '',
    status TEXT DEFAULT 'Pending Review',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stock_takes (
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
CREATE INDEX IF NOT EXISTS idx_stock_takes_date ON stock_takes(take_date);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);

CREATE TABLE IF NOT EXISTS id_counters (
    key TEXT PRIMARY KEY,
    counter INTEGER DEFAULT 1
);

-- Default admin user (password: admin123)
-- bcrypt hash for 'admin123'
INSERT INTO users (email, password_hash, role)
VALUES ('admin@pims.local', '$2a$10$nXrtCI3rOJSiycc3unq0w.rEnlt7CIzUpy/zqofVmq1Fni6RQUpRW', 'admin')
ON CONFLICT (email) DO NOTHING;

CREATE TABLE IF NOT EXISTS system_logs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    log_type TEXT NOT NULL DEFAULT 'INFO',
    message TEXT NOT NULL,
    user_email TEXT DEFAULT '',
    stack_trace TEXT DEFAULT ''
);
