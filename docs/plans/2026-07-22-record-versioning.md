# Per-Record Versioning via System Fields

Status: design proposal (v2: supersedes the sidecar design, kept below as
a rejected alternative)
Date: 2026-07-22

## Goal

Give every record system metadata, mirroring what schemas already have
(`Def.Revision` + CAS):

- `_version`: monotonically increasing counter, bumped on every successful
  write (create = 1).

- `_updated`: timestamp of the last successful write.

- `_id`: the record's id, surfaced as a readable/filterable attribute.

- Writers may pass an expected `_version`; a mismatch fails with
  `core.ErrConflict` (optimistic concurrency, same contract as
  `UpdateSchema`).

## Changes from V1

The v1 sidecar design avoided a SQLite limitation: the column-table engine drops tuples whose attributes are not declared columns. `valuesFromTuples` iterates `sortedColumns(def)`, so an undeclared `_version` tuple could not survive writes to strict or dynamic schemas.

Declaring system fields in every definition lets SQLite create their columns through the existing DDL path. The middleware can then include version tuples in the record mutation:

| | Sidecar (v1) | System fields (v2) |
| --- | --- | --- |
| Record write + version stamp | two `Apply` calls: atomic only on TxDrivers | one `Apply`: atomic on every backend (put/create/patch) |
| Version visible in reads | `GetRecord`/`GetTuple` only (pushdown bypasses middleware) | all reads, uniformly: it is a real attr/column |
| Filterable (`_updated > X`, `_version == N`) | no | yes, including sqlite SQL pushdown |
| Delete cleanup | sidecar sweep on delete + `DropRecords` | free: version dies with the record |
| Reserved names needed | reserved schema prefix `_v_*` | reserved field prefix `_` |
| Affected code | store package only | schema + store + encoding/CLI expectations + migration |

The proposed v2 design adds filterable metadata and atomic data-and-version writes. It requires changes across schema handling, storage middleware, encoding, and CLI output.

## Existing `_id` Column

Both sqlite engines already use `_id` as the physical primary-key column
(`internal/sql/ddl.go`). A user def declaring a field named `_id` today
generates `_id TEXT NOT NULL` twice in `CREATE TABLE`: broken DDL. The
`_`-prefix field reservation below is therefore a latent-bug fix independent
of versioning.

The existing `_id` column already holds `path.ID()`. The virtual `_id` field exposes that value.

## Design

### Fixed Names, Always On

- Reserve `_version`, `_updated`, and `_id` as fixed names. Clients must be able to use the same field names in filters and scripts across namespaces.
- Rename fields at the encoding boundary. `xdbjson.WithIDField("userId")` changes the wire name while storage and filters retain `_id`. Proposed `WithVersionField` and `WithUpdatedField` options would follow this pattern.
- Reserve the `_` prefix for system fields. It matches the encoder's existing `_id`, `_ns`, and `_schema` defaults and avoids taking user names such as `rid`. `_id` also matches the identity key already used by each backend.
- Install versioning on every store, as with enforcement. Put and patch operations need one extra point read of `path#_version`; create needs none. Transactional drivers perform the read within the transaction. A future per-schema flag could disable stamping if needed; it would not rename fields.

### Stored Fields and Projected Identity

- `_version`, `_updated` are stored fields: stamped into every def
  (INTEGER / TIME, not Required), written as ordinary tuples by the
  middleware. Column tables get real columns; KV/flexible/schema-less
  storage stores them like any attr.

- `_id` is virtual. The facade derives it from the record path during assembly. Each backend already stores the ID as a key: SQLite's `_id` primary-key column, the filesystem filename, or the Redis key suffix. Projecting it adds no stored tuple and keeps it consistent with the path.

  Under `ColumnStrategy`, exempt `_id` from the `def.Fields` check in `walkIdent`; SQL generation already emits the correct column name. Under `KVStrategy`, map `_id` to the table's `_id` column. Ordinary attributes use `_attr` lookups, which cannot resolve a virtual field.

### Def Stamping (Enforce Middleware)

- `CreateSchema`: after `Validate`, append `_version` (INTEGER) and
  `_updated` (TIME) to `Fields`, then stamp `Revision = 1` as today.

- `PutSchema`: strip any incoming system fields, re-stamp canonically before
  `ValidateUpdate`: user tampering (type change, Required, deletion) is
  unrepresentable rather than rejected case-by-case.

- `Def.Validate`: reject user-declared `_`-prefixed field names (also fixes
  the `_id` DDL collision). Schema import strips system fields on ingest;
  export includes them so clients can inspect the stored definition.

- Lazy upgrade for pre-existing defs: when enforcement loads a def
  lacking system fields, stamp and `PutSchema` the upgraded def (same
  write-back pattern as dynamic-mode evolution). sqlite's `evolve` then
  issues `ALTER TABLE ADD COLUMN`: no offline migration step.

