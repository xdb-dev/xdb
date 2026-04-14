# TODOs

Deferred work captured during reviews. Each item has enough context for someone to pick it up in 3 months.

## Server-side `batch.Execute` implementation

**What:** Implement `batch.Execute` in [api/batch.go](api/batch.go), which today returns `"batch.execute not implemented"`.

**Why:** The CLI grammar documents `xdb batch -` with NDJSON streaming, and the client-side plumbing ([cmd/xdb/cli/batch.go](cmd/xdb/cli/batch.go) `normalizeBatchOps`) already accepts both JSON arrays and NDJSON. Today, agents following [CONTEXT.md](cmd/xdb/cli/CONTEXT.md) hit an `INTERNAL` envelope from the stub. A working server closes the documented contract.

**Pros:**
- Closes the docs/runtime drift surfaced in the 2026-04-14 eng review.
- Unlocks `xdb records list ... -o ndjson | xdb batch -` pipelines.
- The CLI side is already tested (`normalizeBatchOps` test coverage in [batch_test.go](cmd/xdb/cli/batch_test.go)) — server work is the last missing piece.

**Cons:**
- Transaction semantics are undefined today. A minimal version (iterate ops, dispatch each, collect per-op results) is easy; a truly transactional batch requires store-level coordination that no backend currently exposes uniformly.
- If implemented naively, a partial failure leaves the store in a mixed state. That is acceptable for a minimal version if documented, but not for agents that assume atomicity.

**Context / current state:**
- [api/batch.go](api/batch.go) — `BatchService.Execute` is a stub.
- The request shape (`ExecuteBatchRequest`) already has `Operations json.RawMessage` + `DryRun bool`, and the response shape has `Results`, `Total`, `Succeeded`, `Failed`, `RolledBack`.
- Each op body mirrors the CLI grammar: `{resource, action, uri, payload}`.
- The client-side [normalizeBatchOps](cmd/xdb/cli/batch.go) converts NDJSON → array before sending.
- A minimal implementation loops the array, dispatches each op to the corresponding per-resource service (`RecordsService.Create`, etc.), and returns one result per op. No rollback required for v1 — document it.

**Depends on / blocked by:** none.

**Suggested starting point:** in [api/batch.go:37](api/batch.go#L37), replace the stub body with a dispatch loop, reusing existing service handlers. Add the router wiring. Write table-driven tests covering success, per-op failure (partial success), and dry-run.
