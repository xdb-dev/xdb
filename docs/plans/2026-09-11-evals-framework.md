# Evals: Agent Task Evaluations for the XDB CLI

Status: approved 2026-09-11. Replaces `tests/e2e/`, `.claude/commands/xdb-e2e.md`, and the `xdb-e2e` skill.

## Reason to replace the e2e suite

The e2e suite scripts every command and asserts on its output. It proves that the CLI does what a scripted caller tells it to do. It does not test the claim in the README, that an agent gets XDB right on the first try. An LLM runs the suite, but the LLM adds nothing. It reads YAML and runs bash. That is slower, more expensive, and less reliable than a Go test.

An eval swaps the roles. The agent receives a task the way a user states it. The harness is deterministic Go. It builds the sandbox, runs the agent, inspects the store, and measures friction. A failed eval is a product finding. The CLI, its help, or its errors did not get the agent to the goal.

## Measurements

Each task reports these values.

1. Outcome. The final store state and the answers of the agent match the spec. This is the pass or fail gate.
2. Friction. Failed commands, retries, discovery calls, turns, wall time, cost. These are numbers. A task can set budgets that turn a number into a fail.
3. Design quality. Types, required fields, modes, relations. An LLM judge scores a rubric. Off by default.

## Roles

- Subject. The agent under evaluation. It runs headless with `claude -p`. It sees only the sandbox: the `xdb` binary on PATH, the fixture files, and the task prompt. It does not see the repo.
- Harness. The Go program `evals/cmd/xdb-eval`. It builds the binary, makes the sandbox, drives the phases, runs the checks, computes the metrics, and writes the results.
- Judge. A second headless run that reads the rubric, the final schemas, and the trajectory. Optional.

## Task anatomy

```
evals/tasks/household-ledger/
  task.yaml       # metadata, phases, checks, budgets, rubric
  fixtures/       # the harness copies these files into the work directory of the agent
```

```yaml
name: household-ledger
description: Model and run a household ledger from two bank statements.
tags: [ledger, money, multi-schema]
namespace: household   # fixed in the prompt so that checks can find the data
model: sonnet
max_turns: 60
budget:                # optional. If a run exceeds a value, the task fails
  failed_commands: 8
  cost_usd: 2.00
phases:
  - name: setup
    prompt: |
      I want to track my household finances in xdb. ...
    checks:
      - name: accounts schema exists
        run: xdb schemas get xdb://household/accounts -o json
        expect: { exit: 0 }
  - name: month-end
    prompt: |
      Load the two statements in this directory. ...
      Finish your reply with one line: `ANSWER: <HDFC balance in paise>`.
    checks:
      - name: hdfc balance
        answer: "4880000"
      - name: twelve transactions
        run: xdb records list xdb://household/transactions -o ndjson --page-all
        expect: { ndjson_count: 12 }
rubric: |
  Score 1 to 5: money is stored as integers, not floats. ...
```

### Naming contract

The prompt names the namespace and the domain nouns: accounts, transactions, sources, issues. It does not name fields, types, flags, or commands. The agent designs the shape. Checks find data by the names in the prompt.

The `answer` check does not depend on the shape. It grades what the agent can read back out of its own store. Prefer `answer` checks. Use `run` checks for facts that an answer cannot carry, such as record counts and schema existence.

### Phases

One task is one conversation with several turns. For each phase, the harness sends the prompt, runs the subject until it stops, and runs the checks against the live daemon. Later phases add curveballs: a wrong amount, a duplicate import, a schema change, a stale `_version`. This is the "use" half of the task.

A failed phase stops the task. The report marks the later phases as `NOT_RUN`.

### System prompt and task prompt

The system prompt carries everything about xdb. The task prompt carries only the task, the way a user would state it.

The system prompt has these parts, in this order:

1. The sandbox facts: the working directory, the fixture files, "do not ask questions, decide and state the decision", and "end your reply with the `ANSWER:` line when a task asks for one".
2. The output of `xdb context`, verbatim, from the sandbox binary at run time. The harness does not paste a stale copy.
3. Nothing else. No skills, no `describe` output, no examples beyond what `xdb context` holds.

This is how an agent meets xdb in practice. It reads the context guide once. Then it must find the rest through the CLI.

### Progressive disclosure

The context guide is layer 0. The CLI shows deeper layers on demand. The eval measures whether the agent finds the next layer when it needs it. It also measures whether the layers hold what the agent needs.

| Layer | Surface                                                          |
| ----- | ---------------------------------------------------------------- |
| 0     | `xdb context` (in the system prompt)                             |
| 1     | `xdb --help`, `xdb <resource> --help`, `xdb describe --actions`  |
| 2     | `xdb <resource> <action> --help`, `xdb describe <resource>.<action>` |
| 3     | `xdb describe --uri`, `--filter`, `--errors`, `--value-types`, `<Type>` |
| 4     | `xdb skills`, `xdb skills get <name>`                            |

