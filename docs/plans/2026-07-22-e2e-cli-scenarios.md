# E2E Testing: Agent-Driven CLI Scenarios (Types × Domains)

## Implementation Outcome (2026-07-22)

Shipped: 5 new scenarios appended to `tests/e2e/scenarios.yaml` (8 -> 13): `type-fidelity`, `crm-pipeline`, `account-ledger`, `ecommerce-orders`, `trading-day`. Indian context throughout (INR paise, NSE symbols, Indian names/cities). No Go code, no RUNBOOK change. All 5 validated end-to-end against a freshly built binary + sandboxed sqlite daemon (49/49 assertions pass).

The binary probes found these limitations, which shaped the final scenarios:

- Standalone `json`-typed field cannot be populated through the CLI. The xdbjson decoder flattens nested objects into dotted sub-tuples before schema lookup (`encoding/xdbjson/decoder.go:128` `flatten`), and `convertToType` has no JSON case, so an object value hits "unknown field `j.n`" under strict mode and a scalar/array value hits a type mismatch. JSON coverage via the CLI is therefore dropped from `type-fidelity`.

- Object arrays (`array<json>`) are silently dropped (create returns exit 0, field vanishes) because per-member `items` can't be declared through the API request payload (`schemaFieldPayload` lacks `items`: finding #2 below). The final e-commerce scenario uses separate line-item records linked by `order_id`.

- `array<integer>` cannot be populated: JSON integer arrays like `[1,2,3]` infer as `ARRAY<FLOAT>` and fail the type check. `array<string>` and `array<float>` work; `type-fidelity` uses those.

- Unsigned/integer above 2^53 lose precision (float64 intermediate in the JSON pipeline): `9007199254740993` -> `...992`. Scenarios use sub-2^53 values for verbatim round-trip assertions.

Types covered full-stack via the CLI: STRING (incl. unicode), INTEGER (incl. negative), UNSIGNED (<2^53), FLOAT, BOOLEAN, TIME (RFC3339 second precision), BYTES (base64), ARRAY&lt;string&gt;, ARRAY&lt;float&gt;. Not CLI-reachable (documented findings, not tested as passing): standalone JSON, ARRAY&lt;integer&gt;, ARRAY&lt;json&gt; object arrays, unsigned&gt;2^53, schema revision CAS.

## Context

XDB's e2e suite uses YAML scenarios in `tests/e2e/scenarios.yaml` and the instructions in `tests/e2e/RUNBOOK.md`. `/xdb-e2e` runs each scenario with a sandboxed agent, a separate daemon, and a unique `$NS`. The existing eight scenarios cover basic record, schema, and describe commands. Extend the YAML suite to cover value types and workflows for CRM, account ledgers, e-commerce, and stock trading. Go tests and RPC/ORM coverage are deferred at the user's request.

Product findings from planning (verified in code, affect scenario design):

1. Schema revision CAS unreachable via CLI/RPC: `api/schemas.go` `schemaDefPayload` has no `revision` field; `applySchemaPatch` merges onto freshly-fetched def. CAS cannot be a CLI scenario. Track this as a follow-up.

2. Object-array `Items` (per-member schemas for `ARRAY<JSON>`) cannot be declared via CLI/RPC: `schemaFieldPayload` has `type`/`elem_type` but no `items`. Scenarios use plain `ARRAY<JSON>` (unvalidated members). Track this as a follow-up.

3. `records list` exposes `--limit`/`--offset` (`cmd/xdb/cli/records.go:257`) but output prints items only (no total/next_offset): pagination asserted by page contents.

4. `xdb batch` / `xdb watch` are server stubs (`api/batch.go:37`, `api/watch.go:29`): keep avoiding; existing `bulk-import-via-stdin` already models via `skip_if_unimplemented`.

## Changes

