# XDB Stores

Stores bridge XDB's tuple model to specific database backends.

## Interfaces

Stores implement capability interfaces based on their features:

- **TupleStore**: `TupleReader` and `TupleWriter` for tuple-level operations
- **RecordStore**: `RecordReader` and `RecordWriter` for record-level operations
- **SchemaStore**: `SchemaReader` and `SchemaWriter` for schema management

Not all stores need to support all interfaces. Implement only the capabilities your backend supports.

## Implementation Guide

### Structure

Each store lives in its own subdirectory under `store/`:

```
store/
├── driver.go           # Core interfaces
├── xdbmemory/         # In-memory store (reference implementation)
├── xdbsqlite/         # SQLite store
├── xdbredis/          # Redis store
└── xdbfs/             # Filesystem store
```

### Testing

Use the shared test suites from `tests/` package:

```go
func TestStore(t *testing.T) {
    store := setupStore(t)
    tests.TupleStoreTest(t, store)
    tests.RecordStoreTest(t, store)
    tests.SchemaStoreTest(t, store)
}
```

## Reference Implementation

Use `xdbmemory/` as a reference for implementing new stores. It provides a simple, complete implementation of all interfaces.
