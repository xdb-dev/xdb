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

### Documentation

- All public functions and types should be documented with GoDoc.
- Use simple, concise, and clear language.
- Use doc.go files for detailed package-level documentation.
- Use `[pkg.Type]` annotations for relevant types in GoDoc comments.

## Makefile Targets

| Target       | Purpose                    |
| ------------ | -------------------------- |
| `make test`  | Run all tests              |
| `make check` | Run linting and formatting |
| `make tidy`  | Run go mod tidy            |
| `make build` | Run go build for CLI       |

## Project Structure

## Development Rules

- Keep packages small and focused — no circular dependencies.
- Always run `make check` before committing.
