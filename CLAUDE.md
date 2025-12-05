# CLAUDE.md

This file provides guidance for working with code in this codebase.

## Development Commands

### Build and Test

- `make test` - Run all tests with race detection and coverage across all modules
- `make check` - Run all code quality checks (formatting, vetting, linting)
- `make lint` - Run golangci-lint (includes formatting, vetting, and linting with auto-fix)
- `make tidy` - Run `go mod tidy` on all modules
- `make setup` - Initial project setup

### Coverage Reports

- `make coverage` - Generate coverage report
- `make report` - Generate HTML coverage report and open in browser

### Single Test Execution

- `go test ./core` - Run tests for core package
- `go test -run TestSpecificTest ./package` - Run specific test

### Running the Server

- `cd ./cmd/xdb && go run *.go server` - Start the XDB HTTP server
- The server runs on port 8080 by default
- Use Bruno collections in `bruno/` directory for testing API endpoints

## Project Structure

```
xdb/
├── api/                    # HTTP API layer (see api/README.md)
├── cmd/xdb/               # CLI application (see cmd/xdb/README.md)
│   ├── app/               # Application commands
│   └── main.go            # Entry point
├── codec/                 # Low-level serialization (see codec/README.md)
├── core/                  # Core data model
├── driver/                # Database backends (see driver/README.md)
├── encoding/              # Format conversions (see encoding/README.md)
├── x/                     # Utility functions
├── tests/                 # Shared test suites
├── examples/              # Example applications
```

## Core Architecture

XDB is a tuple-based database abstraction library with a layered architecture.

### Core Data Model (`core/`)

- **Tuple**: Fundamental building block (ID + Attribute + Value)
- **Record**: Collection of tuples with the same ID
- **Value**: Typed container supporting basic types, arrays, and maps
- **URI**: Resource identifiers (`xdb://NS[/SCHEMA][/ID][#ATTRIBUTE]`)
- **Type**: Type system with scalar, array, and map types
- **NS**: Namespace for organizing schemas
- **Schema**: Structure and validation rules for records

## Multi-Module Structure

XDB uses a multi-module structure where some packages have their own `go.mod` files.

The Makefile automatically discovers and runs commands across all modules (via `go.mod` discovery), ensuring consistent formatting, testing, and linting across the entire codebase.

## Package Guidelines

### File Organization

- Each type or closely related set of types should have its own file
- Test files use `_test.go` suffix and are co-located with source
- Example functions go in `examples_test.go` and are named `ExampleXxx`
- Package documentation goes in `doc.go`

## Code Style Requirements

### Comments

- Only write GoDoc comments for exported types, functions, methods, constants, and variables
- Do not write inline comments or non-GoDoc comments
- Code should be self-explanatory; if it needs comments, consider refactoring

### Import Organization

Group imports with blank lines between groups:

1. Standard library
2. Third-party packages
3. Local packages (github.com/xdb-dev/xdb/...)

### Error Handling

- Return errors as the last return value
- Use `errors.Wrap` for error wrapping with context
- Only add context that callers need to know about
- Return early on errors using `if` statements
- Do not use generic error names; be descriptive

### Naming Conventions

- Use CamelCase for exported names, mixedCaps for unexported
- Capitalize acronyms (ID, URL, API)
- Avoid underscores in names
- Use short, meaningful receiver names

### Testing

- Use table-driven tests for functions with multiple cases
- Place test functions in `package_test` form when testing public APIs
- Use `testify/assert` for assertions
- Example tests must have exact `// Output:` comments
- The `tests/` package provides shared test suites for driver implementations:
  - `TupleDriverTest`: Test suite for `TupleDriver` implementations
  - `RecordDriverTest`: Test suite for `RecordDriver` implementations
  - `SchemaDriverTest`: Test suite for `SchemaDriver` implementations
- Use these test suites to ensure consistent behavior across driver implementations

## Important Notes

- All exported functions and types must have GoDoc comments
- Use `context.Context` as the first parameter for functions that need it
- Linting is enforced via `golangci-lint` with configuration in `.golangci.yml`
- Make sure to keep GoDoc comments and respective package's `doc.go` up to date
- Development tools (golangci-lint, gocov, etc.) are managed via `tools.mod` and installed using `go get -modfile=tools.mod -tool`
