# Fix All Agent-CLI Review Findings + E2E Consolidation

## Context

The 2026-07-22 agent-perspective CLI review (`docs/research/2026-07-22-agent-cli-review.md`)
found the CLI's agent-first architecture sound but its self-description untrustworthy:
`--dry-run` silently writes, unknown commands exit 0, conflicting creates silently drop
data, `batch`/`watch` are advertised stubs, the bundled skill fails verbatim, and the
error/exit-code contract drifts badly off the happy path. The parallel e2e-scenario work
(`docs/plans/2026-07-22-e2e-cli-scenarios.md`) added 5 scenarios and surfaced encoding/API
findings: `json`-typed fields unpopulatable, `array<integer>` broken, `array<json>`
silently dropped, >2^53 precision loss, schema revision CAS and `items` unreachable.

This plan fixes all 31 findings across four layers (rpc/core, store/filter, encoding,
api, CLI) and consolidates `tests/e2e/scenarios.yaml` (13 scenarios) so every fix lands
with a regression scenario. Design was produced by 3 explore + 2 design sub-agents; all
mechanisms verified at file:line.

Two extra bugs found during planning, also fixed here:
- `schemas delete` defines `--force` but never reads it (unguarded deletion).
- Root `Before` daemon-skip check is dead code (`cmd.Name` is always "xdb" in urfave v3),
  so EVERY command creates `~/.xdb/config.json` as a side effect.

## Resolved decisions

| Decision | Choice |
|---|---|
| Bare `xdb` / unknown cmd | Help exit 0 / INVALID_ARGUMENT envelope exit 3; CONTEXT.md moves behind new `xdb context` command |
| Error codes (new) | rpc `-32003 CONFLICT`, `-32004 NOT_IMPLEMENTED`; `-32602` absorbs invalid URI/filter/depth (with `data.reason`); unknown-type/invalid-mode → `-32002` with valid values enumerated |
| CLI exit mapping | Unchanged contract: domain errors (NOT_FOUND, ALREADY_EXISTS, SCHEMA_VIOLATION, CONFLICT, NOT_IMPLEMENTED) → 1; INVALID_ARGUMENT → 3; CONNECTION_REFUSED → 2; INTERNAL → 4. Exit 3 becomes actually reachable |
| Create conflict | Identical payload → idempotent success (unchanged); different payload → CONFLICT with remediation hint. Same for schemas |
| Dry-run | Real server support: `dry_run` param on records create/update/upsert/delete + schemas create/delete via validate-only path (pure validators, works on all backends); response `dry_run:{valid,would}`; CLI asserts the marker (guards old-daemon skew) |
| batch | Implement: typed ops `{op,uri,data}`, atomic via `store.TX` on sqlite/memory; fs/redis refuse with NOT_IMPLEMENTED unless `non_atomic:true` |
| watch | Implement: in-process event bus published post-commit in api services; SSE `ready`→`event`→`done` framing; at-most-once, no replay (documented) |
| Filter | Case-SENSITIVE everywhere (sqlite LIKE → instr/substr); strict mode rejects unknown filter fields (message lists available); flexible/dynamic → no-match; sqlite unknown-column falls back to facade scan (no SQL leak); memory ≡ sqlite proven by shared suite |
| List output | BREAKING: `-o json/yaml` list → `{items, total, next_offset}`; ndjson stays bare stream; `--page-all` wired |
| Config creation | Only `init` and `daemon start` write config; other commands use in-memory defaults (missing explicit `--config` errors) |
| `--query` | Removed (no server support exists; silently dead today) |
| `daemon status` | Exit 2 when stopped; `--quiet` added |
| `--file` policy | Fix symlink bug in `validate.FilePath` (resolve cwd too); apply containment to BOTH batch and import |
| Record metadata | `_updated_at` reserved attr only (final, cuttable wave); `_rev`/CAS deferred with design sketch |
| Numbers | `json.Decoder.UseNumber()` always on; `WithNumberInference` becomes documented no-op |
| Catalog | Root-module `api/catalog` package: MethodMeta + type descriptions, single source for daemon registration AND offline CLI describe |

## Phases