For each task the harness reports a `disclosure` block.

| Field                  | Meaning                                                                 |
| ---------------------- | ----------------------------------------------------------------------- |
| `path`                 | The ordered list of discovery calls, each tagged with its layer.        |
| `layers_visited`       | The set of layers in `path`.                                            |
| `deepest_layer`        | The highest layer reached.                                              |
| `blind_failures`       | Failed xdb calls on an action that the agent did not look up before the call. |
| `discovery_recoveries` | A failed call, then a discovery call, then a successful call on the same action. The docs fixed it. |
| `blind_recoveries`     | A failed call, then a successful call with no discovery between. The error hint or a guess fixed it. |
| `unknown_flag_errors`  | Failures with `flag provided but not defined`. The agent invented a flag that layer 0 did not exclude. |
| `hint_followed`        | Failures where the next xdb call contains the command from the error `hint`. |
| `redundant_discovery`  | Discovery calls on an action that `xdb context` already shows. The agent did not trust layer 0, or layer 0 was not clear. |

Many `blind_failures` with few `discovery_recoveries` means that the agent guesses instead of reading, or that the errors do not point at the docs. Many `discovery_recoveries` at layer 4 means that layers 0 to 3 lack something that the skill has. Many `redundant_discovery` calls mean that the context guide is not clear on that action. A high `hint_followed` rate means that the error envelopes help.

A budget can limit any counter, for example `budget.blind_failures: 3`. The optional judge rubric adds one question: "Name each moment where the agent lacked a fact that a deeper layer holds, and name the layer."

### Check vocabulary

The `run` checks use the e2e assertions, verbatim: `exit`, `exit_nonzero`, `stdout_contains`, `stderr_contains`, `stdout_empty`, `json`, `ndjson_count`, `ndjson_ids`, and `error`. The harness runs the command through the sandbox shim. Then the check sees the same daemon as the agent.

Two new assertions grade the reply: `answer` (exact match on the `ANSWER:` line, whitespace trimmed) and `answer_regex`.

## Sandbox

```
$T = /tmp/xdb-eval-<task>-<rand>/
  home/            # HOME for xdb only. ~/.xdb is here
  libexec/xdb      # copy of bin/xdb
  bin/xdb          # shim: exec env HOME=$T/home $T/libexec/xdb "$@"
  work/            # agent cwd. The fixtures are copied here
```

The subject process keeps the real HOME, so `claude` can find its credentials. Only xdb sees the fake HOME, through the shim. The binary is a copy, so the repo path does not appear in the environment of the agent. `PATH=$T/bin:$PATH`. The root is under `/tmp`, because nested temp paths exceed the Unix socket path limit on macOS.

The harness runs `xdb init` before phase 1. Daemon setup is not graded.

The subject invocation:

```
claude -p --model $model \
  --output-format stream-json --verbose \
  --allowedTools Bash,Read,Write \
  --max-turns $max_turns --max-budget-usd $budget \
  --setting-sources "" --disable-slash-commands \
  --permission-mode bypassPermissions \
  --append-system-prompt "$sandbox_prompt" \
  [--resume $session_id] \
  "$phase_prompt"
```

`--setting-sources ""` removes the user and project CLAUDE.md, hooks, and skills. The subject has no memory of xdb from the setup of the user. The session id comes from the `init` record of phase 1. Later phases pass it to `--resume`.

The `--append-system-prompt` value is the system prompt described under "System prompt and task prompt". The harness builds it per run from the sandbox binary.

## Trajectory and metrics

The harness parses the stream-json output. These facts were tested against Claude Code 2.1.263 on 2026-09-11:

- `assistant` records carry `tool_use` blocks with `id`, `name`, and `input.command`.
- `user` records carry `tool_result` blocks with `tool_use_id`, `is_error`, and the output. A failed command has `is_error: true` and the text `Exit code N`. Results can arrive out of order. Link them by `tool_use_id`.
- The `result` record carries `subtype` (`success`, `error_max_turns`, and more), `num_turns`, `duration_ms`, `total_cost_usd`, `session_id`, `usage.input_tokens`, `usage.output_tokens`, and the final text in `result`.

For each Bash call the harness classifies the command.

| Class       | Rule                                                         |
| ----------- | ------------------------------------------------------------ |
| `xdb`       | The command contains an `xdb` token.                         |
| `discovery` | An `xdb` call with `--help`, `-h`, `describe`, `context`, or `skills`. Tagged with its disclosure layer (see above). |
| `failed`    | `is_error` is true.                                          |
| `recovered` | A failed `xdb` call, then a successful call with the same `<resource> <action>` within the next two `xdb` calls. |

