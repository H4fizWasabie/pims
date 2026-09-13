# PIMS

Procurement & Inventory Management System for Starlight Veterinary Hospital.

PIMS is a Go web application for managing procurement requests, inventory workflows, stock records, and approvals through a single-page interface.

[![CI](https://github.com/H4fizWasabie/pims/actions/workflows/ci.yml/badge.svg)](https://github.com/H4fizWasabie/pims/actions/workflows/ci.yml)

## Features

- Dashboard with stock and workflow summaries
- Lab, pharmacy, medical ward, and surgical ward purchase orders
- Internal indents with approval and rejection workflows
- Goods Receipt Note (GRN) entry
- Stock take and scan history
- Stock disposal and variance analysis
- Expiry tracking and remarks
- New item specification requests with approval
- Master item data and inventory browsing
- Admin-only user management and data import
- Session authentication with demo read-only access
- Optional AI-assisted stock-take image analysis through OpenRouter or Gemini

## Stock data ownership

Procura is the source of truth for current stock quantities. PIMS opens the Procura SQLite database read-only and uses it for inventory displays, searches, dashboard counts, and stock-related checks.

To change stock quantities, upload the Stock Balance History file in Procura. PIMS records procurement and stock activity but does not mutate Procura-owned quantities.

## Technology

- Go 1.25+
- `net/http` and embedded static assets via `embed.FS`
- PostgreSQL for PIMS data, users, sessions, approvals, and audit records
- SQLite read-only access for Procura stock data
- Vanilla HTML, CSS, and JavaScript single-page frontend
- GitHub Actions for `go vet` and `go test`

## Local development

### Requirements

- Go 1.25 or newer
- PostgreSQL
- A Procura SQLite database containing the `items` table, or a configured compatible stock database

### Run

Create a PostgreSQL database, then configure the connection and Procura database path:

```bash
export DATABASE_URL='postgres://pims:pims@localhost:5432/pims?sslmode=disable'
export PROCURA_DB_PATH='/path/to/procura.sqlite'
export SESSION_SECRET='replace-with-a-long-random-secret'

go run .
```

PIMS applies its PostgreSQL migrations automatically at startup and listens on `http://localhost:8083` by default. The default port can be changed with `PORT`.

For local development, the migration seeds `admin@pims.local` with the password `admin123`. Change or remove this account before using a deployment outside development.

### Optional OCR configuration

Stock-take image analysis uses OpenRouter first and Gemini as a fallback when configured:

```bash
export OPENROUTER_API_KEY='...'
export OPENROUTER_MODEL='google/gemma-4-31b-it:free'
export GEMINI_API_KEY='...'
```

Role lists are comma-separated environment variables:

```bash
export INDENT_APPROVERS='approver@example.com'
export SPEC_APPROVERS='approver@example.com'
export MASTER_ADMINS='admin@example.com'
```

See [`deploy/pims.env.example`](deploy/pims.env.example) for the complete configuration template.

## Testing

Tests use PostgreSQL. Set `TEST_DATABASE_URL` or `DATABASE_URL` to a test database, then run:

```bash
go vet ./...
go test ./... -v -count=1
```

The test suite covers authentication, protected routes, procurement workflows, stock ownership boundaries, stock take, disposal, expiry, specifications, dashboard data, analysis, and orders.

## Deployment

The [`deploy/`](deploy) directory contains examples for:

- Environment configuration
- A systemd service for the single PIMS binary
- Caddy reverse proxy routing under `/pims`

Build the application with:

```bash
go build -o pims .
```

Keep production secrets outside the repository and replace the development session secret and seeded admin credentials before exposing the service.

## Project layout

```text
.
├── main.go              # Server startup and route registration
├── internal/auth        # Sessions and role checks
├── internal/config      # Environment configuration
├── internal/db          # PostgreSQL and Procura data access
├── internal/handler     # HTTP handlers and middleware
├── internal/ocr         # Optional stock-take OCR providers
├── static/index.html    # Embedded single-page frontend
└── deploy/              # Deployment templates
```

PDF generation for GRNs, PRFs, and analysis reports is not currently included.