TDD throughout: failing test → minimum code → green → `make check`. One commit per phase.
Root module and cmd/xdb module both touched; `cmd/xdb/go.mod` already `replace`s root.
E2E scenarios are added/updated IN the phase that changes behavior (listed per phase).

### Phase 1 — Error taxonomy (root: rpc/core/schema/filter)

- `rpc/errors.go`: `CodeConflict=-32003`, `CodeNotImplemented=-32004`, constructors.
- `rpc/router.go` `MapError` (rpc/router.go:257-274) new cases: `core.ErrConflict`→Conflict,
  `core.ErrNotImplemented`→NotImplemented, `core.ErrInvalidURI`→InvalidParams
  (`data.reason="invalid_uri"`), `core.ErrInvalidFilter`→InvalidParams
  (`data.reason="invalid_filter"`), `core.ErrUnknownType`→SchemaViolation,
  `schema.ErrInvalidMode`→SchemaViolation.
- `core/errors.go`: add `ErrInvalidFilter`, `ErrNotImplemented` sentinels.
- `core/types.go`: `ValueTypeNames()`; `ParseType` error lists valid types
  (types.go:71-79). `schema`: `ValidModes()`; mode errors enumerate (validate.go:48-50,
  json.go:92-93).
- `filter/filter.go`: every `Compile` error wraps `core.ErrInvalidFilter`.
- `api/batch.go:37` + `api/watch.go:29` interim stubs → `core.ErrNotImplemented`.
- Tests: `rpc/router_test.go` MapError table (bare + wrapped per sentinel, data.reason);
  `core/types_test.go` valid-list message; `filter/filter_test.go` sentinel table.

### Phase 2 — URI depth + catalog (root: core/api)

- `core/uri.go`: `(*URI).Depth()`. New `api/uri.go`: `parseURI(raw, method, min, max,
  allowAttr)` → wraps `core.ErrInvalidURI` with "expects xdb://ns/schema/id"-style hints.
  Apply per-method depth table in all handlers (records get/delete: 3+attr; create/update/
  upsert: 3; list: 1-2 [enables Phase 10 tree recovery]; schemas.*: 2, list: 1;
  namespaces.get: 1; watch: 1-3).
- New `api/catalog` package: `Methods() map[string]rpc.MethodMeta` (literals moved verbatim
  from cmd/xdb/daemon/daemon.go:81-265), `Method(name)`, `Types()` (moved from
  api/introspect.go:149-156, completed: add Filter/Mode/BatchOperation/Event/DryRunResult,
  fix "bool" drift in Value). Import graph: catalog→rpc only.
- `cmd/xdb/daemon/daemon.go`: registration pulls meta via `mustMeta(name)`.
- `api/introspect.go`: unknown method/type → `rpc.NotFound` with hint (was raw -32603).
- Tests: `core/uri_test.go` Depth table; per-service depth tables; `api/catalog/
  catalog_test.go` completeness; `daemon_test.go` router↔catalog two-way completeness.

### Phase 3 — CLI harness + dispatch + usage errors (cmd/xdb: W0, W1, W2)

- W0 harness: `NewAppWithIO(stdout, stderr)`; replace ALL direct `os.Stdout/Stderr` writes
  in cli package with `cmd.Root().Writer/ErrWriter` (app.go root Action, formatOne/List/
  RawJSON, ExitErrHandler, skills.go, import_export.go, init.go, daemon.go); extract shared
  `buildRoot` used by NewApp + NewEmbeddedCommand. New `harness_test.go`: `runCLI(t,
  args...) (stdout, stderr, code)` + `startCLITestDaemon(t)` (in-process daemon on temp
  socket + config, reuse integration_test.go pattern). Post-refactor grep check for stray
  os.Stdout writes.
- W1: new `context.go` `contextCmd()` prints CONTEXT.md. Root Action: args present →
  `invalidArgError` "unknown command %q" + suggestion (`cli.SuggestCommand`) + hint → exit
  3; no args → `cli.ShowAppHelp` exit 0. Update rootHelpTemplate + CONTEXT.md
  self-reference.
