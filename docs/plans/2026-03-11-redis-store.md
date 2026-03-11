# Redis Store Implementation Plan

**Date:** 2026-03-11
**Package:** `store/xdbredis`

## Overview

Implement a Redis-backed store using Redis Hash maps for records and RedisJSON for schemas. Records are stored as flat hashes where each attribute maps to a hash field, making individual field access efficient and aligning with Redis-native data patterns.

## Dependencies

- `github.com/redis/go-redis/v9` — official Go Redis client

## Key Design

All keys live under a configurable prefix (default `xdb`).

| Entity         | Key Pattern                          | Redis Type    |
| -------------- | ------------------------------------ | ------------- |
| Record         | `{prefix}:{ns}:{schema}:{id}`        | Hash          |
| Schema         | `{prefix}:{ns}:{schema}:_schema`     | String (JSON) |
| Record index   | `{prefix}:{ns}:{schema}:_idx`        | Set           |
| Schema index   | `{prefix}:{ns}:_idx`                 | Set           |
| Namespace index | `{prefix}:_idx`                     | Set           |

### Record Hash Layout

Each record is a Redis hash. Tuple attributes become hash fields, values are encoded as strings with a type prefix for lossless round-tripping:

```
HSET xdb:com.example:posts:post-1
  title       "s:Hello World"
  rating      "f:4.5"
  active      "b:true"
  created_at  "t:2026-03-11T00:00:00Z"
  tags        "a:[\"go\",\"redis\"]"
  raw         "x:base64data"
  count       "i:42"
  bignum      "u:999"
  meta        "j:{\"key\":\"val\"}"
```

Type prefixes: `s:` STRING, `i:` INTEGER, `u:` UNSIGNED, `f:` FLOAT, `b:` BOOLEAN, `t:` TIME, `x:` BYTES (base64), `a:` ARRAY (JSON), `j:` JSON.

This avoids needing a schema lookup on every read — the type is self-describing.

### Schema Storage

Schemas are stored as JSON strings (reusing `schema.MarshalJSON` / `schema.UnmarshalJSON`), keyed with a `_schema` suffix to coexist in the same keyspace.

### Index Sets

Rather than using `SCAN` (which is O(n) over the full keyspace), the store maintains Redis Sets as indexes that are updated atomically alongside every write:

- **Record index** (`{prefix}:{ns}:{schema}:_idx`) — contains record IDs for a given schema. Updated on create/upsert/delete.
- **Schema index** (`{prefix}:{ns}:_idx`) — contains schema names for a given namespace. Updated on schema create/delete.
- **Namespace index** (`{prefix}:_idx`) — contains namespace names. Updated on schema create/delete.

All index mutations happen in the same `MULTI/EXEC` transaction as the data write, so indexes are always consistent.

Listing operations use `SMEMBERS` to read the index, sort in-memory, then paginate — O(m) where m is the size of the set, not the full keyspace.

## Package Structure

```
store/xdbredis/
├── doc.go           # Package documentation
├── store.go         # Store struct, constructor, options
├── codec.go         # Value ↔ hash field encoding (type-prefixed strings)
├── codec_test.go    # Codec round-trip tests
├── record.go        # RecordReader + RecordWriter implementation
├── schema.go        # SchemaReader + SchemaWriter implementation
├── namespace.go     # NamespaceReader implementation
├── batch.go         # BatchExecutor via MULTI/EXEC
├── health.go        # HealthChecker via PING
└── store_test.go    # Shared test suites + integration tests
```

## Constructor & Options

```go
type Store struct {
    client redis.UniversalClient
    prefix string
}

type Option func(*Store)

func WithPrefix(prefix string) Option       // default: "xdb"

func New(client redis.UniversalClient, opts ...Option) *Store
```

Accept `redis.UniversalClient` so callers can pass a regular client, cluster client, or sentinel client.

## Implementation Plan

### Phase 1: Codec

Implement the hash field codec — encode `core.Value` to type-prefixed strings and decode back.

1. **Write codec tests** — round-trip every type (STRING, INTEGER, UNSIGNED, FLOAT, BOOLEAN, TIME, BYTES, ARRAY, JSON) through encode/decode.
2. **Implement codec** — `Encode(*core.Value) (string, error)` and `Decode(string) (*core.Value, error)`.

### Phase 2: Schema Store

Schemas are simpler (just JSON blobs) and records depend on schema validation, so implement schemas first.

1. **Write tests** using `tests.NewSchemaStoreSuite`.
2. **Implement `SchemaReader`:**
   - `GetSchema(ctx, uri)` — `GET {prefix}:{ns}:{schema}:_schema`, unmarshal JSON.
   - `ListSchemas(ctx, ns, offset, limit)` — `SMEMBERS {prefix}:{ns}:_idx`, sort, paginate.
