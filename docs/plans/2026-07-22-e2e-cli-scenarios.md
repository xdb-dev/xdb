# E2E Testing: Agent-Driven CLI Scenarios (Types × Domains)

## Implementation outcome (2026-07-22)

Shipped: **5 new scenarios** appended to `tests/e2e/scenarios.yaml` (8 → 13): `type-fidelity`, `crm-pipeline`, `account-ledger`, `ecommerce-orders`, `trading-day`. Indian context throughout (INR paise, NSE symbols, Indian names/cities). No Go code, no RUNBOOK change. All 5 validated end-to-end against a freshly built binary + sandboxed sqlite daemon (49/49 assertions pass).

**CLI-surface limitations discovered while probing the real binary** (these shaped the final scenarios — all are genuine product findings worth follow-up issues):
- **Standalone `json`-typed field is unpopulatable via the CLI.** The xdbjson decoder flattens nested objects into dotted sub-tuples before schema lookup (`encoding/xdbjson/decoder.go:128` `flatten`), and `convertToType` has no JSON case — so an object value hits "unknown field `j.n`" under strict mode and a scalar/array value hits a type mismatch. JSON coverage via the CLI is therefore dropped from `type-fidelity`.
- **Object arrays (`array<json>`) are silently dropped** (create returns exit 0, field vanishes) because per-member `items` can't be declared through the API request payload (`schemaFieldPayload` lacks `items` — finding #2 below). E-commerce line items are therefore modeled as **separate linked records keyed by `order_id`**, not an embedded `items` array — the CLI-native pattern.
- **`array<integer>` is unpopulatable** — JSON integer arrays like `[1,2,3]` infer as `ARRAY<FLOAT>` and fail the type check. `array<string>` and `array<float>` work; `type-fidelity` uses those.
- **Unsigned/integer above 2^53 lose precision** (float64 intermediate in the JSON pipeline): `9007199254740993` → `...992`. Scenarios use sub-2^53 values for verbatim round-trip assertions.

Types covered full-stack via the CLI: STRING (incl. unicode), INTEGER (incl. negative), UNSIGNED (<2^53), FLOAT, BOOLEAN, TIME (RFC3339 second precision), BYTES (base64), ARRAY&lt;string&gt;, ARRAY&lt;float&gt;. Not CLI-reachable (documented findings, not tested as passing): standalone JSON, ARRAY&lt;integer&gt;, ARRAY&lt;json&gt; object arrays, unsigned&gt;2^53, schema revision CAS.

## Context

xdb's e2e coverage is the agent-driven CLI suite: `tests/e2e/scenarios.yaml` (8 scenarios) + `tests/e2e/RUNBOOK.md`, run via `/xdb-e2e` (one sandboxed sub-agent per scenario, own daemon via `HOME=$T`, unique `$NS`). Current scenarios touch only records/schemas/describe basics. Goal: extend this suite — **YAML scenarios only, no Go code** — to cover all 9 value types and realistic domain workflows (CRM, account ledger, e-commerce, stock trading). RPC/ORM surface testing: explicitly deferred (scope cut by user).

**Product findings from planning (verified in code, affect scenario design):**
1. Schema revision CAS unreachable via CLI/RPC — `api/schemas.go` `schemaDefPayload` has no `revision` field; `applySchemaPatch` merges onto freshly-fetched def. CAS cannot be a CLI scenario. Follow-up issue material.
2. Object-array `Items` (per-member schemas for `ARRAY<JSON>`) cannot be declared via CLI/RPC — `schemaFieldPayload` has `type`/`elem_type` but no `items`. Scenarios use plain `ARRAY<JSON>` (unvalidated members). Also follow-up material.
3. `records list` exposes `--limit`/`--offset` (`cmd/xdb/cli/records.go:257`) but output prints items only (no total/next_offset) — pagination asserted by page contents.
4. `xdb batch` / `xdb watch` are server stubs (`api/batch.go:37`, `api/watch.go:29`) — keep avoiding; existing `bulk-import-via-stdin` already models via `skip_if_unimplemented`.