- W2: recursive `installUsageErrorHandler` walk sets `OnUsageError` on every command →
  INVALID_ARGUMENT envelope (suppresses urfave "Incorrect Usage" text). `errors.go`:
  `ExitCodeFor` handles `cli.ExitCoder` → 3; `normalizeError` converts ExitCoder to
  envelope in `WriteError`. Also extend `codeFromRPC` (errors.go:106-118): -32003→
  `CONFLICT`, -32004→`NOT_IMPLEMENTED` (both exit 1 via default); add both to
  `describe --errors` catalog (describe.go listErrorCodes) with descriptions + exits.
- Tests: `app_dispatch_test.go` tables (unknown cmd 3+envelope+small stdout; bare xdb help
  0; context cmd; typo suggestion); flag-parse envelope single-render; nonexistent
  subcommand `--help` → 3 + envelope; valid help still 0.
- E2E: new scenario `cli-contract-basics`.

### Phase 4 — CLI error-render unification + config policy (cmd/xdb: W3, W6)

- W3: wrap ALL describe RPC calls with `wrapRPCError("introspect", <action>, ...)` —
  listMethods/listTypes/describeMethod/describeType/describeDataSchema (+ParseURI via
  invalidArgError); fix `"action":"methods"` label in listActions. skills get: missing arg
  → invalidArgError; unknown skill → NOT_FOUND envelope with list hint. import: per-line
  errors wrapped (preserve code so daemon-drop mid-import exits 2; single `line %d:`
  prefix), misuse → invalidArgError. Export list error wrapped.
- W6: `LoadConfig` stops calling `EnsureConfigAt`; missing default path → validated
  in-memory defaults; missing explicit `--config` → error. Root Before skips connect for
  init/daemon/context/skills/help via `cmd.Args().First()` (fixes dead check). `daemon
  start` + `init` call `EnsureConfigAt` explicitly → init's created-flag truthful
  ("Created ..." on first run).
- Tests: `describe_errors_test.go` daemon-down table (all paths → exit 2 CONNECTION_REFUSED
  + hint); `skills_test.go`; `config_test.go` (no-write on missing default; explicit
  errors); `init_test.go` first/second run messages.
- E2E: extend `agent-self-discovery` with daemon-down describe assertions (new scenario
  `error-classes`, first half).

### Phase 5 — Filter hardening (root: filter/sqlgen/store)

- `filter.Compile`: strict mode rejects unknown idents (`core.ErrInvalidFilter`, message
  names field + sorted available list); flexible/dynamic keep DynType no-match; Filter
  retains its def; reserved-attr allowlist (`_updated_at` as TimestampType, for Phase 12).
- `sqlgen`: unknown ident under ColumnStrategy → `ErrUnknownColumn`; known idents
  double-quoted; `contains/startsWith/endsWith` → `instr`/`substr` (case-sensitive), both
  Column and KV strategies. `store/xdbsqlite/query.go`: map ErrUnknownColumn →
  `store.ErrUnsupportedQuery` → facade scan fallback (dynamic-mode alignment, no SQL leak).
- `store/facade.go:191`: fetch schema def for schema-scoped list URIs and pass to Compile
  (aligns memory/fs/redis with sqlite).
- Tests: filter_test strict/flexible/nil tables; sqlgen SQL-shape tests; shared
  `tests/query_suite.go` new subtests (unknown-field strict errors, flexible no-match,
  case-sensitivity) run against ALL four drivers — the sqlite≡memory proof.
- Docs: filters concept doc gains case-sensitivity + unknown-field policy.
- E2E: `error-classes` second half (filter syntax error → exit 3; strict unknown field →
  exit 3 naming field).

### Phase 6 — Schema API completion + create conflict (root: api) [S8, S3]

- S8: `schemaFieldPayload` gains recursive `Items`; `schemaDefPayload` gains `Revision`
  (api/schemas.go:216-234, unmarshalSchemaDef, applySchemaPatch sets caller's base
  revision). CAS free via existing `schema.NextRevision`: revision 0 → unconditional
  (compat), stale → CONFLICT. Field removal explicitly deferred (meta text says so).