3. **Implement `SchemaWriter`:**
   - `CreateSchema(ctx, def)` — In a `MULTI/EXEC`: `SET NX` schema key, `SADD` schema name to `{prefix}:{ns}:_idx`, `SADD` namespace to `{prefix}:_idx`. Return `ErrAlreadyExists` if key exists.
   - `UpdateSchema(ctx, def)` — Check existence, then `SET`. No index changes needed.
   - `DeleteSchema(ctx, uri)` — In a `MULTI/EXEC`: `DEL` schema key, `DEL` record index, `DEL` all record keys, `SREM` schema from namespace index. If namespace has no remaining schemas, `SREM` namespace from `{prefix}:_idx`.

### Phase 3: Record Store

1. **Write tests** using `tests.NewRecordStoreSuite`.
2. **Implement `RecordReader`:**
   - `GetRecord(ctx, uri)` — `HGETALL`, decode each field via codec, build `core.Record`.
   - `ListRecords(ctx, ns, schema, offset, limit)` — `SMEMBERS {prefix}:{ns}:{schema}:_idx`, sort IDs, paginate, then `HGETALL` each.
3. **Implement `RecordWriter`:**
   - `CreateRecord(ctx, record)` — Check existence with `EXISTS`, validate against schema (if strict/dynamic), then `MULTI/EXEC`: `HSET` all fields + `SADD` ID to `{prefix}:{ns}:{schema}:_idx`. Return `ErrAlreadyExists` if exists.
   - `UpdateRecord(ctx, record)` — Check existence, validate, then `MULTI/EXEC`: `DEL` + `HSET` (full replace). No index change needed. Return `ErrNotFound` if missing.
   - `UpsertRecord(ctx, record)` — Validate, then `MULTI/EXEC`: `DEL` + `HSET` + `SADD` ID to record index.
   - `DeleteRecord(ctx, uri)` — `MULTI/EXEC`: `DEL` record key + `SREM` ID from `{prefix}:{ns}:{schema}:_idx`. Return `ErrNotFound` if key didn't exist.

### Phase 4: Namespace Reader

1. **Write tests** using `tests.NewNamespaceStoreSuite`.
2. **Implement `NamespaceReader`:**
   - `ListNamespaces(ctx, offset, limit)` — `SMEMBERS {prefix}:_idx`, sort, paginate.
   - `GetNamespace(ctx, ns)` — `SISMEMBER {prefix}:_idx {ns}`, return `ErrNotFound` if not a member.

### Phase 5: Health Check & Batch

1. **Implement `HealthChecker`** — `PING` command, return error if fails.
2. **Write batch tests** using `tests.NewBatchSuite`.
3. **Implement `BatchExecutor`** — Use Redis `MULTI/EXEC` (pipeline) for atomic execution. Create a transactional store wrapper that buffers commands.

### Phase 6: Integration Testing

1. **Add build tag** `//go:build redis` for integration tests requiring a live Redis.
2. **Add Makefile target** `make test-redis` that starts Redis via podman and runs tagged tests.
3. **Run all shared test suites** against a real Redis instance.

## Key Design Decisions

| Decision            | Choice                | Rationale                                                                  |
| ------------------- | --------------------- | -------------------------------------------------------------------------- |
| Record storage      | Hash map              | Per-user request; enables per-field reads/writes; natural Redis pattern    |
| Value encoding      | Type-prefixed strings | Self-describing; no schema lookup needed on read; lossless round-trip      |
| Schema storage      | JSON string           | Reuses existing `schema` package marshal/unmarshal; schemas are read-whole |
| Listing             | Set indexes           | O(m) reads vs O(n) SCAN; indexes updated atomically with data writes      |
| Existence checks    | `EXISTS` before write | Needed for Create/Update semantics; WATCH+MULTI for race safety            |
| Transaction support | `MULTI/EXEC`          | Native Redis atomicity; maps to `BatchExecutor` interface                  |
| Client type         | `UniversalClient`     | Works with standalone, cluster, and sentinel deployments                   |
| Dot-notation attrs  | Stored flat in hash   | `address.street` is just a hash field name; no nesting in Redis hash       |

## Race Condition Handling

For Create (must not exist) and Update (must exist), use optimistic locking:

```
WATCH key
EXISTS key → check
MULTI
HSET key field1 val1 field2 val2 ...
EXEC
```

If `EXEC` returns nil (key was modified between WATCH and EXEC), retry or return an error.

## Risks & Mitigations

| Risk                                | Mitigation                                                        |
| ----------------------------------- | ----------------------------------------------------------------- |
| Index/data drift                    | All index mutations in same MULTI/EXEC as data writes             |
| Type prefix parsing errors          | Comprehensive codec tests; return clear errors for malformed data |
| Redis unavailability                | HealthChecker surfaces issues; context cancellation for timeouts  |
| Key collision with `_schema` suffix | Document reserved ID; reject record IDs equal to `_schema`        |
