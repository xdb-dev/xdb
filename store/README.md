# XDB Stores

Database backend implementations.

```
store/
├── store.go           # Core interfaces (TupleStore, RecordStore, SchemaStore)
├── xdbmemory/         # In-memory store (reference implementation)
├── xdbsqlite/         # SQLite store
├── xdbredis/          # Redis store
└── xdbfs/             # Filesystem store
```
