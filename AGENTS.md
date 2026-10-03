# PIMS AGENTS.md

## graphify

Use graphify for semantic codebase queries:
```bash
graphify query "<question>"        # scoped subgraph
graphify path "<A>" "<B>"           # relationships
graphify explain "<concept>"        # focused concept
```
After code changes: `graphify update .`

## codegraph

Use codegraph for symbol-level queries (functions, types, methods):
```
codegraph query "<search>"
codegraph explore "<topic>"
```

Before reading any code file, query graphify or codegraph first.

## ponytail rules

- Smallest diff that works. One-liner over boilerplate.
- Stdlib over deps. No interface with one impl.
- If something already exists in the codebase, reuse it.
- YAGNI: skip speculative features, abstractions, "for later" scaffolding.
- Bug fix = root cause, not symptom. Fix once where all callers route through.

## modular design

- Every module (`internal/handler/*`, `internal/db/*`) is independent.
- One module breaking must not crash the server or block other modules.
- Handlers recover from panics. DB errors return `{success: false, message: "..."}` — never 500 silently.
- Each module has its own handler file and db file. No cross-module imports in handler layer.

## commit style

Conventional Commits: `type(scope): description`
- `feat(indent): submit indent endpoint`
- `fix(grn): double-entry guard`
- `chore(ci): add github actions`

## CI

GitHub Actions on push: `go vet ./...` + `go test ./...`

## production / deploy

Production runs on the user's VPS. Reach it with `ssh vps` (host `vultr`, Tailscale IP 100.101.53.98).

- Live URL: https://pims.wasabietech.com (Caddy terminates TLS, proxies to `127.0.0.1:8083`)
- Service: `pims.service` (runs as user `pims`); logs: `sudo journalctl -u pims`
- Binary: `/usr/local/bin/pims`; env file: `/opt/pims.env` (root-only, holds secrets; never print or commit it)
- Server checkout used for builds: `/home/theoses/icm-workspaces/pims`
- Procura (stock source of truth) is a separate app on the same host, port 8082

Read-only inspection over ssh is fine. Anything that changes production (restart, install, DB access, reading `/opt/pims.env`) needs the user's explicit go-ahead in the current conversation.

Deploy flow (matches `/home/theoses/pims-merge-deploy.sh`):
1. PR to `main`, wait for CI green, merge.
2. Back up first: `pg_dump` the DB to `/root/pims-pre-<change>-<date>.sql.gz` and copy `/usr/local/bin/pims` to `/root/pims.bin.pre-<change>` (needs `sudo`).
3. On the server: `git checkout main && git pull`, `CGO_ENABLED=0 go build -o /tmp/pims-new .`, `sudo install -o root -g root -m 755 /tmp/pims-new /usr/local/bin/pims`, `sudo systemctl restart pims.service`.
4. Verify: `/api/master/all` unauthenticated returns 401; demo login (`POST /api/auth/demo`) shows only `DEMO-*` items; `admin@pims.local`/`admin123` login returns 401; journal has no errors.
5. Rollback: install the saved binary back and restart.

First admin on a fresh install comes from `ADMIN_EMAIL` / `ADMIN_PASSWORD` (there is no seeded account).

## testing locally

Tests need Postgres via `TEST_DATABASE_URL`; without it most tests skip silently. The local Postgres on :5432 needs credentials you may not have; start a throwaway one instead: `initdb -D /tmp/pimspg -U pims --auth=trust`, `pg_ctl ... -o "-p 54329 -k /tmp"`, `createdb pims_test`, then `TEST_DATABASE_URL='postgres://pims@127.0.0.1:54329/pims_test?sslmode=disable' go test ./...`.