- S3: records.create on `ErrAlreadyExists` → canonical compare (encode both → map →
  DeepEqual, reserved attrs excluded); equal → success, differ →
  `core.ErrConflict` "record exists with different data (use records.update or
  records.upsert)". schemas.create same (revision zeroed pre-compare, hint names
  schemas update/diff). Catalog meta text updated.
- Tests: api/schemas_test items round-trip + member violation; revision CAS trio;
  api/records_test conflict trio.
- E2E: new scenario `conflict-and-idempotency`.

### Phase 7 — Decoder rewrite (root: encoding/xdbjson) [S7]

- Schema-aware `flattenWithDef`: json-typed field (top-level or dotted) captures subtree
  verbatim as RawMessage, no descent (decoder.go:317-331 replacement).
- `convertToType` takes full `core.Type`; new TIDJSON case (marshal→JSONVal) and TIDArray
  case (per-element conversion to declared elem_type; ARRAY<JSON> without Items = plain
  passthrough — kills silent drop); float→int only when lossless.
- `UseNumber` always (decoder.go:77-89); `WithNumberInference` → documented no-op.
- Declared-field conversion failures → error wrapped with `core.ErrSchemaViolation`
  naming field + type; undeclared values keep flexible silent-skip. Verify
  `core.NewSafeValue` handles RawMessage → JSONVal.
- Tests: decoder_test tables — json field object round-trip; ARRAY<INTEGER> from [1,2,3];
  big int 9007199254740993 exact; ARRAY<JSON> ± Items; number defaults; declared garbage
  errors. Full `make test` = blast-radius net for UseNumber.
- E2E: EXTEND `type-fidelity` with newly reachable types (json object field,
  array<integer>, array<json>, >2^53 integer) — removes the documented CLI-unreachable
  caveats from the scenario description.

### Phase 8 — Dry-run server + CLI wiring (root api/store + cmd/xdb) [S4, W4]

- Server: extract `enforcer.check` → package-level `checkMutation` (store/enforce.go);
  `store.Validator` optional interface + facade impl (validate-only, zero writes, all
  backends). `DryRun bool` on 6 request structs; shared `api/dryrun.go`
  `DryRunResult{Valid, Would}` (would: create|update|replace|delete|noop); responses gain
  `dry_run` pointer (nil on real ops). Dry-run applies S3 conflict + S2 depth semantics.
- CLI W4: wire `--dry-run` → request field; assert response marker present else INTERNAL
  "daemon ignored dry_run" (old-daemon guard). `schemas update --dry-run` → fail-closed
  refusal (server defers it). Remove `--query` flag (records.go:255). `schemas delete`
  requires `--force` (mirror records.go:211-213).
- Tests: store/facade_test ValidateRecord table (incl. dynamic-mode no-persist proof);
  api dry-run tables (absent-after, would values, conflict, delete-preserves);
  cli records_test/schemas_test (dry-run validates without writing; query flag now parse
  error; schemas delete force trio).
- E2E: new scenarios `dry-run-contract`; `schemas delete --force` assertions into
  `schema-evolution`.

### Phase 9 — Batch implementation (root api + catalog) [S5]

- `BatchOperation{Op, URI, Data}`, `BatchResult{Index, URI, Status, Error, DryRun}`;
  `Operations []BatchOperation`; `NonAtomic bool`. Upfront validation of all ops (unknown
  op → InvalidParams listing allowed). TxDriver present → `tx.Run` all-or-nothing
  (RolledBack, per-index error, subsequent "skipped"); absent → NOT_IMPLEMENTED unless
  `non_atomic:true` (sequential, per-op attribution). DryRun → per-op S4 validation.
  Catalog meta documents op shape verbatim (describe teaches it).
- Tests: api/batch_test — success counts; mid-batch rollback proof; unknown op; dry-run;
  noTxStore refusal + non_atomic path.
- E2E: `bulk-import-via-stdin` drops `skip_if_unimplemented`, tightened; new
  `batch-atomic` scenario.

### Phase 10 — Tree recovery + list pagination render (root api + cmd/xdb) [S10, W8]

- S10: bless namespace-depth `records.list` (depth 1-2 from Phase 2; facade ns-scan
  already works; items carry `_schema`); `namespaces.get` response gains `schemas []string`
  + `total_schemas` (additive).
