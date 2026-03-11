# XDB - Claude

## Overview

XDB is a Go library that provides a tuple-based abstraction for modeling, storing, and querying data across multiple databases. Rather than writing database-specific schemas, queries, and migrations, XDB allows developers to model their domain once and use it with one or more databases.

The `xdb` CLI is a S3 like CLI that allows creating, listing, getting, and deleting data from the XDB store.

Read ./README.md for more details.

## Tech Stack

- Go (latest stable)

## Code Quality Expectations

### Test Driven Development

- Always write tests BEFORE writing implementation code.
- Run the failing test to confirm it fails for the right reason.
- Write the minimum implementation to make the test pass.
- Run tests again to confirm they pass.
- Only commit when tests are passing.
- Do not skip this cycle. No exceptions.
- Use github.com/stretchr/testify for assertions (require for fatal, assert for non-fatal)
- Table-driven tests preferred.

### Build Commands

- Use `make` as the single entry point for all build, test, check commands.
- Do not invoke `go test`, `go build`, `go vet`, `golangci-lint`, or other tools directly — use the corresponding Makefile target.
- If a new build/dev command is needed, add a Makefile target for it first.

### Go Style

Write vertical, readable Go code. Favor more lines over longer lines:

- Break long function calls with one argument per line (trailing comma on last arg)
- Use intermediate variables instead of deeply nested expressions — the compiler inlines them
- Break long conditionals into named booleans
- Use early returns to keep logic flat and reduce nesting
- Write struct literals vertically with one field per line

### Core Value Access

- Use typed `As*` methods (`AsStr()`, `AsInt()`, `AsBool()`, etc.) on `core.Tuple` or `core.Value` — never use `Unwrap()`.
- Prefer `Tuple.As*()` (e.g., `record.Get("title").AsStr()`) over `Tuple.Value().As*()`.
- Always check the error returned by `As*` methods.

### Documentation

- All public functions and types should be documented with GoDoc.
- Use simple, concise, and clear language.
- Use doc.go files for detailed package-level documentation.
- Use `[pkg.Type]` annotations for relevant types in GoDoc comments.

## Makefile Targets

| Target           | Purpose                                  |
| ---------------- | ---------------------------------------- |
| `make setup`     | Setup the project and update deps        |
| `make build`     | Build all packages                       |
| `make test`      | Run all tests (with race + coverage)     |
| `make check`     | Run linting and formatting (tidy + lint) |
| `make lint`      | Run golangci-lint with auto-fix          |
| `make tidy`      | Run go mod tidy                          |
| `make coverage`  | Generate coverage report                 |
| `make report`    | Generate and open HTML coverage report   |

## Project Structure

```
core/             # URI, Tuple, Record, Value, Type — the data model
schema/           # Schema definitions and validation
store/            # Store interfaces (RecordStore, SchemaStore, etc.)
  xdbfs/          # Filesystem-backed store implementation
  xdbmemory/      # In-memory store (reference/testing)
  xdbredis/       # Redis-backed store using RedisJSON (planned)
encoding/
  xdbjson/        # JSON encoder/decoder for records
types/            # Type codec for database backend mappings
tests/            # Shared test suites for store implementations
docs/
  plans/          # Implementation plans (YYYY-MM-DD-name.md)
  research/       # Research documents (YYYY-MM-DD-name.md)
```

## Development Rules

- Keep packages small and focused — no circular dependencies.
- Always run `make check` before committing.

## Research & Plans

- All plans MUST be tracked in the `./docs/plans` directory.
- All plans MUST be named like `YYYY-MM-DD-plan-name.md`.
- All research MUST be tracked in the `./docs/research` directory.
- All research MUST be named like `YYYY-MM-DD-research-name.md`.
