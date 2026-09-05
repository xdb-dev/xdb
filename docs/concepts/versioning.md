---
title: Versioning
description: Per-record system metadata (_id, _version, _updated) and the optimistic-concurrency contract they carry from storage to the CLI.
package: store, schema, api
---

# Versioning

Every record in XDB carries three **system fields**:

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

## Safe read-modify-write, by default

The version travels **inside the record**, so a read-edit-write round trip is protected without any request from the caller:

```go
record, _ := st.GetRecord(ctx, uri)   // carries _version: 3
record.Set("title", "edited")
err := st.UpsertRecord(ctx, record)   // succeeds only if still at 3
```

If another writer landed in between, the store returns `core.ErrConflict` and **nothing is written**. To overwrite unconditionally, build a fresh record, or remove the attribute. A write with no `_version` always wins:

```go
fresh := core.NewRecord("app", "posts", "post-1").Set("title", "forced")
st.UpsertRecord(ctx, fresh)           // unconditional
```

This is the reason the fields flow all the way up. Lost-update protection is the default, and blind overwrite is the option that you opt into.

## Which fields you can write

| Field | On write |
|-------|----------|
| `_version` | Precondition. A matching value bumps the version, a stale value conflicts, and an absent value (or 0) writes unconditionally |
| `_updated` | Ignored. The store stamps it |
| `_id` | Ignored when it matches the URI. `ErrSchemaViolation` when it disagrees |

The echo of a record that you just read is the normal path, so derived fields are removed, not rejected. An `_id` that disagrees with the URI is not an echo but a misaddressed write, so it is refused. A delete of a system attribute is refused too.

Reserved names are enforced at the schema level. A definition cannot declare a top-level field that starts with `_` (`schema.ErrInvalidField`). The Items of an object-array field are a separate namespace stored inside a JSON value, so `_id` is legal there. External documents routinely carry one.

## Filtering

System fields are queryable like any other field, including through the filter pushdown of SQLite:

```
_version > 5
_updated > timestamp("2026-01-01T00:00:00Z")
_id.startsWith("user-")
```

`_version` and `_updated` are declared fields, so they are real columns in a column-table backend. `_id` is never stored. It is the addressing key of the record in every backend: the `_id` column in SQLite, the filename in xdbfs, the key suffix in xdbredis. The facade projects it on read, and resolves it to that key when a filter is pushed down.

## How it works

Versioning is **driver middleware**. `store.New` installs it unconditionally:

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

Two mechanisms, deliberately different:

- **`_version` and `_updated` are stored fields.** Enforcement stamps them into every definition. A column-table backend materializes real columns, and a KV backend stores them like any attribute. The versioning middleware then writes them as ordinary tuples, in the *same* mutation as the own data of the record. This is one atomic write on every backend, not two.
- **`_id` is virtual.** A stored `_id` is only a copy of the path. The facade projects it from the URI of the record on read.

Drivers contain no versioning code. A new driver inherits versioning the way it inherits validation.

### Definitions written before versioning

A definition stored before system fields existed is upgraded on first use. Enforcement stamps it and writes it back, exactly like a dynamic-mode evolution. On SQLite this issues `ALTER TABLE ADD COLUMN`. There is no offline migration step.

### Concurrency

On backends with native transactions (memory, SQLite), the read, the CAS, and the write share one transaction, so the check cannot go stale. On `xdbfs` and `xdbredis`, the read-modify-write has the same window as the patch-that-creates check in the enforcement middleware. This is a documented property of non-transactional backends, not a new one.

## At the API and CLI

The fields are ordinary JSON keys, so they travel without new request or response types:

```console
$ xdb records get xdb://app/posts/post-1 -o json
{"_id":"post-1","_ns":"app","_schema":"posts","_version":3,
 "_updated":"2026-07-23T11:48:52+05:30","title":"Hello"}
```

Every write response reports the version that the store just stamped, so a client never re-reads to learn what it wrote. The table view omits `_updated` for readability. Every machine-readable format keeps it.

**Delete** is the one verb with no payload to carry a precondition, so it takes one explicitly:

```console
$ xdb records delete xdb://app/posts/post-1 --force --if-version 3
```

A mismatch fails with `CONFLICT` and leaves the record unchanged.

**Watch events** carry the version as a top-level field:

```json
{"ts":"...","type":"record.update","uri":"xdb://app/posts/post-1","version":4}
```

A delete event carries the version that the record held before the delete. The bus is deliberately lossy: a subscriber that falls behind misses events silently. Versions make that loss *detectable*. A jump from 3 to 7 means that writes were dropped, and the client must re-read the record.

## Errors

| Error | When |
|-------|------|
| `core.ErrConflict` | The `_version` precondition did not match the stored version |
| `core.ErrSchemaViolation` | `_id` disagrees with the URI, a system attribute was deleted, or a definition declared a `_`-prefixed field |

Over RPC, a conflict is code `-32003`. The CLI reports `CONFLICT`. To recover, the client re-reads the record and retries.

## Related Concepts

- [Records](records.md) — What system fields are attached to
- [Stores](stores.md) — The facade and middleware stack
- [Drivers](drivers.md) — Why drivers contain no versioning code
- [Schemas](schemas.md) — Reserved field names and definition stamping
- [Filters](filters.md) — Querying system fields