Metrics per phase and per task: `turns`, `tool_calls`, `xdb_calls`, `discovery_calls`, `failed_calls`, `recovered_calls`, `recovery_rate`, `duration_s`, `cost_usd`, `input_tokens`, `output_tokens`, and the `disclosure` block. If `result.subtype` is not `success`, the phase fails with that subtype as the reason.

## Results

```
evals/results/<run-id>/          # git ignores this. run-id = 20260911-143210
  summary.md
  summary.json
  <task>/
    trajectory.<phase>.jsonl
    checks.json
    metrics.json
```

Summary columns: task, result, checks passed, turns, xdb calls, failed, blind failures, deepest layer, recovery rate, cost. With `--repeat N`, the harness runs each task N times and reports the pass rate and the medians.

## Make targets

```
make evals                        # all tasks, one run each
make evals TASK=household-ledger
make evals REPEAT=3
make evals RUBRIC=1
```

`make evals` builds `bin/xdb` and then runs `go run ./evals/cmd/xdb-eval`. The e2e build exception in CLAUDE.md is removed. The target needs `claude` on PATH.

## Hard rules

These rules come from the e2e runbook. The harness enforces them in code.

- The subject does not see the repo. The binary is a copy, and the cwd is under `$T`.
- The harness does not patch. A failure is the finding.
- After a run, the harness runs `git status --short` and `pgrep -af "$T"`. It reports a dirty tree or a stray daemon. It does not revert or kill.
- Teardown always runs: `xdb daemon stop` through the shim, then a copy of the results, then `rm -rf "$T"`.

## Decisions

- A Go harness instead of a slash command. A Go program is deterministic and can compute metrics. A slash command cannot see the trajectory.
- `claude -p` is the only runner in the first version. `Runner` is a small interface: run one phase, return a trajectory. A Codex runner can come later.
- Checks go through the CLI and not through the store package. The grader is a black box, like the agent.
- The context guide is always in the system prompt. Without it, the eval tests a situation that xdb does not put an agent in. The disclosure metrics start at layer 1.
- The default subject model is `sonnet`. `haiku` is for cheap smoke runs. Pass `--model` to compare.

## The four tasks

The framework comes first. The outline below fixes what each task measures.

1. `household-ledger`. Fixtures: two bank statement CSVs, HDFC savings and ICICI credit card, in INR. Phases: model accounts and transactions and load the statements, then categorize and report a month total, then fix a duplicate import and a wrong amount, then add a currency field and record a USD transaction. Measures: integer money, filters, update against upsert, schema evolution.
2. `research-citations`. Fixtures: a JSON list of sources with authors, year, URL, and notes. Phases: model sources, claims, and citations, then load and answer which sources support a claim, then mark a source retracted and list the claims that lost support, then export a bibliography sorted by year. Measures: relations by id, nested attributes, string arrays, export.
3. `ecommerce-store`. Fixtures: a product catalog CSV and an order stream in NDJSON. Phases: model products, inventory, and orders, then place orders, decrement stock, and reject an order over stock, then resolve a stale `_version` write, then cascade delete a discontinued category. Measures: batch, versioning, cascade.
4. `issue-tracker`. Fixtures: a GitHub-style issue dump in JSON. Phases: model projects, issues, labels, and comments, then triage with assign, label, and close, then run query workflows with filters, field masks, and pages, then bulk relabel with batch. Measures: filters, pagination, batch, dynamic mode.

## Implementation steps

Write the test first at every step. Use `testify` and table-driven tests.

1. `evals/task.go`. Parse and validate `task.yaml`: a name, at least one phase, a known check vocabulary. Tests use a fixture task.
2. `evals/sandbox.go`. Make `$T`, write the shim, copy the binary and fixtures, run `xdb init`, tear down. The test skips when `bin/xdb` is missing.
3. `evals/trajectory.go`. Parse stream-json, link results to calls, classify commands, tag discovery calls with their layer, and compute the metrics and the `disclosure` block. Tests use recorded JSONL. The smoke run at `/tmp/xdb-eval-smoke/out.jsonl` is the first fixture.
4. `evals/checks.go`. The assertion vocabulary, ported from the runbook, plus `answer` and `answer_regex`.
5. `evals/runner.go`. The phase loop, the `claude -p` exec, `--resume`, budgets. Tests put a fake `claude` script on PATH that emits canned JSONL.
6. `evals/report.go`. Write `summary.md` and `summary.json`. Golden tests.
7. `evals/cmd/xdb-eval/main.go`, the `evals` Make target, `evals/results/` in `.gitignore`, `evals/README.md`.
8. The first task, `household-ledger`. Run it three times. Adjust the prompt and the checks until the checks fail only on product bugs.
9. The other three tasks.
10. Delete `tests/e2e/`, `.claude/commands/xdb-e2e.md`, and the `xdb-e2e` skill. Update CLAUDE.md: the build exception and the project structure. Audit the 24 e2e scenarios. Move each contract assertion that the `cmd/xdb/cli` integration tests do not already cover into those tests.

