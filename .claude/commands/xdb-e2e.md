---
description: Run the xdb agent-facing e2e suite in sandboxed temp daemons via parallel sub-agents.
argument-hint: "[scenario-name | scenario,scenario]"
---

Run the XDB end-to-end suite. `$ARGUMENTS` is an optional scenario filter. It can be empty.

## Hard rules, for you and for every sub-agent that you spawn

1. **No edits to anything in the repo.** Do not edit source, `scenarios.yaml`, `RUNBOOK.md`, or the commit history. The only mutable state is `$T=$(mktemp -d)`, one per sub-agent.
2. **Sub-agents are black-box runners.** If a scenario fails because the CLI is broken, that failure is the finding. Do not patch the code. Do not change the YAML to make the test pass.
3. **You (the orchestrator) do not patch either.** If several sub-agents report the same failure, report it once and stop. Do not spawn a "fixer" agent.
4. **The build is the only permitted write.** Run `cd cmd/xdb && go build -o ../../bin/xdb .` from the repo root. `make build` only type-checks and writes no binary. Write nothing else.
5. **Do not skip [tests/e2e/RUNBOOK.md](tests/e2e/RUNBOOK.md).** It contains the same hard rules in agent-facing form. Give it to each sub-agent as the prompt, verbatim. Do not paraphrase or shorten it.

## Steps

1. **Build the binary.** Before you spawn sub-agents, always run `cd cmd/xdb && go build -o ../../bin/xdb .` from the repo root. Do not reuse an existing `bin/xdb`. A stale binary tests old code. If the build fails, stop and report the build error. Do not spawn sub-agents.

2. **Spawn one sub-agent per scenario, in parallel** (one Agent tool call per scenario, all in a single message). Each agent owns its own `HOME=$T`, so daemons cannot collide.

   First read `tests/e2e/scenarios.yaml` to list the scenario names. For each scenario that matches `$ARGUMENTS` (empty = all):
   - `subagent_type: "general-purpose"`
   - `model: "haiku"`. The runner is mechanical: read YAML, run bash, compare outputs.
   - `description: "xdb e2e: <scenario-name>"`
   - `prompt`: the **full contents** of `tests/e2e/RUNBOOK.md`, followed by:
     ```
     Repo root: <absolute path>
     xdb binary: <absolute path>/bin/xdb
     Scenario filter: <single-scenario-name>
     ```

3. **Single-scenario invocation.** If `$ARGUMENTS` names exactly one scenario, spawn only that one sub-agent.

4. **Cleanup verification.** After all sub-agents finish:
   - Run `git status --short`. `bin/` is ignored by git. If any file appears (in particular under `cmd/`, `api/`, `rpc/`, `core/`, `store/`, `tests/e2e/scenarios.yaml`, `tests/e2e/RUNBOOK.md`), a sub-agent overstepped. Report which files and stop. **Do not revert** without telling the user.
   - Run `pgrep -f "/tmp.*xdb.*daemon\|xdb-test.*daemon"`. If a stray daemon is still running, list it. Ask the user before you kill it.

5. **Report.** Each sub-agent ends its message with the literal line `SUITE_PASSED` or `SUITE_FAILED`. Merge all sub-agent tables into one Markdown table. Print only:
   - the table
   - the summary line, in the RUNBOOK order, for example `24 scenarios: 23 PASS, 1 FAIL`
   - any cleanup-verification anomalies from step 4
   - the final line: `SUITE_FAILED` if any sub-agent ended with `SUITE_FAILED` or ended without a marker, else `SUITE_PASSED`

   Suppress build output and per-agent reasoning. Do not propose fixes for failing scenarios in this message. That is a separate task that the user can request.