- W8: `output.Page{Items, Total, NextOffset}` + `FormatPage` per formatter (json/yaml
  envelope; ndjson bare lines; table items + stderr total). records/schemas list render
  pages; `--page-all` wired (loop, merged). `--quiet` added to schemas mutations, batch,
  import, schemaimport. `daemon status`: exit 2 stopped + `--quiet`. aliases.go: explicit
  depth guards with expected-depth messages (put=3, make-schema=2, rm/get/ls per table)
  → INVALID_ARGUMENT instead of NOT_FOUND conflation. Update CONTEXT.md + help exit-code
  table + describeDaemon.
- Tests: output FormatPage table; records list envelope/page-all; daemon status exit
  codes (PID-file simulation); aliases depth table.
- E2E: new `pagination-and-fields`; tighten existing list assertions to new shape.

### Phase 11 — Watch (root api/rpc/client + daemon + cmd/xdb) [S6]

- `api/events.go`: Bus (buffered 64, non-blocking publish, drop-on-full counter),
  component-wise `matchScope` (never string-prefix). Services publish post-success via
  `api.WithEvents(bus)` option. `WatchService`: parse (depth 1-3) → subscribe → send
  `ready` first → loop until ctx/close → clean `done`. Daemon owns bus lifecycle
  (Close on stop).
- `rpc/client`: add `Stream` method consuming SSE framing (ready/event/done/error).
- CLI watch command: NDJSON events to stdout, ready marker included, clean EOF on daemon
  stop.
- Semantics documented in meta + concept doc: at-most-once, in-process, no replay.
- Tests: events_test (matchScope table, delivery, unsubscribe, full-buffer, close);
  watch_test (ready-first, filter, cancel); services publish asserts; daemon SSE
  integration test; CLI watch harness test.
- E2E: new `watch-stream` scenario (ready marker, event per mutation, clean end).

### Phase 12 — Describe + import/export + skills (cmd/xdb: W5, W7, W9)

- W5: describe offline fallback to `api/catalog` on connection error (`"source":
  "embedded"` marker); `--schema-format` topic (modes from schema.ValidModes, types from
  core.ValueTypeNames — drift-proof); `describeMethod` gains `cli` section (flags walked
  from live command tree); bare `describe` → topic overview exit 0.
- W7: `validate.FilePath` resolves cwd symlinks (validate.go:66-91); import routes --file
  through it (policy unified). Export: `-o json` array mode, other formats rejected;
  depth-3 URI → single-record export; depth-1 → INVALID_ARGUMENT; pre-flight `schemas.get`
  → NOT_FOUND for missing schema. Import: independent line counter (blank-line fix);
  summary JSON `{imported, skipped, failed, first_error_line}` on stdout; `--create-only`
  counts skipped via CONFLICT/identical distinction (S3 signal).
- W9: fix getting-started SKILL.md (lowercase keys, `boolean`, `xdb init` prereq,
  next-steps footer); `skills <name>` acts as get, unknown → NOT_FOUND envelope; new
  skills: query-and-filter, schema-evolution, bulk-data (contents per design outline).
- Tests: describe_test (offline fallback table, cli-flags section, schema-format,
  overview); catalog↔router drift test; validate symlink test (t.Chdir into symlink);
  import/export tables; skills_test incl. `TestSkillsPayloadsAreValidAgainstServer`
  (executes skill payloads against live test daemon — the anti-drift gate).
- E2E: new `import-accounting`, `skills-verbatim` (replays every SKILL.md code block).

### Phase 13 — Cosmetics + docs + e2e style pass (cmd/xdb + root) [W10]

- `invalidArgError` gains uri param + sets hint via `hintFor`; hintFor(SCHEMA_VIOLATION on
  schemas.create) → points at `describe --schema-format` (non-circular); pluralization
  helper; `SetEscapeHTML(false)` in output JSON writers; typeDescriptions completeness
  test (`len == len(core.ValueTypes)`).