### Versioning Middleware (Write Path)

New driver middleware, installed unconditionally by `store.New` below
`enforce`, above the raw driver: it needs no schema reads at all (fixed
names), and stamping below validation means user intent is validated first
and system tuples bypass dynamic-mode evolution and Required checks:

```
stack: logging(enforce(cache(version(raw))))
```

`Apply(m)` for a mutation on `xdb://ns/S/id`:

1. Extract the CAS carrier. Remove attr `_version` from `m.Tuples` if
   present -> `want` (absent = 0 = unconditional). Reject user-supplied
   `_updated`/`_id` tuples upstream in enforcement (`ErrSchemaViolation`): only `_version` is writable, and only as an expectation, never a value.

2. Current version: `OpCreate` skips the read (`cur = 0`; a losing racer
   fails in the driver before any stamp). Otherwise point-read
   `GetTuples(path#_version)` -> `cur`.

3. CAS: `next, err := schema.NextRevision(cur, want)`: reused verbatim;
   stale `want` -> `core.ErrConflict`, which joins the record-write error
   tables.

4. Stamp and forward. Append `_version = next` and `_updated = now` to
   `m.Tuples` and forward: one atomic `Apply` for patch/create/put on every
   backend. `OpDelete` with empty `Attrs` forwards untouched (version dies
   with the record; recreate restarts at 1, matching schema-revision
   behavior). `OpDelete` with attrs forwards, then applies a follow-up
   `OpPatch{_version, _updated}`: and if the delete emptied the record,
   nothing (the record, and its version, are gone).

On filesystem and Redis, another writer can change the record between the version read and the write. Transactional drivers avoid that race because the facade wraps the read and write in one transaction.

Reads need no middleware work: `_version`/`_updated` come back as ordinary
tuples in scans, point reads, lists, and sqlite pushdown. Records read then
re-upserted carry their `_version`: read-modify-write gets optimistic
locking by default, etag-style. Building a fresh record without `_version`
writes unconditionally; deleting the attr from a read record is the
way to request an unconditional overwrite.

### Facade and Periphery

- `scanRecords` / `NewRecordFromTuples` call sites inject virtual `_id`.

- `ValidateRecord` mirrors the write path: strip `_version` before checks so
  validate-only agrees with `Apply`.

- Clock: package-level `now func() time.Time` (or option) for deterministic
  `_updated` in tests.

## API and CLI Integration

The API, RPC, watch, and CLI surfaces expose the system fields:

1. Every record surface already speaks `Data json.RawMessage`.
   `CreateRecordRequest`, `UpdateRecordRequest`, `UpsertRecordRequest`,
   their responses, and `BatchOperation` all carry an opaque JSON record
   body. System fields are keys in it: no new request or response
   fields for get/create/update/upsert/batch.

2. `xdbjson` already has the metadata-field machinery: `_id`, `_ns`,
   `_schema` with per-client renaming (`WithIDField`). `_version`/`_updated`
   extend the existing pattern rather than inventing one.

3. Conflict is already wired end to end. `core.ErrConflict` ->
   `rpc.CodeConflict` (-32003) at `rpc/router.go:272` -> CLI `CONFLICT` /
   `ExitAppError` at `cli/errors.go:41`, built for the schema revision CAS.
   Record CAS inherits the whole chain with no new error surface.

### Read-Modify-Write Preconditions

A client that writes back the record body returned by a read includes its version precondition:

```
xdb get xdb://app/users/123     → {"_id":"123","_version":3,"name":"Ada",…}
  (edit name)
xdb update xdb://app/users/123  → _version:3 becomes the precondition
                                  → CONFLICT if anyone wrote meanwhile
```

Updates, HTTP/RPC calls, and batch operations can use the record body to carry the precondition. Clients that want an unconditional write build a fresh body or remove `_version`. Deletes need an explicit precondition because they have no record body.

### Watch Gains an Explicit `Version`

Add a top-level `Version int64` to `WatchEvent{TS, Type, URI, Data}`:

- Delete events carry no `Data`, so the version would be unavailable for consumers to inspect.

- Consumers should not parse a payload to get the ordering key.

The bus drops events when a subscriber exceeds its buffer of 64. A version jump from 3 to 7 for the same record reveals that the subscriber missed writes and should re-read the record. Delivery remains at-most-once.

### API and CLI Decisions

- Delete has nowhere to put a precondition.
  `DeleteRecordRequest{URI, DryRun}` has no `Data`. Delete-if-unchanged
  needs an explicit `Version int64 \`json:"version,omitempty"\`` field: the one place the uniform in-body carrier does not reach. This supports deletion only when the record has not changed.

