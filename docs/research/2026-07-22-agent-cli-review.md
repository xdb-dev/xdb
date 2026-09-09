# Agent-Perspective CLI Review

Date: 2026-07-22
Method: six parallel sub-agent probes against a freshly built binary, each in an
isolated sandbox (`HOME=$(mktemp -d)`, own daemon), black-box rules from
`tests/e2e/RUNBOOK.md`. Probes: cold-start discoverability (blind agent, no docs),
error taxonomy (~20 error classes), introspection fidelity (describe/help vs
reality), state recovery + retry safety, batch/watch/import/export + output
hygiene, and skills follow-through. The three highest-impact claims were
re-verified by hand before writing this doc.

## Verdict

The CLI exposes structured output, error hints, introspection, and embedded skills. Several commands fail to meet their documented contracts: `--dry-run` writes data, `batch` is unimplemented, and some failures exit successfully. Incorrect error codes also make user input errors look like server failures.

## Verified Behavior

- Top-level `--help` describes the command structure: URI scheme with
  concrete examples, all resources/operations/aliases, exit-code table, pointers
  to `describe` and `skills`. The agent using only CLI help completed a seven-step task with little flag guessing.

- Error hints suggest recovery commands: `delete requires --force to confirm`,
  `is the daemon running? try: xdb daemon start`, NOT_FOUND suggesting the parent
  list. The suggested commands resolved the tested errors.

- Stdout hygiene is clean: in everything probed, human prose (init/daemon
  messages, import summary) goes to stderr; stdout carries only JSON/NDJSON.
  Export pipes to `jq` unmodified. One exception: see P0-2.

- Crash durability and idempotency: `kill -9` on the daemon lost zero
  records; stale pid/socket detected cleanly; `daemon start` recovers; delete /
  upsert / import-upsert retries are idempotent; export -> delete -> import
  roundtrip is byte-identical including `_id`.

- `describe --uri <schema>` is sufficient to construct a valid record without
  guessing (fields, types incl. `array<string>`, required, mode).

- `describe --config` / `--daemon` / `--errors` / `--filter` are accurate,
  offline, and useful (config matches the generated file key-for-key; all
  documented filter operators/functions execute).

- Skills are embedded in the binary: list and fetch work offline, before
  `init`.

## Gaps

Severity: P0 means documented behavior is false; P1 prevents reliable error handling or planning; P2 adds avoidable work.

### P0: Advertised Behavior That Is False

1. `--dry-run` is a silent no-op; mutations persist. `records
   create/update/upsert/delete --dry-run` and `schemas create/delete --dry-run`
   all execute for real, exit 0. The flag is defined in `records.go`/`schemas.go`
   but only ever read in `batch.go` and `schemaimport.go`. (Verified by hand.)

