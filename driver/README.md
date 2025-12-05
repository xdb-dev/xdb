# XDB Drivers

Drivers bridge XDB's tuple model to specific database backends.

## Interfaces

Drivers implement capability interfaces based on their features:

- **TupleDriver**: `TupleReader` and `TupleWriter` for tuple-level operations
- **RecordDriver**: `RecordReader` and `RecordWriter` for record-level operations
- **SchemaDriver**: `SchemaReader` and `SchemaWriter` for schema management

Not all drivers need to support all interfaces. Implement only the capabilities your backend supports.

## Implementation Guide

### Structure

Each driver lives in its own subdirectory under `driver/`:

```
driver/
├── driver.go           # Core interfaces
├── xdbmemory/         # In-memory driver (reference implementation)
├── xdbsqlite/         # SQLite driver
├── xdbredis/          # Redis driver
└── xdbfs/             # Filesystem driver
```

### Testing

Use the shared test suites from `tests/` package:

```go
func TestDriver(t *testing.T) {
    driver := setupDriver(t)
    tests.TupleDriverTest(t, driver)
    tests.RecordDriverTest(t, driver)
    tests.SchemaDriverTest(t, driver)
}
```

## Reference Implementation

Use `xdbmemory/` as a reference for implementing new drivers. It provides a simple, complete implementation of all interfaces.
