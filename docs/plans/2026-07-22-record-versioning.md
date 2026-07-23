# Per-Record Versioning via System Fields

**Status:** design proposal (v2 — supersedes the sidecar design, kept below as
a rejected alternative)
**Date:** 2026-07-22

## Goal

Give every record system metadata, mirroring what schemas already have
(`Def.Revision` + CAS):

- `_version` — monotonically increasing counter, bumped on every successful
  write (create = 1).
- `_updated` — timestamp of the last successful write.
- `_id` — the record's id, surfaced as a readable/filterable attribute.
- Writers may pass an expected `_version`; a mismatch fails with
  `core.ErrConflict` (optimistic concurrency, same contract as
  `UpdateSchema`).

## The pivot from v1

The v1 (sidecar) design existed to route around one fact: sqlite's
column-table engine silently drops tuples whose attr is not a declared
column (`valuesFromTuples` iterates `sortedColumns(def)`), so a hidden
`_version` tuple on the record path could not survive strict/dynamic
schemas without driver changes.

**Declaring the system fields in every def dissolves that blocker.** A
declared field is a real column — the DDL follows automatically from the
def, drivers stay pure storage, and the version tuple rides *inside the
record's own mutation*. That is strictly better than the sidecar:

| | Sidecar (v1) | System fields (v2) |
| --- | --- | --- |
| Record write + version stamp | two `Apply` calls — atomic only on TxDrivers | one `Apply` — atomic on **every** backend (put/create/patch) |
| Version visible in reads | `GetRecord`/`GetTuple` only (pushdown bypasses middleware) | all reads, uniformly — it is a real attr/column |
| Filterable (`_updated > X`, `_version == N`) | no | yes, including sqlite SQL pushdown |
| Delete cleanup | sidecar sweep on delete + `DropRecords` | free — version dies with the record |
| Reserved names needed | reserved schema prefix `_v_*` | reserved field prefix `_` |
| Blast radius | store package only | schema + store + encoding/CLI expectations + migration |

The extra blast radius buys capabilities the sidecar structurally cannot
offer (filterable metadata, single-write atomicity), so v2 is the
recommendation.

## Forcing fact: `_id` is already taken

Both sqlite engines already use `_id` as the physical primary-key column
(`internal/sql/ddl.go`). A user def declaring a field named `_id` today
generates `_id TEXT NOT NULL` twice in `CREATE TABLE` — broken DDL. The
`_`-prefix field reservation below is therefore a latent-bug fix independent
of versioning.

Taken by the *same concept*, though: that column holds `path.ID()` — exactly
what the virtual `_id` exposes. Reserving the name and surfacing it as the
record id are one act, not two competing claims on it.

## Design

### Fixed names, always on

- **Fixed reserved names** (`_version`, `_updated`, `_id`), not
  client-configurable field names. XDB is agent-first: an agent walking into
  any namespace must be able to rely on `_version` meaning the same thing
  everywhere. Configurable names fragment CEL filters, CLI columns, docs,
  and tooling for zero structural gain.
- **Renaming belongs at the encoding boundary, not in storage.**
  `xdbjson` already does this: `WithIDField("userId")` renames `_id` on the
  wire while storage and filters keep the canonical name. Adding
  `WithVersionField`/`WithUpdatedField` is the natural follow-on, and it
  gives clients the naming freedom without fragmenting the query model.