Add the scenarios to `tests/e2e/scenarios.yaml` (8 -> 13). RUNBOOK.md and `.claude/commands/xdb-e2e.md` unchanged: existing assertion vocabulary (`exit`, `exit_nonzero`, `stdout_contains`, `stderr_contains`, `stdout_empty`, `json` partial-match, `ndjson_count`, `ndjson_ids`, `error{code,resource,action}`) and templating (`$NS`, `$NOW`) suffice.

### Coverage

- Types (all 9): STRING/TIME everywhere; INTEGER + UNSIGNED (ledger, trading); FLOAT (trading, CRM score); BOOLEAN (CRM, trading); BYTES + JSON + ARRAY (type-fidelity; arrays also CRM tags, e-comm items).

- Modes (all 3): flexible (CRM), strict explicit (ledger), dynamic (trading); default-strict (type-fidelity, e-comm).

- Features: required violations + error envelopes (ledger), nested dotted paths (CRM), patch-vs-upsert semantics (e-comm), filters (CRM/ledger/trading), pagination (trading), import/export (CRM), cascade delete (e-comm), dynamic schema evolution (trading).

### Scenario: `type-fidelity-all-nine`

Schema with all 9 types: `s` string, `i` integer, `u` unsigned, `f` float, `b` boolean, `t` time, `by` bytes, `j` json, `tags` array/elem_type string (+ `nums` array/integer, `objs` array/json).

1. Create the schema.
2. Create a record with every field. Use base64 bytes (`"aGVsbG8="`), an RFC3339 time (`"2024-06-01T10:00:00Z"`), a negative integer, a large unsigned value, and a Unicode string.
3. Run `get -o json` and use `json:` partial matching to assert every value. Bytes and timestamps must round-trip unchanged; use second precision for TIME, as specified by `tests/types_suite.go`.
4. Patch one field with `update` and assert that the other fields remain unchanged.
5. Run `list -o ndjson` and assert the record count.

Fixture context: India. Money in integer paise with `currency: "INR"` (no Decimal builtin: ₹499.00 = `49900`), Indian names/cities (Priya Sharma, Arjun Mehta; Mumbai, Bengaluru, Jaipur), `+91` phones, GSTIN-style fields where natural, NSE symbols (RELIANCE, TCS, INFY), IST-relevant times still stored as RFC3339 UTC.

### Scenario: `crm-pipeline` (Mode: Flexible)

Schema `xdb://$NS/contacts`, `"mode":"flexible"`: `name`/`email` string (email required), `verified` boolean, `score` float, `created_at` time, `tags` array<string>, `address.city` + `address.state` (dotted nested).

1. Create the schema and contacts for Priya Sharma in Mumbai, Arjun Mehta in Bengaluru, and Kavita Rao in Jaipur. Give one contact an undeclared `gstin` field and assert that flexible mode preserves it.
2. Patch Priya's `address.city` to Pune and assert that `address.state` stays unchanged.
3. Run `list --filter 'verified == true && score >= 80.0' --fields _id,name -o ndjson` and check `ndjson_ids`.
4. Export with `xdb export --uri xdb://$NS/contacts -o ndjson` and import into `xdb://$NS/contacts_v2`. Verify whether `xdb import --uri xdb://$NS/contacts_v2 -f -` accepts stdin. If it does not, use a `$(mktemp)` file in the same `run` line.
5. List the target schema and assert that its `ndjson_ids` match the source.
6. Add a `phone` field with `schemas update`. Confirm that old records remain readable and a new record can use `"+91-98200-12345"`.

### Scenario: `account-ledger` (Mode: Strict, Money = Integer Paise, INR)

Schemas: `accounts` {owner string required, balance_paise integer required, currency string required, frozen boolean}; `entries` {account_id string required, delta_paise integer required, kind string, at time, seq unsigned}.

