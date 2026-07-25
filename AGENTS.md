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
