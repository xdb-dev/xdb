# XDB - Claude

XDB is an agent-first data layer. Model once, store anywhere. See ./README.md for details.

## TDD (strict)

Write the tests before the implementation. Run the failing test. Write the minimum code to make it pass. Make sure that the test is green.

- Use `github.com/stretchr/testify` (`require` for fatal assertions, `assert` for non-fatal assertions). Prefer table-driven tests.

## Build

Use `make` for every build, test, and lint step. The Makefile pins the tool versions and the lint config, so a direct `go test`, `go build`, `go vet`, or `golangci-lint` call gives different results. Before you commit, run `make check`.

| Target               | Purpose                                                  |
| -------------------- | -------------------------------------------------------- |
| `make setup`         | Prepare the project and update dependencies              |
| `make build`         | Type-check all packages (writes no binary)               |
| `make install`       | Install the `xdb` binary with `go install`               |
| `make test`          | Run all tests (root and sub-modules)                     |
| `make bench`         | Run all benchmarks                                       |
| `make check`         | Tidy and lint (`make tidy` + `make lint`)                |
| `make lint`          | Run golangci-lint with auto-fix                          |
| `make tidy`          | Run go mod tidy                                          |
| `make coverage`      | Generate the coverage report                             |
| `make report`        | Generate and open the HTML coverage report               |
| `make services-up`   | Start the service containers (Apple container)           |
| `make services-down` | Stop the service containers                              |
| `make services-logs` | Tail the service container logs                          |

The e2e orchestrator (`.claude/commands/xdb-e2e.md`) is the one exception to the `make` rule: it builds `bin/xdb` with `cd cmd/xdb && go build -o ../../bin/xdb .`, because `make build` only type-checks and writes no binary. The e2e sub-agents (`tests/e2e/RUNBOOK.md`) do not build.

## Go Style

Write vertical, readable code. Prefer more lines to longer lines:

- Keep packages small and focused. Do not create circular dependencies.
- For a long function call, put one argument per line, with a trailing comma on the last argument.
- Use intermediate variables instead of deeply nested expressions.
- Use named booleans for long conditionals.
- Use early returns to keep the logic flat.
- Write struct literals vertically, one field per line.

## Core Value Access

Use the typed `As*` methods (`AsStr()`, `AsInt()`, `AsBool()`, and the others) on `core.Tuple` or `core.Value`. Never use `Unwrap()`. Prefer `record.Get("title").AsStr()` to `Tuple.Value().As*()`.

## Documentation

- Write GoDoc for all public functions and types. Use `[pkg.Type]` links. Put package-level docs in `doc.go`.
- **Concept docs** (`docs/concepts/`): one file per concept, with YAML frontmatter (`title`, `description`, `package`). When you add an abstraction, create a concept doc. When an API or a behavior changes, update the concept doc. Include Go examples that match the actual API. Update the index in `docs/concepts/README.md`.

## Project Structure

```
api/                # Transport-agnostic services: records, schemas, namespaces, batch, watch
  catalog/          # Source of truth for JSON-RPC methods and types (behind `xdb describe`)
cmd/xdb/            # The `xdb` binary
  cli/              # CLI commands, CONTEXT.md, and the embedded skills
core/               # URI, Tuple, Record, Value, Type: the data model
schema/             # Definitions and validation
store/              # Store facade, middleware, and driver interfaces
  xdbfs/            # Filesystem driver
  xdbmemory/        # In-memory driver (reference/testing)
  xdbredis/         # Redis driver [module] (requires `make services-up`)
  xdbsqlite/        # SQLite driver [module]
encoding/           # Format adapters: one package per external format
  xdbjson/          # JSON Schema import; JSON record encode/decode
  xdbproto/         # Protobuf message import; proto record encode/decode [module]
  xdbstruct/        # Go struct import; struct record encode/decode
filter/             # CEL filter parsing and evaluation
  sqlgen/           # Compiled CEL filter to parameterized SQL
rpc/                # JSON-RPC 2.0 server
  client/           # JSON-RPC 2.0 client used by the CLI
x/                  # Generic helpers: grouping, mapping, filtering
tests/              # Shared conformance suites for drivers
  e2e/              # Agent-facing end-to-end scenarios and runbook
docs/
  concepts/         # Concept docs (one per concept)
  plans/            # Plans: YYYY-MM-DD-plan-name.md
  research/         # Research: YYYY-MM-DD-research-name.md
```

`[module]` marks a directory with its own `go.mod`, alongside `cmd/xdb`. Each one exists to keep a heavy driver or codec dependency out of the root module, so it is a separate module only when the dependency justifies it. `make` targets discover these automatically; nothing needs registering.

## Plans and Research

- Plan drafts live in `~/.claude/plans/`. After the user approves a plan, move it to `./docs/plans/` before implementation.
- Research goes in `./docs/research/`. Name plans and research `YYYY-MM-DD-name.md`.