Estimate: steps 1 to 7 take about one day. Each task takes about half a day with adjustments. Step 10 takes about half a day.

## Out of scope

- A comparison of two runs (`xdb-eval compare`). Add it after two real runs exist.
- A Codex runner.
- Evals in CI. Each run costs money and takes minutes.
- The `setup: agent` knob, where the subject runs `xdb init` itself.

## Implementation Outcome (2026-09-11)

Shipped: the `evals` package (task loader, sandbox, Claude runner, trajectory parser, command classifier, metrics with the disclosure block, checks, LLM judge, report), the `xdb-eval` command, `make evals`, four tasks, and `evals/README.md`. `tests/e2e/`, `.claude/commands/xdb-e2e.md`, and the CLAUDE.md build exception are removed. The e2e contract assertions that had no Go test are now in `cmd/xdb/cli/records_contract_test.go`, `schemas_contract_test.go`, and `pipes_contract_test.go`.

First runs, one attempt each:

| Task               | Model  | Result | Checks | Turns | xdb calls | Failed | Blind | Deepest | Cost  | Rubric |
| ------------------ | ------ | ------ | ------ | ----- | --------- | ------ | ----- | ------- | ----- | ------ |
| household-ledger   | haiku  | PASS   | 14/14  | 17    | 10        | 1      | 1     | -       | $0.21 | -      |
| household-ledger   | sonnet | PASS   | 15/15  | 27    | 19        | 2      | 1     | L4      | $0.60 | 5.0/5  |
| ecommerce-store    | haiku  | PASS   | 12/12  | 19    | 9         | 3      | 3     | -       | $0.15 | -      |
| issue-tracker      | haiku  | PASS   | 11/11  | 32    | 25        | 3      | 3     | L3      | $0.20 | -      |
| research-citations | haiku  | FAIL   | 7/8    | 16    | 12        | 2      | 2     | L3      | $0.08 | -      |

The research-citations failure was a shell bug of the agent. An awk backreference, which awk does not support, collapsed the citation ids. The agent stored 7 citations instead of 10. The task is correct.

Haiku stored money and dates as strings in the ledger and passed every count-based check. That is why the judge exists, and why the ledger got a fifth phase. The question "which INR transactions were over ₹5,000" gives a wrong count when money is a string. Sonnet read layers 3 and 4 before its first write. It modeled the ledger with integer paise, time fields, and required fields.

Product findings from these runs. All five are now fixed, with a regression test each.

1. Time-range filters could not be expressed from the CLI on sqlite. `date >= timestamp("2026-08-01T00:00:00Z")` failed with `INTERNAL` "[xdb/sqlgen] unsupported function: timestamp". A ledger "by month" had no filter. `filter/sqlgen` now folds a constant `timestamp()` call and binds it as milliseconds, the form the sqlite driver stores. The conformance suite covers a month range on every backend.
2. Input errors came back as `INTERNAL`, exit 4. An agent reads exit 4 as a server bug. A JSON payload that does not decode now maps to `INVALID_ARGUMENT` with `reason: invalid_payload`. A sqlite filter fault splits two ways: a valid CEL construct with no SQL form, such as `matches()`, falls back to a scan, and any other fault reaches the caller as `INVALID_ARGUMENT`.
3. The `INVALID_ARGUMENT` hint was the placeholder "run xdb describe <resource>.<action>". It now names the action that failed. `filter.Compile` also tags every rejection with `fix`, so a filter error points at `xdb describe --filter`.
4. `xdb export | xdb import` into another schema failed with `CONFLICT` on line 1. Export emits `_version`, and a write reads that field as a compare-and-swap precondition. Import now drops the stamp, because an import moves data and does not compare and swap. A failed precondition is also tagged with the two versions and with an accurate fix: a stored version of 0 says the record is absent. `TestExportImport_Roundtrip` is no longer skipped.
5. There was no filter for an absent field. The CLI rejected `!has(assignee)`. XDB now replaces the standard CEL `has()` macro, because an XDB attribute name is flat and a dotted name is one attribute, not a field of a nested message. `has(attr)` expands to a membership test on the new reserved `_attrs` list. It compiles to `IS NOT NULL` on a column table and to an attribute-row test on a KV table.

`xdb describe --filter` now lists `timestamp()`, `has()`, `matches()`, and the reserved attributes, with an example of each. The `query-and-filter` skill and `docs/concepts/filters.md` carry the same two recipes: a time range and an absent field.

Not shipped, by design: a comparison of two runs, a Codex runner, CI, and the `setup: agent` knob. The stdout of the `xdb watch` command and the sqlite-only `UNIQUE_VIOLATION` envelope have no CLI test. The RPC stream and the driver have tests.