## Changes

**One file: `tests/e2e/scenarios.yaml` — add 5 scenarios (8 → 13).** RUNBOOK.md and `.claude/commands/xdb-e2e.md` unchanged — existing assertion vocabulary (`exit`, `exit_nonzero`, `stdout_contains`, `stderr_contains`, `stdout_empty`, `json` partial-match, `ndjson_count`, `ndjson_ids`, `error{code,resource,action}`) and templating (`$NS`, `$NOW`) suffice.

### Coverage union across new scenarios
- Types (all 9): STRING/TIME everywhere; INTEGER + UNSIGNED (ledger, trading); FLOAT (trading, CRM score); BOOLEAN (CRM, trading); BYTES + JSON + ARRAY (type-fidelity; arrays also CRM tags, e-comm items).
- Modes (all 3): flexible (CRM), strict explicit (ledger), dynamic (trading); default-strict (type-fidelity, e-comm).
- Features: required violations + error envelopes (ledger), nested dotted paths (CRM), patch-vs-upsert semantics (e-comm), filters (CRM/ledger/trading), pagination (trading), import/export (CRM), cascade delete (e-comm), dynamic schema evolution (trading).

### Scenario 1: `type-fidelity-all-nine`
Schema with all 9 types: `s` string, `i` integer, `u` unsigned, `f` float, `b` boolean, `t` time, `by` bytes, `j` json, `tags` array/elem_type string (+ `nums` array/integer, `objs` array/json).
Steps (~6): create schema → `records create` with full JSON — bytes as base64 (`"aGVsbG8="`), time as RFC3339 (`"2024-06-01T10:00:00Z"`), negative integer, large unsigned, unicode string → `get -o json` with `json:` partial-match asserting every field **verbatim** (base64 and RFC3339 strings round-trip unchanged; TIME at second precision per `tests/types_suite.go` policy) → `update` patch one field, assert others preserved → `list -o ndjson` count.

**Fixture context: India.** Money in integer paise with `currency: "INR"` (no Decimal builtin — ₹499.00 = `49900`), Indian names/cities (Priya Sharma, Arjun Mehta; Mumbai, Bengaluru, Jaipur), `+91` phones, GSTIN-style fields where natural, NSE symbols (RELIANCE, TCS, INFY), IST-relevant times still stored as RFC3339 UTC.

### Scenario 2: `crm-pipeline` (mode: flexible)
Schema `xdb://$NS/contacts`, `"mode":"flexible"`: `name`/`email` string (email required), `verified` boolean, `score` float, `created_at` time, `tags` array<string>, `address.city` + `address.state` (dotted nested).
Steps (~12): create schema → create 3 contacts (Priya Sharma/Mumbai, Arjun Mehta/Bengaluru, Kavita Rao/Jaipur), one carrying an undeclared extra field like `gstin` (flexible accepts — assert it round-trips) → patch nested `address.city` (Mumbai → Pune), assert sibling `address.state` preserved → `list --filter 'verified == true && score >= 80.0' --fields _id,name -o ndjson` with `ndjson_ids` → export/import migration: `xdb export --uri xdb://$NS/contacts -o ndjson` piped or via `$(mktemp)` in one `run` line into `xdb import --uri xdb://$NS/contacts_v2 -f …` (implementer verifies `-f -` stdin support; else mktemp file in the same shell line) → list target, `ndjson_ids` equality → schema evolution: `schemas update` adds `phone` → old record still readable, new record uses it (`"+91-98200-12345"`).

