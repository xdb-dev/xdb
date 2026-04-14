---
description: Run the xdb agent-facing e2e suite in sandboxed temp daemons via parallel sub-agents.
argument-hint: "[scenario-name | scenario,scenario]"
---

Run the xdb end-to-end suite. `$ARGUMENTS` (may be empty) is an optional scenario filter.

## Hard rules — for you AND every sub-agent you spawn

1. **No edits to anything in the repo.** Not source, not `scenarios.yaml`, not `RUNBOOK.md`, not commit history. The only mutable state is `$T=$(mktemp -d)` per sub-agent.
2. **Sub-agents are black-box runners.** If a scenario fails because the CLI is broken, that is the *finding*. Do not patch the code. Do not modify the YAML to make the test pass.
3. **You (the orchestrator) do not patch either.** If multiple sub-agents report the same failure, surface it once and stop. Do not spawn a "fixer" agent.
4. **Build is the only allowed write.** `make build` (or `go build -o bin/xdb ./cmd/xdb` from the cmd module if `make build` doesn't produce a binary). Nothing else.
5. **Do not skip [tests/e2e/RUNBOOK.md](tests/e2e/RUNBOOK.md).** It contains the same hard rules in agent-facing form. Spawn sub-agents with it as the prompt verbatim — do not paraphrase or shorten.

## Steps

1. **Build if missing.** If `bin/xdb` does not exist, build it. If it exists, skip and tell the user.

2. **Spawn one sub-agent per scenario, in parallel** (one Agent tool call per scenario, all in a single message). Each agent owns its own `HOME=$T` so daemons cannot collide.

   Read `tests/e2e/scenarios.yaml` first to enumerate scenario names. For each scenario matching `$ARGUMENTS` (empty = all):
   - `subagent_type: "general-purpose"`
   - `model: "haiku"` — runner is mechanical: read YAML, exec bash, diff outputs.
   - `description: "xdb e2e: <scenario-name>"`
   - `prompt`: the **full contents** of `tests/e2e/RUNBOOK.md` followed by:
     ```
     Repo root: <absolute path>
     xdb binary: <absolute path>/bin/xdb
     Scenario filter: <single-scenario-name>

     REMINDER: black-box runner only. No edits to any file in the repo. CLI bugs are FAILs to report, not problems to solve. Violating this is itself a test failure.
     ```

3. **Single-scenario invocation.** If `$ARGUMENTS` names exactly one scenario, spawn just that one sub-agent (no fan-out).

4. **Cleanup verification.** After all sub-agents finish:
   - Run `git status --short`. If anything beyond untracked temp files appears (especially under `cmd/`, `api/`, `rpc/`, `core/`, `store/`, `tests/e2e/scenarios.yaml`, `tests/e2e/RUNBOOK.md`), a sub-agent overstepped — report which files and stop. **Do not auto-revert** without telling the user.
   - Run `pgrep -f "/tmp.*xdb.*daemon\|xdb-test.*daemon"`. If any stray daemon is still running, list it and ask the user before killing.

5. **Report.** Merge all sub-agent reports into one markdown table. Print only:
   - the table
   - the summary line (e.g. `8 scenarios: 5 PASS, 1 SKIP, 2 FAIL`)
   - any cleanup-verification anomalies from step 4
   - final `**suite passed**` or `**suite failed**`

   Suppress build chatter and per-agent reasoning. Do not propose fixes for failing scenarios in this message — that's a separate task the user can request.
