---
title: Versioning
description: Per-record system metadata (_id, _version, _updated) and the optimistic-concurrency contract they carry from storage to the CLI.
package: store, schema, api
---

# Versioning

XDB adds system fields to every stored record:

| Field | Type | Meaning |
|-------|------|---------|
| `_id` | STRING | The id of the record, projected from its path |
| `_version` | INTEGER | Revision counter: 1 on create, +1 on every write |
| `_updated` | TIME | Timestamp of the last write of the record |

They are always present, on every backend, for records with a schema and for schema-free records alike. A read returns them. A write uses `_version` as an optimistic-concurrency precondition.

```go
record, _ := st.GetRecord(ctx, uri)

version, _ := record.Get("_version").AsInt()   // 3
id, _ := record.Get("_id").AsStr()             // "post-1"
updated, _ := record.Get("_updated").AsTime()
```

## Safe Read-Modify-Write, by Default

A record read includes `_version`. Writing that record back uses its version as a precondition:

```go
record, _ := st.GetRecord(ctx, uri)   // carries _version: 3
record.Set("title", "edited")
err := st.UpsertRecord(ctx, record)   // succeeds only if still at 3
```

If the stored version has changed, the store returns `core.ErrConflict` without writing. To omit the version check, build a fresh record or remove `_version`:

```go
fresh := core.NewRecord("app", "posts", "post-1").Set("title", "forced")
st.UpsertRecord(ctx, fresh)           // unconditional
```

The check and write are atomic on memory and SQLite. See [Concurrency](#concurrency) for filesystem and Redis limitations.

## System Fields on Write

| Field | On write |
|-------|----------|
| `_version` | Precondition. A matching value bumps the version, a stale value conflicts, and an absent value (or 0) writes unconditionally |
| `_updated` | Ignored. The store stamps it |
| `_id` | Ignored when it matches the URI. `ErrSchemaViolation` when it disagrees |

You can write a record back with the system fields returned by a read. The store removes derived fields before writing, rejects an `_id` that disagrees with the URI, and rejects deletion of a system attribute.

Reserved names are enforced at the schema level. A definition cannot declare a top-level field that starts with `_` (`schema.ErrInvalidField`). The Items of an object-array field are a separate namespace stored inside a JSON value, so `_id` is legal there.

## Filtering

System fields are queryable like any other field, including through the filter pushdown of SQLite:

```
_version > 5
_updated > timestamp("2026-01-01T00:00:00Z")
_id.startsWith("user-")
```

`_version` and `_updated` are declared fields, so they are real columns in a column-table backend. `_id` is never stored. It is the addressing key of the record in every backend: the `_id` column in SQLite, the filename in xdbfs, the key suffix in xdbredis. The facade projects it on read, and resolves it to that key when a filter is pushed down.

## Middleware

Versioning is driver middleware. `store.New` installs it unconditionally:

```
        Store facade          projects _id from the record path
  ┌──────────────────────┐
  │  logging             │  observes                     (opt-in)
  │  schema enforcement  │  stamps _version/_updated onto every Def,
  │                      │  normalizes derived attrs     (ALWAYS)
  │  schema cache        │  accelerates                  (opt-in)
  │  versioning          │  CAS + stamps system tuples   (ALWAYS)
  └──────────────────────┘
        Driver                pure storage — knows nothing about versions
```

The fields are read and stored as follows:

- `_version` and `_updated` are stored fields. Enforcement stamps them into every definition. A column-table backend materializes real columns, and a KV backend stores them like any attribute. The versioning middleware then writes them as ordinary tuples, in the same atomic mutation as the user data.

- `_id` is virtual.  The facade projects it from the URI of the record on read.

Drivers contain no versioning code. A new driver inherits versioning the way it inherits validation.

### Definitions Written Before Versioning

A definition stored before system fields existed is upgraded on first use. Enforcement stamps it and writes it back, exactly like a dynamic-mode evolution. On SQLite this issues `ALTER TABLE ADD COLUMN`. There is no offline migration step.

### Concurrency

On memory and SQLite, the version read, comparison, and write share a transaction. On `xdbfs` and `xdbredis`, another writer can change the record between the version read and the write. Version checks on those backends do not guarantee protection from concurrent overwrites.

## At the API and CLI

The API and CLI return system fields as JSON keys:

```console
$ xdb records get xdb://app/posts/post-1 -o json
{"_id":"post-1","_ns":"app","_schema":"posts","_version":3,
 "_updated":"2026-07-23T11:48:52+05:30","title":"Hello"}
```

Write responses include the resulting version, so clients can use it without another read. The table view omits `_updated` for readability. Every machine-readable format keeps it.

Delete is the one verb with no payload to carry a precondition, so it takes one explicitly:

```console
$ xdb records delete xdb://app/posts/post-1 --force --if-version 3
```

A mismatch fails with `CONFLICT` and leaves the record unchanged. The error carries the `expected`, `got`, and `fix` tags, the same as a stale write, so the CLI hint says to re-read the record.

Watch events carry the version as a top-level field:

```json
{"ts":"...","type":"record.update","uri":"xdb://app/posts/post-1","version":4}
```

A delete event carries the version that the record held before the delete. The bus drops events when a subscriber falls behind. A jump from version 3 to 7 for the same record reveals missed writes; the client must re-read it.

## Errors

| Error | When |
|-------|------|
| `core.ErrConflict` | The `_version` precondition did not match the stored version |
| `core.ErrSchemaViolation` | `_id` disagrees with the URI, a system attribute was deleted, or a definition declared a `_`-prefixed field |

Over RPC, a conflict is code `-32003`. The CLI reports `CONFLICT`. To recover, the client re-reads the record and retries.

## Related Concepts

- [Records](records.md): What system fields are attached to

- [Stores](stores.md): The facade and middleware stack

- [Drivers](drivers.md): Why drivers contain no versioning code

- [Schemas](schemas.md): Reserved field names and definition stamping

- [Filters](filters.md): Querying system fields