### Scenario 3: `account-ledger` (mode: strict, money = integer paise, INR)
Schemas: `accounts` {owner string required, balance_paise integer required, currency string required, frozen boolean}; `entries` {account_id string required, delta_paise integer required, kind string, at time, seq unsigned}.
Steps (~12): create both schemas → open two accounts (owners "Priya Sharma", "Arjun Mehta", `currency: "INR"`, balances like `5000000` = ₹50,000) → post debit/credit entry pair — a ₹1,200 UPI transfer as `delta_paise: -120000` / `+120000`, `kind: "upi"`, `at: $NOW`, `seq` → patch both balances via `update`, assert new balances and untouched fields → error paths (the envelope-assertion showcase): create account missing required `owner` → `error{code: SCHEMA_VIOLATION, resource: records, action: create}`; `balance_paise` as string → SCHEMA_VIOLATION; undeclared field on strict schema → SCHEMA_VIOLATION; get missing account → NOT_FOUND → filter entries `--filter 'account_id == "priya" && delta_paise < 0'` → `ndjson_ids`.

### Scenario 4: `ecommerce-orders`
Schemas: `products` {sku string required, title string, price_paise integer, stock unsigned, categories array<string>}; `orders` {status string, placed_at time, total_paise integer, items array/elem_type json (plain — Items undeclarable, finding #2)}.
Steps (~10): create schemas → create 2 products (e.g. "Kanchipuram Silk Saree" `price_paise: 1299900`, "Masala Chai 500g" `49900`, categories like `["sarees","handloom"]`) → create order with 2-item JSON array in `items` (sku/qty/price_paise per member) → get, assert items array and `total_paise` round-trip → upsert order dropping an optional field → assert field absent (full-replace semantics) vs `update` patch preserving → stock decrement patch on product → `schemas delete --uri xdb://$NS/orders --cascade` → order records gone (`list` → `ndjson_count: 0`), products intact.

### Scenario 5: `trading-day` (mode: dynamic, NSE)
Schema `ticks`, `"mode":"dynamic"`: `symbol` string, `price` float, `volume` unsigned, `ts` time, `halted` boolean.
Steps (~11): create schema → create 6 ticks across 2 NSE symbols (RELIANCE ~`2985.55`, INFY ~`1642.10`; one penny-stock float like `7.05` for sub-rupee precision) → dynamic evolution: create tick with new undeclared `exchange: "NSE"` field → succeeds; `schemas get` → `stdout_contains: "exchange"` (schema inferred the field) → filter `--filter 'symbol == "RELIANCE" && price > 2900.0' -o ndjson` → `ndjson_ids` → pagination: `list --limit 3 -o ndjson` → `ndjson_count: 3`; `--limit 3 --offset 3` → remaining count; assert the two pages' `ndjson_ids` are disjoint and union-complete → `ts` returned verbatim RFC3339.

## Out of scope (deferred)
- RPC (`rpc/client`) and ORM (`xdbstruct`) e2e layers; Go e2e package; real-binary Go smoke test; Makefile target; YAML Go executor; backend matrix (suite runs the daemon's default backend per `xdb init`); CAS and `Items` API gaps (follow-ups); `batch`/`watch`.

## Execution steps
1. Move this plan to `docs/plans/2026-07-22-e2e-cli-scenarios.md`.
2. Add the 5 scenarios to `tests/e2e/scenarios.yaml` (style-match existing entries; comments where behavior is non-obvious, e.g. base64/RFC3339 wire forms).
3. Verify each `run` line's exact CLI syntax against `cmd/xdb/cli` (flags: `--filter`, `--fields`, `--limit/--offset`, `--cascade`, `--force`, import `-f`) — adjust if any assumption breaks (esp. import-from-stdin).
4. Run `/xdb-e2e` (full suite): expect 12 PASS + 1 SKIP (`bulk-import-via-stdin`), `SUITE_PASSED`, clean `git status`.
5. If a scenario exposes a real product bug (e.g. a type that doesn't round-trip), report it — don't paper over with a weaker assertion.

## Verification
- `/xdb-e2e` full suite green (12 PASS / 1 SKIP).
- Spot-run one scenario solo: `/xdb-e2e trading-day`.
- No repo files changed post-run (`git status --short` clean — orchestrator already enforces).
