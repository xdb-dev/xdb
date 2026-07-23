---
title: Versioning
description: Per-record system metadata — _id, _version, _updated — and the optimistic-concurrency contract they carry from storage to the CLI.
package: store, schema, api
---

# Versioning

Every record in XDB carries three **system fields**:

| Field | Type | Meaning |
|-------|------|---------|
| `_id` | STRING | The record's id, projected from its path |
| `_version` | INTEGER | Revision counter: 1 on create, +1 on every write |
| `_updated` | TIME | Timestamp of the record's last write |

They are always present, on every backend, for schema'd and schema-less
records alike. Reading a record returns them; writing one back uses
`_version` as an optimistic-concurrency precondition.

```go
record, _ := st.GetRecord(ctx, uri)

version, _ := record.Get("_version").AsInt()   // 3
id, _ := record.Get("_id").AsStr()             // "post-1"
updated, _ := record.Get("_updated").AsTime()
```

## Safe read-modify-write, by default

The version travels **inside the record**, so a read-edit-write round trip
is protected without the caller asking for it:

```go
record, _ := st.GetRecord(ctx, uri)   // carries _version: 3
record.Set("title", "edited")
err := st.UpsertRecord(ctx, record)   // succeeds only if still at 3
```

If another writer landed in between, the store returns
[`core.ErrConflict`](../../core/errors.go) and **nothing is written**. To
overwrite unconditionally, build a fresh record (or drop the attr) — a
write with no `_version` always wins:

```go
fresh := core.NewRecord("app", "posts", "post-1").Set("title", "forced")
st.UpsertRecord(ctx, fresh)           // unconditional
```

This is the reason the fields flow all the way up: lost-update protection
is the default, and blind overwrite is the thing you opt into.

## Which fields you can write

| Field | On write |
|-------|----------|
| `_version` | Precondition. Matching bumps, stale conflicts, absent writes unconditionally |
| `_updated` | Ignored — stamped by the store |
| `_id` | Ignored when it matches the URI; `ErrSchemaViolation` when it disagrees |

Echoing back a record you just read is the normal path, so derived fields
are dropped rather than rejected. An `_id` that disagrees with the URI is
not an echo but a misaddressed write, so it is refused. Deleting a system
attr is refused too.

Reserved names are enforced at the schema level: a definition may not
declare a top-level field starting with `_` (`schema.ErrInvalidField`).
The Items of an object-array field are a separate namespace stored inside
a JSON value, so `_id` is legal there — external documents routinely
carry one.

## Filtering

System fields are queryable like any other, including through SQLite's
filter pushdown:

```
_version > 5
_updated > timestamp("2026-01-01T00:00:00Z")
_id.startsWith("user-")
```

`_version` and `_updated` are declared fields, so they are real columns in
a column-table backend. `_id` is never stored — it is the record's
addressing key in every backend (SQLite's `_id` column, xdbfs's filename,
xdbredis's key suffix), so it is projected on read and resolved to that
key when a filter is pushed down.

## How it works

Versioning is **driver middleware**, installed unconditionally by
`store.New`:

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

- **`_version` and `_updated` are stored fields.** Enforcement stamps them
  into every schema definition, so a column-table backend materializes real
  columns and a KV backend stores them like any attr. The versioning
  middleware then writes them as ordinary tuples in the *same* mutation as
  the record's own data — one atomic write on every backend, not two.
- **`_id` is virtual.** Storing it would duplicate the path, so the facade
  projects it from the record's URI on read.

No driver changed to support any of this. A new backend inherits
versioning the way it inherits validation.

### Definitions written before versioning

A definition stored before system fields existed is upgraded on first use:
enforcement stamps it and writes it back, exactly like a dynamic-mode
evolution. On SQLite that issues `ALTER TABLE ADD COLUMN`. There is no
offline migration step.

### Concurrency

On backends with native transactions (memory, SQLite) the read, the CAS,
and the write share one transaction, so the check cannot go stale. On
`xdbfs` and `xdbredis` the read-modify-write has the same window the
enforcement middleware's merge-that-creates check already has — a
documented property of non-transactional backends, not a new one.

## At the API and CLI

The fields are ordinary JSON keys, so they travel without new request or
response types:

```console
$ xdb records get xdb://app/posts/post-1 -o json
{"_id":"post-1","_ns":"app","_schema":"posts","_version":3,
 "_updated":"2026-07-23T11:48:52+05:30","title":"Hello"}
```

Every write response reports the version the store just stamped, so a
client never re-reads to learn what it wrote. The table view omits
`_updated` for readability; every machine-readable format keeps it.

**Delete** is the one verb with no payload to carry a precondition, so it
takes one explicitly:

```console
$ xdb records delete xdb://app/posts/post-1 --force --if-version 3
```

A mismatch fails with `CONFLICT` and leaves the record alone.

**Watch events** carry the version as a top-level field:

```json
{"ts":"...","type":"record.update","uri":"xdb://app/posts/post-1","version":4}
```

Delete events have no payload to read it from, and the bus is
deliberately lossy — a subscriber that falls behind misses events
silently. Versions make that loss *detectable*: a jump from 3 to 7 means
writes were dropped and the record should be re-read.

## Errors

| Error | When |
|-------|------|
| `core.ErrConflict` | `_version` precondition did not match the stored version |
| `core.ErrSchemaViolation` | `_id` disagrees with the URI, a system attr was deleted, or a definition declared a `_`-prefixed field |

Over RPC, a conflict is code `-32003`; the CLI reports `CONFLICT`. Re-read
the record and retry.

## Related Concepts

- [Records](records.md) — What system fields are attached to
- [Stores](stores.md) — The facade and middleware stack
- [Drivers](drivers.md) — Why drivers needed no changes
- [Schemas](schemas.md) — Reserved field names and definition stamping
- [Filters](filters.md) — Querying system fields