1. Create both schemas and accounts for Priya Sharma and Arjun Mehta. Use `currency: "INR"` and balances in paise, such as `5000000` for ₹50,000.
2. Create debit and credit entries for a ₹1,200 UPI transfer. Use `delta_paise: -120000` and `+120000`, `kind: "upi"`, `at: $NOW`, and a sequence number.
3. Patch both balances with `update`. Assert the new balances and the preservation of other fields.
4. Attempt an account create without `owner`; assert `error{code: SCHEMA_VIOLATION, resource: records, action: create}`. Also assert `SCHEMA_VIOLATION` for a string balance and an undeclared field.
5. Read a missing account and assert `NOT_FOUND`.
6. Filter entries with `--filter 'account_id == "priya" && delta_paise < 0'` and check `ndjson_ids`.

### Scenario: `ecommerce-orders`

Schemas: `products` {sku string required, title string, price_paise integer, stock unsigned, categories array<string>}; `orders` {status string, placed_at time, total_paise integer, items array/elem_type json (plain: Items undeclarable, finding #2)}.

1. Create both schemas and products for "Kanchipuram Silk Saree" (`price_paise: 1299900`) and "Masala Chai 500g" (`49900`). Use categories such as `["sarees","handloom"]`.
2. Create an order with a two-element `items` JSON array. Each item contains `sku`, `qty`, and `price_paise`.
3. Read the order and assert that `items` and `total_paise` round-trip.
4. Upsert the order without an optional field and assert that the field is absent. Compare this with an `update` patch that preserves omitted fields.
5. Patch a product's stock count to decrement it.
6. Run `schemas delete --uri xdb://$NS/orders --cascade`. Assert `ndjson_count: 0` when listing orders and confirm that products remain.

### Scenario: `trading-day` (Mode: Dynamic, NSE)

Schema `ticks`, `"mode":"dynamic"`: `symbol` string, `price` float, `volume` unsigned, `ts` time, `halted` boolean.

1. Create the schema and six ticks for RELIANCE and INFY, with prices around `2985.55` and `1642.10`. Include a low price such as `7.05` to test fractional values.
2. Create a tick with an undeclared `exchange: "NSE"` field. Assert success, then run `schemas get` and check `stdout_contains: "exchange"`.
3. Run `--filter 'symbol == "RELIANCE" && price > 2900.0' -o ndjson` and assert `ndjson_ids`.
4. List with `--limit 3 -o ndjson` and assert `ndjson_count: 3`. List again with `--limit 3 --offset 3` and check the remaining count. The pages must have disjoint IDs and together contain every expected record.
5. Assert that `ts` returns the original RFC3339 value.

## Out of Scope (Deferred)

- RPC (`rpc/client`) and ORM (`xdbstruct`) e2e layers; Go e2e package; real-binary Go smoke test; Makefile target; YAML Go executor; backend matrix (suite runs the daemon's default backend per `xdb init`); CAS and `Items` API gaps (follow-ups); `batch`/`watch`.

## Execution Steps

1. Move this plan to `docs/plans/2026-07-22-e2e-cli-scenarios.md`.

2. Add the 5 scenarios to `tests/e2e/scenarios.yaml` (style-match existing entries; comments where behavior is non-obvious, e.g. base64/RFC3339 wire forms).

3. Verify each `run` line's exact CLI syntax against `cmd/xdb/cli` (flags: `--filter`, `--fields`, `--limit/--offset`, `--cascade`, `--force`, import `-f`): adjust if any assumption breaks (esp. import-from-stdin).

4. Run `/xdb-e2e` (full suite): expect 12 PASS + 1 SKIP (`bulk-import-via-stdin`), `SUITE_PASSED`, clean `git status`.

5. If a scenario exposes a real product bug (e.g. a type that doesn't round-trip), report it: keep the assertion and report the failure.

## Verification

- `/xdb-e2e` full suite green (12 PASS / 1 SKIP).

- Spot-run one scenario solo: `/xdb-e2e trading-day`.

- No repo files changed post-run (`git status --short` clean: orchestrator already enforces).