- **Why `_`-prefixed, and why `_id` specifically** (vs `xid` / `rid`):
  `xdbjson` already ships `_id`, `_ns`, `_schema` as its default metadata
  names, so `_id` is *already* this project's public name for record
  identity — an unprefixed spelling would put storage at odds with the
  encoder. The prefix also makes reservation one rule ("no user field starts
  with `_`") instead of a hardcoded name list, and keeps the three system
  fields coherent; unprefixed names invade the user namespace (`rid` is a
  plausible user field: request id, row id, resource id). And `_id` is the
  most recognizable system field in document stores, so an agent writing a
  filter guesses it first. Decisively, though, `_id` is simply what every
  backend already calls this value at rest — the name matches the thing, so
  column pushdown needs no translation layer (see below).
- **Always on**, like enforcement — not a store option. Uniformity is the
  point; opt-in creates two classes of stores whose record shapes differ.
  Cost is one point read (`GetTuples(path#_version)`) per put/patch write —
  create needs none — and that read is in-tx on TxDrivers. If a hot path
  ever needs out, the escape hatch is a per-schema `Def` flag added later,
  disabling stamping — never renaming.

### Two mechanisms, not one

- **`_version`, `_updated` are stored fields**: stamped into every def
  (INTEGER / TIME, not Required), written as ordinary tuples by the
  middleware. Column tables get real columns; KV/flexible/schema-less
  storage stores them like any attr.
- **`_id` is virtual**: never stored as a tuple, because every backend
  already stores it as the record's *addressing key* — sqlite's `_id` PK
  column (both engines), `xdbfs`'s filename (`paths.go`), `xdbredis`'s key
  suffix (`driver.go`). It is the same `_id` one layer down, not a colliding
  name: same value (`path.ID()`), same meaning. Consequences: it costs
  nothing at rest on any backend, it can never disagree with the path, and
  it needs no stamping at all — unlike `_version`/`_updated`, it is pure
  projection. The facade injects it at record-assembly time from the path.

  Pushdown follows from that identity. Under `ColumnStrategy`, exempting
  `_id` from `walkIdent`'s `def.Fields` membership check is *sufficient* —
  emission is already `"_id"`, the correct column, with no name translation
  (a renamed field like `rid` would need one). Under `KVStrategy` a genuine
  special case is required under any spelling: identifiers are bound as
  `_attr` query parameters, and a virtual id has no attr row, so `_id` must
  target the KV table's `_id` column instead of an attr lookup.

### Def stamping (enforce middleware)

- `CreateSchema`: after `Validate`, append `_version` (INTEGER) and
  `_updated` (TIME) to `Fields`, then stamp `Revision = 1` as today.
- `PutSchema`: strip any incoming system fields, re-stamp canonically before
  `ValidateUpdate` — user tampering (type change, Required, deletion) is
  unrepresentable rather than rejected case-by-case.
- `Def.Validate`: reject user-declared `_`-prefixed field names (also fixes
  the `_id` DDL collision). Schema import strips system fields on ingest;
  export includes them (honest describe output).
- **Lazy upgrade for pre-existing defs**: when enforcement loads a def
  lacking system fields, stamp and `PutSchema` the upgraded def (same
  write-back pattern as dynamic-mode evolution). sqlite's `evolve` then
  issues `ALTER TABLE ADD COLUMN` — no offline migration step.

### Versioning middleware (write path)

New driver middleware, installed unconditionally by `store.New` **below
`enforce`, above the raw driver** — it needs no schema reads at all (fixed
names), and stamping below validation means user intent is validated first
and system tuples never hit dynamic-mode evolution or Required checks:

```
stack: logging(enforce(cache(version(raw))))
```

`Apply(m)` for a mutation on `xdb://ns/S/id`:

1. **Extract the CAS carrier.** Remove attr `_version` from `m.Tuples` if
   present → `want` (absent = 0 = unconditional). Reject user-supplied
   `_updated`/`_id` tuples upstream in enforcement (`ErrSchemaViolation`) —
   only `_version` is writable, and only as an expectation, never a value.
2. **Current version**: `OpCreate` skips the read (`cur = 0`; a losing racer
   fails in the driver before any stamp). Otherwise point-read
   `GetTuples(path#_version)` → `cur`.
3. **CAS**: `next, err := schema.NextRevision(cur, want)` — reused verbatim;
   stale `want` → `core.ErrConflict`, which joins the record-write error
   tables.
4. **Stamp and forward.** Append `_version = next` and `_updated = now` to
   `m.Tuples` and forward — one atomic `Apply` for patch/create/put on every
   backend. `OpDelete` with empty `Attrs` forwards untouched (version dies
   with the record; recreate restarts at 1, matching schema-revision
   behavior). `OpDelete` with attrs forwards, then applies a follow-up
   `OpPatch{_version, _updated}` — and if the delete emptied the record,
   nothing (the record, and its version, are gone).

The read-modify-write races on non-transactional backends (fs, redis)
exactly as the enforcement middleware's merge-that-creates check already
does; on TxDrivers the facade wraps everything in one transaction. Same
documented property, no new risk class.

Reads need no middleware work: `_version`/`_updated` come back as ordinary
tuples in scans, point reads, lists, and sqlite pushdown. Records read then
re-upserted carry their `_version` — read-modify-write gets optimistic
locking by default, etag-style. Building a fresh record without `_version`
writes unconditionally; deleting the attr from a read record is the
explicit unconditional-overwrite escape hatch.

### Facade and periphery

- `scanRecords` / `NewRecordFromTuples` call sites inject virtual `_id`.
- `ValidateRecord` mirrors the write path: strip `_version` before checks so
  validate-only agrees with `Apply`.
- Clock: package-level `now func() time.Time` (or option) for deterministic
  `_updated` in tests.

## Flowing to the top

The system fields are not a store-layer detail — they travel unchanged to
the API, RPC, watch, and CLI surfaces. Three facts make that nearly free:

1. **Every record surface already speaks `Data json.RawMessage`.**
   `CreateRecordRequest`, `UpdateRecordRequest`, `UpsertRecordRequest`,
   their responses, and `BatchOperation` all carry an opaque JSON record
   body. System fields are just keys in it — **no new request or response
   fields** for get/create/update/upsert/batch.
2. **`xdbjson` already has the metadata-field machinery** — `_id`, `_ns`,
   `_schema` with per-client renaming (`WithIDField`). `_version`/`_updated`
   extend the existing pattern rather than inventing one.
3. **Conflict is already wired end to end.** `core.ErrConflict` →
   `rpc.CodeConflict` (-32003) at `rpc/router.go:272` → CLI `CONFLICT` /
   `ExitAppError` at `cli/errors.go:41`, built for the schema revision CAS.
   Record CAS inherits the whole chain with **no new error surface**.

### The payoff: safe read-modify-write by default

Because the CAS carrier rides *inside* the record body, the round trip is
self-securing at every layer:

```
xdb get xdb://app/users/123     → {"_id":"123","_version":3,"name":"Ada",…}
  (edit name)
xdb update xdb://app/users/123  → _version:3 becomes the precondition
                                  → CONFLICT if anyone wrote meanwhile
```

No `If-Match` header, no `--if-version` flag, no client opt-in. The same
holds for HTTP/RPC clients and for batch operations. **Lost-update
protection becomes the default rather than an advanced feature** — a client
must go out of its way (build a fresh body, or drop the key) to overwrite
blindly. For an agent-first system, where the writer is usually a model
doing read-modify-write, this is the single largest reason to flow the
fields up rather than stop at the store.

### Watch gains an explicit `Version`

`WatchEvent{TS, Type, URI, Data}` gets a top-level `Version int64`. Two
reasons it should not be left implicit inside `Data`:

- Delete events carry no `Data`, so the version would be unavailable exactly
  where ordering matters most.
- Consumers should not parse a payload to get the ordering key.

The real win is **gap detection**. The bus is deliberately lossy — a
subscriber further behind than `subscriberBuffer` (64) silently misses
events ("at-most-once delivery... never blocking the publisher"). With
per-record versions on the wire, a consumer that sees `3 → 7` *knows* it
missed writes and can re-read. Today that loss is undetectable. This turns
watch from "lossy, hope for the best" into "lossy but detectable" — a
materially different contract for the same delivery guarantee.

### Gaps and decisions at the top

- **Delete has nowhere to put a precondition.**
  `DeleteRecordRequest{URI, DryRun}` has no `Data`. Delete-if-unchanged
  needs an explicit `Version int64 \`json:"version,omitempty"\`` field —
  the one place the uniform in-body carrier does not reach. Worth adding:
  "delete only if nobody has touched it" is a real need, and the asymmetry
  is unavoidable.
- **Forged system fields must be rejected on decode.** `_version` is
  client-writable but only as an *expectation*; `_updated` and `_id` are
  derived and a client-supplied value is `ErrSchemaViolation`, not a
  silent overwrite.
- **CLI output grows three keys per record.** Recommendation: JSON output
  shows all three (machine consumers and honesty); the human table view
  shows `_version` but omits `_updated` unless `-o json`/wide. Flagged as a
  presentation decision, not settled here.
- **HTTP `ETag`/`If-Match` mapping: declined for now.** The in-body carrier
  already delivers the semantics, and the transport is JSON-RPC rather than
  REST, so etag headers would be a second, redundant mechanism to keep in
  sync. Revisit only if a REST surface appears.

## Costs, stated plainly

- Every record read now includes `_version`/`_updated` (+ virtual `_id`) —
  existing test suites, e2e expectations, JSON/CLI output all shift. This is
  the bulk of the diff.
- Defs visibly carry two system fields in `describe`/list/export; import
  must strip them.
- One extra point read per put/patch write (none for create).
- `_`-prefixed field names become reserved — the only compatibility break,
  and one that fixes a live DDL collision.

## Alternatives considered

| Alternative | Why not |
| --- | --- |
| Sidecar version records `xdb://ns/_v_S/id` (v1) | Two-apply write window on fs/redis; version invisible to lists/filters (pushdown bypasses middleware); sweep bookkeeping on delete/drop. Superseded. |
| Hidden `_version` tuple, fields not declared | Dropped by sqlite's column engine; fixing it puts policy in drivers |
| Unprefixed names (`xid`/`rid`, `xversion`/…) | Renames a value every backend already stores as `_id`, so column pushdown gains a translation layer for nothing; contradicts `xdbjson`'s shipped `_id`/`_ns`/`_schema` defaults; invades the user namespace (`rid` is a plausible user field); turns one prefix rule into a hardcoded name list; loses agents' strong `_id` prior |
| Client-configurable field names in storage | Fragments filters/tooling/docs; agents can't rely on a name. The legitimate version of this already exists one layer up, at the encoding boundary (`xdbjson.WithIDField`). |
| Store-level opt-in (`WithVersioning()`) | Two classes of stores with different record shapes; uniformity is the feature |
| `Mutation.Version` + native driver CAS | Touches all four drivers + conformance suite; right second step if a backend ever needs hard CAS under concurrency, wrong first step |

## Implementation plan (TDD)

1. **Reservation first** (standalone fix): `Def.Validate` rejects
   `_`-prefixed fields — test with `_id` (the DDL collision), `_version`,
   `_updated`.
2. `tests` shared suite additions run on all four backends: create stamps
   `_version=1` + `_updated`; put/patch/attr-delete increment and re-stamp;
   whole-delete + recreate restarts at 1; CAS pass / conflict / absent =
   unconditional; `_version` visible in GetRecord, GetTuple, ListRecords
   (incl. filtered), pushdown paths; `_id` virtual in reads, rejected in
   defs and writes; filters on `_version`/`_updated`/`_id`; rolled-back tx
   leaves version untouched (TX stores); lazy def upgrade on first write to
   a pre-existing schema.
3. `store/version.go` — the middleware; reuse `schema.NextRevision`; clock
   injection.
4. `enforce` — def stamping on create/put, lazy upgrade write-back, reject
   user `_updated`/`_id` writes, strip in `checkMutation`.
5. Facade — stack order, virtual `_id` injection, `ValidateRecord` parity.
6. `filter/sqlgen` — `_id` → physical column mapping under both strategies.
7. `encoding/xdbjson` — encode `_version`/`_updated`; reject client-supplied
   `_updated`/`_id` on decode; `WithVersionField`/`WithUpdatedField` to match
   the existing `WithIDField` renaming.
8. API/RPC — nothing for get/create/update/upsert/batch (the fields ride in
   `Data`); add `Version` to `DeleteRecordRequest` and to `WatchEvent`;
   publish the post-write version from the record services.
9. CLI — render `_version` in record output; surface `CONFLICT` guidance
   ("re-read and retry") in the existing error-code help.
10. Existing-suite fallout: update expectations across store/e2e/CLI tests.
11. Docs — `docs/concepts/versioning.md`; updates to `stores.md`,
    `drivers.md`, `schemas.md`, `records.md`, `filters.md`, and the watch
    docs for the new `Version` field and gap detection.

## Future extension: history snapshots

With `_version` in-record, retained history becomes a separate middleware
writing `xdb://ns/_h_S/<id>.<version>` snapshots per write (the one place
the sidecar pattern still fits). Needs read-modify for full snapshots on
patches and a keep-last-N pruning policy. Deliberately out of scope.