2. Unknown command exits 0 and dumps markdown to stdout. `xdb frobnicate` (or
   any typo'd verb) prints the 4.5KB "# XDB CLI Context" doc on stdout, exit 0: the root `Action` in `app.go` treats any unmatched arg as a context request.
   This breaks `| jq` pipelines and hides typos. It was the only stdout-format violation found. (Verified by hand.)

3. Conflicting duplicate `create` silently drops the new payload. Second
   `records create` (or `schemas create`) on an existing URI with a *different*
   payload returns the OLD resource, exit 0, no warning. `ALREADY_EXISTS` is in
   the error catalog but unreachable. For schemas, the command reports success while the revision stays at 1. (Verified by hand.)

4. `batch` and `watch` are advertised but unimplemented. `batch.execute`
   returns `INTERNAL / not implemented` exit 4 despite top-level help ("Execute
   multiple operations atomically"), full `batch --help` flags, and a
   `describe batch.execute` response promising atomic transactions. `watch` exits
   1 with "pubsub support (not yet implemented)" despite "Stream change
   notifications as NDJSON" in help. Agents will plan around atomicity and
   streaming that don't exist. Either implement, or mark unavailable in
   help/describe and return a distinct `NOT_IMPLEMENTED` code.

5. The bundled `getting-started` skill fails at step 1 as written. It teaches
   `{"Type": "bool"}`; the engine only accepts `boolean`
   (`cli/skills/getting-started/SKILL.md:18`). The bundled starting instructions fail as written: and fails as `INTERNAL` exit 4. Skills
   should be CI-tested against the binary (an e2e scenario that replays each
   skill's code blocks).

### P1: Error Contract Drift (Agents Can't Branch Reliably)

6. Exit 3 is nearly unreachable; documented mapping is wrong. Help promises
   exit 3 for INVALID_URI and bad JSON. Observed: bad URI -> `INTERNAL` exit 4;
   bad `--json` -> bare-envelope exit 1; only the `--force` guard produced exit 3.
   `INVALID_URI` appears in help but exists in neither `describe --errors` nor
   `errors.go`.

7. Pure input errors misfile as `INTERNAL` exit 4: malformed URI, filter
   syntax error, filter on unknown field (which leaks
   `sqlite3: SQL logic error: no such column: ...`), unknown schema type,
   invalid schema mode. Agents treating exit 4 as "server bug: don't retry,
   escalate" misclassify their own mistakes. These should be
   INVALID_ARGUMENT/SCHEMA_VIOLATION with the valid options enumerated
   (`unknown type "bool"; valid: string, integer, ..., boolean, ...`).

8. Three error shapes exist. (a) the standard envelope; (b) fallback
   `{"error": "<string>"}`: no code, no hint, always exit 1 regardless of class
   (`ExitCodeFor` returns `ExitAppError` for non-envelope errors); (c) plain-text
   `Incorrect Usage: flag provided but not defined` for flag parses. The fallback
   path covers bad `--json`, `describe <name>` errors, `skills get` misuse, and
   `import` misuse: and leaks `rpc error -32603` internals. In `describe.go`
   only `listActions` wraps with `wrapRPCError`; `listMethods`, `listTypes`,
   `describeMethod`, `describeType`, `describeDataSchema` return raw errors, so
   daemon-down on those paths yields exit 1 instead of 2 with no hint: breaking
   auto-start-daemon retry logic on the exact commands a cold-start agent runs
   first.

9. Other commands that report success without the expected result: `import --create-only`
   prints "Imported 3 records" when it wrote 0 (existing IDs silently kept);
   `export` of a nonexistent schema exits 0 with empty output (indistinguishable
   from an empty schema); `export` of a record-level URI silently exports the
   whole schema; `xdb skills <name>` (missing `get`) re-lists and exits 0;
   import silently rewrites conflicting `_ns`/`_schema` to the target URI.

10. `records list --query` is a dead flag: defined, advertised as
    "Structured JSON query", never read; garbage input exits 0 with unfiltered
    results.

### P1: Introspection Gaps

11. The schema-definition format is not introspectable. `describe Schema` is
    a one-liner; `describe schemas.create` says only `data: "Schema definition as
    JSON object"`; valid `mode` values are listed nowhere (found by guessing);
    `elem_type` (required for arrays) is only learnable from an error message.
    Without the (broken) skill, an agent cannot construct a schema from the
    machine surface. Needs a `describe` topic for the definition shape:
    `{fields: {<name>: {type, required, elem_type, ...}}, mode, description}`.

12. `describe <resource.action>` documents the RPC, not the CLI. `--force`
    (delete), `--page-all`, `--query`, `--json/--file/--quiet/-o` are absent;
    conversely it lists params that are RPC-only. An agent driving the CLI from
    `describe` output alone is guaranteed a failed first delete. Nothing states
    which surface `describe` reflects.

13. Method/type catalogs need a live daemon (`introspect.methods` RPC) even
    though they are static facts about the binary. The hand-authored halves
    (filter/errors/config/daemon/value-types) are offline; the halves a
    cold-start agent needs first are not. Embed the catalogs like the error
    catalog is.

14. The `json` value type is unusable end-to-end: object payloads get
    flattened and rejected as unknown nested fields; array/string/number get
    type-mismatch. A documented type cannot be stored via the CLI.

15. Documented filter dialect drifts: string functions (`contains`,
    `startsWith`, `endsWith`) are case-insensitive (compiled to SQLite `LIKE`)
    while `==` is case-sensitive: the "CEL (AIP-160)" claim implies otherwise
    and the grammar doc doesn't mention it. One of the six shipped examples
    (`!(archived == true)`) fails on any schema lacking `archived` with a raw
    sqlite error.

### P1: Sync and Scale

16. Records carry no revision or timestamp in get/list/export payloads
    (schemas have `revision`; records don't). With watch unimplemented, an agent
    has no primitive: streaming *or* polling: for incremental mirroring; full
    re-export is the only sync path.

17. `import` is fail-fast partial with no progress accounting: mid-file
    violation commits earlier lines, stops, and reports only the failing line: no "imported N of M", no machine-readable summary (the human summary is
    stderr prose). Resume-safety exists only because default mode is upsert.

18. No full-tree dump: cold-start inventory costs `1 + N + M` calls
    (namespaces -> schemas per ns -> records per schema); `export` is per-schema
    only; `namespaces get` returns the name without schemas. A recursive list or namespace-level export would reduce the calls needed to reconstruct stored state.

### P2: Friction

19. First `init` on a fresh HOME prints "Config already exists": the
    app-level `Before` hook (`app.go`) calls `LoadConfig` -> `EnsureConfigAt`,
    creating the config before `initAction` checks it (`init.go:28`). Side
    effect: *every* command, including read-only `describe --filter`, silently
    creates `~/.xdb/config.json`.

20. `batch --file` cwd-containment check breaks on symlinked cwd (macOS
    `/tmp` -> `/private/tmp`): a file directly in cwd is rejected as "outside
    working directory". Restriction is undocumented and `import --file` doesn't
    have it.

21. `--fields id` returns key `_id`: mask name and output key differ; jq
    users guess `.id` and get null. Accept/emit symmetrically or document the
    `_`-prefix mapping.

22. `--quiet` only on `records create/get/update/upsert/delete`: missing
    from list, schemas, import/export, init, daemon. (`records get --quiet` is a
    good existence probe; more commands should get it.)

23. Nonexistent subcommand `--help` exits 1 with zero output
    (`schemas upsert --help`, `records watch --help`): silent failure.

24. `daemon status` exits 0 when stopped: health-gating scripts must parse
    JSON instead of branching on exit code.

25. Misc: `describe --types` Value description says "bool" and omits
    unsigned/json/array; INVALID_ARGUMENT envelopes omit `uri`/`hint` (a hint
    exists in `hintFor` but `invalidArgError` never sets it); SCHEMA_VIOLATION
    hint on failed `schemas create` points at describing a schema that doesn't
    exist; "Imported 1 records" pluralization; hints JSON-escape `<` as `<`;
    `int64` mismatch reported as `got=FLOAT`; skill catalog has one entry and no
    "next steps" chain; skill omits the `xdb init` prerequisite; skill uses
    non-canonical `Fields`/`Type` casing.

## Priority Fix List

1. Wire `--dry-run` through records/schemas actions (or remove the flag). (P0-1)

2. Unknown command -> exit 3 + envelope on stderr; keep the context doc behind an
   explicit `xdb context`/`xdb agent` command. (P0-2)

3. Conflict signal on create-with-different-payload (`ALREADY_EXISTS` or a
   `"conflict": true` field); same for `schemas create`, with a hint pointing at
   `schemas update`/`diff`. (P0-3)

4. Make help/describe state that `batch`/`watch` are unavailable (or implement them); distinct
   NOT_IMPLEMENTED code. (P0-4)

5. Fix the skill (`boolean`, lowercase keys, init prerequisite) and add a CI
   scenario that replays every skill code block. (P0-5)

6. Unify error rendering: wrap all `describe` paths, `--json` parse errors, flag
   parse errors, and `skills`/`import` misuse into the envelope with correct
   classes; kill the `{"error": ...}` fallback; map input errors off `INTERNAL`;
   enumerate valid values in type/mode errors. (P1-6/7/8)

7. Add a schema-definition-format topic to `describe`; document CLI flags per
   action (or state that `describe` is the RPC surface and `--help` the CLI one);
   embed method/type catalogs for offline use. (P1-11/12/13)

8. Import accounting (`imported/skipped/failed` counts, machine-readable) +
   documented fail-fast semantics; fix `--create-only` miscounting. (P1-9/17)

9. Record-level `revision`/`updated_at` so polling sync is possible. (P1-16)

10. Fix or remove `--query`; fix `json` value type; document filter
    case-insensitivity or make it CEL-faithful. (P1-10/14/15)

## Probe Notes

Raw probe reports (cold-start friction log, full error-case table with exit
codes/streams/envelope keys, offline matrix, semantics tables) live in the
session transcripts; the tables above are deduplicated from them. Sandboxes were
torn down; no repo files were modified by probes.
