# XDB - Claude

XDB is an agent-first data layer. Model once, store anywhere. See ./README.md for details.

## TDD (strict)

Write tests BEFORE implementation. Run failing test, write minimum code to pass, confirm green. No exceptions.

- Use `github.com/stretchr/testify` (`require` for fatal, `assert` for non-fatal). Table-driven tests preferred.

## Build

Use `make` exclusively — never invoke `go test`, `go build`, `go vet`, `golangci-lint` directly. Always run `make check` before committing.

| Target               | Purpose                                |
| -------------------- | -------------------------------------- |
| `make setup`         | Setup project and update deps          |
| `make build`         | Build all packages                     |
| `make test`          | Run all tests (root + sub-modules)     |
| `make check`         | Linting and formatting (tidy + lint)   |
| `make lint`          | Run golangci-lint with auto-fix        |
| `make tidy`          | Run go mod tidy                        |
| `make coverage`      | Generate coverage report               |
| `make report`        | Generate and open HTML coverage report |
| `make services-up`   | Start compose services (podman)        |
| `make services-down` | Stop compose services                  |
| `make services-logs` | Tail compose service logs              |

## Go Style

Write vertical, readable code. Favor more lines over longer lines:

- Keep packages small and focused — no circular dependencies
- One argument per line for long function calls (trailing comma on last arg)
- Intermediate variables over deeply nested expressions
- Named booleans for long conditionals
- Early returns to keep logic flat
- Vertical struct literals (one field per line)

## Core Value Access

Use typed `As*` methods (`AsStr()`, `AsInt()`, `AsBool()`, etc.) on `core.Tuple` or `core.Value` — never `Unwrap()`. Prefer `record.Get("title").AsStr()` over `Tuple.Value().As*()`. Always check errors.

## Documentation

- GoDoc on all public functions/types. Use `[pkg.Type]` annotations. Use `doc.go` for package-level docs.
- **Concept docs** (`docs/concepts/`): one file per concept with YAML frontmatter (`title`, `description`, `package`). Create for new abstractions, update when APIs/behavior change. Include Go examples matching actual API. Update `docs/concepts/README.md` index.

## Project Structure

```
core/             # URI, Tuple, Record, Value, Type — the data model
schema/           # Schema definitions and validation
store/            # Store interfaces (RecordStore, SchemaStore, etc.)
  xdbfs/          # Filesystem-backed store
  xdbmemory/      # In-memory store (reference/testing)
  xdbredis/       # Redis-backed store (requires `make services-up`)
  xdbsqlite/      # SQLite-backed store
encoding/
  xdbjson/        # JSON encoder/decoder for records
tests/            # Shared test suites for store implementations
docs/
  concepts/       # Concept docs (one per concept)
  plans/          # Plans: YYYY-MM-DD-plan-name.md
  research/       # Research: YYYY-MM-DD-research-name.md
```

## Plans and Research

- Plans go in `./docs/plans/`, research in `./docs/research/`, both named `YYYY-MM-DD-name.md`.
- After user approves the plan, move the plan to `./docs/plans/` before implementation.