- Docs sync: concept docs for filters (case-sensitivity), stores/drivers (batch/watch
  semantics), new `docs/concepts/` entries if warranted (dry-run, batch, watch, catalog);
  README + CONTEXT.md sweep for changed shapes.
- E2E style pass: inline flow-map `error:` assertions everywhere; drop doc-path reference
  from type-fidelity description; tighten `exit_nonzero` → exact codes now that classes
  are reliable; verify all 13+new scenarios green via /xdb-e2e.

### Phase 14 (final, cuttable) — `_updated_at` reserved attr (root, all layers) [S9]

- `core.ReservedAttrs`/`IsReservedAttr`; enforcement Apply stamps `_updated_at` TimeVal
  post-check (injectable clock); schema validation + dynamic evolution skip reserved;
  sqlite tableEngine reserved column + lazy ALTER migration; decoder strips incoming
  reserved attrs (import safe); filter carve-out (Phase 5 allowlist) makes it filterable.
  `_rev`/CAS explicitly deferred (design sketch retained in server design).
- Tests: shared tests/record_suite.go (present, increases, not schema-rejected, not
  dynamically inferred) across all 4 drivers; sqlite migration fixture test; decoder strip.
- E2E: extend `type-fidelity`/`crm-pipeline` with `_updated_at` presence + filter.

## Breaking-changes ledger (changelog entries required)

bare `xdb`/unknown cmd output+exit · flag-parse errors → envelope exit 3 · `schemas
delete` requires --force · `--query` removed · `--dry-run` now validates (was silent
write) · divergent create → CONFLICT (was silent success) · config no longer auto-created
by every command · import --file cwd containment · list `-o json` shape
`{items,total,next_offset}` · `daemon status` exit 2 when stopped · filter case-sensitive
+ strict unknown-field rejection · UseNumber default (int-looking values decode as int64)
· record JSON gains `_updated_at` (Phase 14).

## Verification

- Per phase: `make test` red→green, `make check` before commit.
- Shared driver suites (`tests/`) prove sqlite ≡ memory ≡ fs ≡ redis for filter and
  `_updated_at` changes (redis needs `make services-up`).
- E2E: `/xdb-e2e` full suite after each phase that touches scenarios; final run expects
  all scenarios PASS (no skip_if_unimplemented remaining except watch until Phase 11).
- Skill anti-drift: `TestSkillsPayloadsAreValidAgainstServer` + `skills-verbatim` scenario.
- Catalog anti-drift: router↔catalog two-way completeness test.
- Final: re-run the review's three hand-verified probes (dry-run persists?, unknown cmd
  exit?, conflicting create?) — all must show fixed behavior.

## Implementation strategy: Sonnet sub-agents

- Each phase is implemented by a Sonnet sub-agent (`model: sonnet`) with a self-contained
  prompt carrying: the phase's full spec from this plan, the relevant mechanism facts
  (file:line), TDD requirements, and `make test`/`make check` gates.
- Orchestrator (this session) runs phases in dependency order; independent phases run in
  parallel where safe (e.g. Phase 1 root-module vs Phase 3 CLI-harness touch disjoint
  modules; Phases 5/6/7 are disjoint packages after Phase 1 lands).
- After each phase: orchestrator reviews the diff, runs `make test` + `make check`
  itself, commits (one commit per phase, plain capitalized message per project
  convention), then dispatches the next phase.
- Sub-agents get hard rules: TDD (failing test first), no scope creep beyond their phase,
  `make` only, match existing style; report back diffstat + test names added.
- E2E scenario additions ride inside their phase's sub-agent; /xdb-e2e runs at phase
  boundaries that touch scenarios.

## Execution notes

- On approval: copy this plan to `docs/plans/2026-07-22-agent-cli-fixes.md` (project
  convention) before implementation.
- Phases 1-2 (root error taxonomy + catalog) unblock everything; 3-4 (CLI contract) next;
  5-7 (semantics) then 8-11 (features) then 12-13 (surface polish); 14 cuttable.
- The 5 appended e2e scenarios and `docs/plans/2026-07-22-e2e-cli-scenarios.md` are kept
  as-is; consolidation is additive + style alignment (Phase 13) + caveat removal as fixes
  land (Phase 7).