- Forged system fields must be rejected on decode. `_version` is
  client-writable but only as an *expectation*; `_updated` and `_id` are
  derived and a client-supplied value is `ErrSchemaViolation`, not a
  silent overwrite.

- CLI output grows three keys per record. Recommendation: JSON output
  shows all three (for machine consumers); the human table view
  shows `_version` but omits `_updated` unless `-o json`/wide. Flagged as a
  presentation decision, not settled here.

- HTTP `ETag`/`If-Match` mapping: declined for now. The in-body carrier
  already delivers the semantics, and the transport is JSON-RPC rather than
  REST, so etag headers would be a second, redundant mechanism to keep in
  sync. Revisit only if a REST surface appears.

## Costs

- Every record read now includes `_version`/`_updated` (+ virtual `_id`): existing test suites, e2e expectations, JSON/CLI output all shift. This is
  the bulk of the diff.

- Defs visibly carry two system fields in `describe`/list/export; import
  must strip them.

- One extra point read per put/patch write (none for create).

- `_`-prefixed field names become reserved: the only compatibility break,
  and one that fixes a live DDL collision.

## Alternatives Considered

| Alternative | Why not |
| --- | --- |
| Sidecar version records `xdb://ns/_v_S/id` (v1) | Two-apply write window on fs/redis; version invisible to lists/filters (pushdown bypasses middleware); sweep bookkeeping on delete/drop. Superseded. |
| Hidden `_version` tuple, fields not declared | Dropped by sqlite's column engine; fixing it puts policy in drivers |
| Unprefixed names (`xid`/`rid`, `xversion`/…) | Renames a value every backend already stores as `_id`, so column pushdown gains a translation layer without adding a capability; contradicts `xdbjson`'s shipped `_id`/`_ns`/`_schema` defaults; invades the user namespace (`rid` is a plausible user field); turns one prefix rule into a hardcoded name list; departs from the existing `_id` convention |
| Client-configurable field names in storage | Fragments filters/tooling/docs; agents can't rely on a name. The legitimate version of this already exists one layer up, at the encoding boundary (`xdbjson.WithIDField`). |
| Store-level opt-in (`WithVersioning()`) | Creates stores with different record shapes; always-on versioning keeps those shapes consistent |
| `Mutation.Version` + native driver CAS | Touches all four drivers + conformance suite; could provide atomic CAS for backends without transactions; deferred from this proposal |

## Implementation Plan (TDD)

1. Reservation first (standalone fix): `Def.Validate` rejects
   `_`-prefixed fields: test with `_id` (the DDL collision), `_version`,
   `_updated`.

2. `tests` shared suite additions run on all four backends: create stamps
   `_version=1` + `_updated`; put/patch/attr-delete increment and re-stamp;
   whole-delete + recreate restarts at 1; CAS pass / conflict / absent =
   unconditional; `_version` visible in GetRecord, GetTuple, ListRecords
   (incl. filtered), pushdown paths; `_id` virtual in reads, rejected in
   defs and writes; filters on `_version`/`_updated`/`_id`; rolled-back tx
   leaves version untouched (TX stores); lazy def upgrade on first write to
   a pre-existing schema.

3. `store/version.go`: the middleware; reuse `schema.NextRevision`; clock
   injection.

4. `enforce`: def stamping on create/put, lazy upgrade write-back, reject
   user `_updated`/`_id` writes, strip in `checkMutation`.

5. Facade: stack order, virtual `_id` injection, `ValidateRecord` parity.

6. `filter/sqlgen`: `_id` -> physical column mapping under both strategies.

7. `encoding/xdbjson`: encode `_version`/`_updated`; reject client-supplied
   `_updated`/`_id` on decode; `WithVersionField`/`WithUpdatedField` to match
   the existing `WithIDField` renaming.

8. API/RPC: nothing for get/create/update/upsert/batch (the fields ride in
   `Data`); add `Version` to `DeleteRecordRequest` and to `WatchEvent`;
   publish the post-write version from the record services.

9. CLI: render `_version` in record output; surface `CONFLICT` guidance
   ("re-read and retry") in the existing error-code help.

10. Existing-suite fallout: update expectations across store/e2e/CLI tests.

11. Docs: `docs/concepts/versioning.md`; updates to `stores.md`,
    `drivers.md`, `schemas.md`, `records.md`, `filters.md`, and the watch
    docs for the new `Version` field and gap detection.

## Future Extension: History Snapshots

With `_version` in-record, retained history becomes a separate middleware
writing `xdb://ns/_h_S/<id>.<version>` snapshots per write (the one place
the sidecar pattern still fits). Needs read-modify for full snapshots on
patches and a keep-last-N pruning policy. Out of scope for this proposal.
