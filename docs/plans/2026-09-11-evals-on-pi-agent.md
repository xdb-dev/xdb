# Move the evals onto pi.Agent

status: approved 2026-09-11

## What changes

`evals/` becomes `internal/evals/`, one flat Go module of `package main`. The
subject agent runs in process on `pi.Agent` against OpenRouter. The
`claude -p --output-format stream-json` subprocess goes away, and the code
that parsed it goes with it.

The disclosure metrics do not change. They are the value of this harness.
Grading stays deterministic, but the check vocabulary is replaced by the
shell.

## Reason

Three files exist only to drive and read a subprocess: `runner.go`,
`trajectory.go`, and the transport half of `judge.go`. They encode the
stream-json record shape of one Claude Code version. That shape is not a
contract, and a release can break the harness with no warning.

`pi.Agent` gives the same facts as typed events: tool calls, tool results,
token usage, and cost. The harness reads them instead of parsing them.

## Layout

```
internal/evals/
  go.mod              module github.com/xdb-dev/xdb/internal/evals
  main.go             flags and exit codes (was cmd/xdb-eval/main.go)
  agent.go            catalog, OpenRouter, bash tool, budget hook (was runner.go)
  trace.go            agent.Event -> Call and totals (was trajectory.go)
  judge.go            pi.GenerateObject and the evidence builder
  sandbox.go          fixtures, fake HOME, xdb binary, daemon
  run.go              phases, grading, results
  task.go             task.yaml
  checks.go           CLI and answer assertions
  classify.go         xdb call -> resource, action, disclosure layer
  metrics.go          friction and disclosure metrics
  report.go           the results table
  README.md
  tasks/              the four tasks, with rewritten checks
  testdata/
```

`internal/evals` imports no xdb package. It drives the built binary through a
shell. So the module needs no `replace` for the root module. The Makefile
finds it already, because `SUBMODULES` matches every `go.mod` below depth 2.

`package main` in one directory is the whole point of the flattening. There is
no library here and no second consumer.

## The code changes

### 1. The subject runs in process

`agent.go` registers one OpenRouter provider in a `catalog.Catalog`, with the
real OpenRouter slug and the real cost per model:

```go
p := openairesponses.NewForOpenRouter(
    option.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    option.WithBaseURL("https://openrouter.ai/api/v1"),
)
cat := catalog.New()
cat.RegisterTextProvider("openrouter", p, models...)
a, err := cat.Agent("openrouter/"+slug, pi.WithSystemPrompt(sys), pi.WithTools(bashTool), ...)
```

Auto-detection is not used. It registers OpenRouter under the OpenAI model
list, so an OpenRouter slug does not resolve. A model with no `ai.Cost` also
reports a cost of zero, which breaks the cost budget.

`models` is a small table: the slug, the context window, and the input and
output price. It starts with `anthropic/claude-haiku-4.5` and
`anthropic/claude-sonnet-4.6`. Add a row to add a model.

### 2. One agent for every phase

A task is a conversation. `pi.Agent` keeps the history in the object, so phase
two is a second `Run` on the same agent. The `--resume` flag, the session id,
and the field that carried it through `PhaseRequest`, `Trajectory`, and
`run.go` all go away.

### 3. The bash tool is the sandbox runner

`pi`'s bash tool inherits `os.Environ()`, so it cannot take a per-agent `HOME`
or `PATH`. The harness defines its own, over the `sb.Run` that already exists:

```go
ai.DefineTool(bash.ToolName, bash.Description, func(ctx context.Context, in bash.Input) (string, error) {
    out, err := sb.Run(ctx, in.Command)
    ...
})
```

The tool description is the one embedded in `pi`'s bash package, so the subject
reads the same instructions.

This removes the shim. Today the sandbox writes `bin/xdb` as a wrapper that
re-points `HOME` at `home/`, because the `claude` subprocess needed the real
`HOME` for its own credentials. In process there is no such subprocess. The
sandbox copies the binary and sets `HOME` and `PATH` in the command
environment, and `libexec/` is deleted.

### 4. The trace replaces the trajectory

`trace.go` reads the event stream once and builds the same `Call{Command,
Output, Failed}` list that `classify.go` and `metrics.go` already consume:

- `EventToolExecutionStart` gives the tool name and the arguments.
- `EventToolExecutionEnd` gives the result and `IsError`.
- `EventAgentEnd` gives the accumulated `ai.Usage`.
- `EventTurnStart` counts turns.

`Subtype` is replaced by two booleans the harness computes. `pi` has no
max-turns stop reason, so a truncated run is a turn count equal to the limit
with a last stop reason of `tool_use`. An aborted run is an error out of
`Wait`.

The raw event stream is written to the results directory as NDJSON, because
`agent.Event` marshals itself. The results directory keeps the same shape.

### 5. The judge is one call

`pi.GenerateObject[verdict]` takes the rubric and the evidence and returns a
typed result. The hand-written JSON schema string, the subprocess, and the
output parser are deleted. `BuildEvidence` is unchanged.

### 6. The budget is a hook

`pi` has no cost limit. `agent.WithAfterTurn` sees the usage of each turn and
returns an error to stop the run. That is the replacement for
`--max-budget-usd`, and it is about ten lines.

### 7. A check is a shell command

`Expect` has nine verbs. The four tasks use two of them: `exit: 0` and
`ndjson_count`. The other seven are 237 lines that reimplement the shell.

A check becomes a command. Exit code 0 passes. A count is `wc -l` and `test`:

```yaml
- name: fifteen transactions
  run: test "$(xdb records list xdb://$NS/txns -o ndjson --limit 100 | wc -l)" -eq 15
```

`checks.go` keeps the exit code and the `ANSWER:` line reader, and loses the
rest. The `Expect` struct, its loader validation, and their tests go with it.

The check result carries the captured stdout and stderr, because an exit code
alone does not say what went wrong.

Grading stays deterministic. An LLM does not run the checks. The five product
faults of 2026-09-11 were findable because a check failed the same way every
time. A score of "3 out of 5 on querying" is not a finding.

### 8. The task model names an OpenRouter slug

`model: sonnet` in `task.yaml` becomes `model: anthropic/claude-sonnet-4.6`.
The `-model` flag takes the same form. The four task files change one line
each.

### 9. The report is one table

`-repeat` runs a task N times and `report.go` reports the median of seven
metrics across those runs. Every run so far was one attempt. Nothing has read
a median.

Both go. `report.go` becomes a `text/tabwriter` loop over `[]*TaskResult`,
and `summary.json` is that slice. `TaskSummary`, `summarizeTask`, `medianInt`,
`medianFloat`, the repeat loop, and the `-N` results suffix are deleted.

To compare runs, run the command more than once and read the result files:

```bash
for i in 1 2 3; do make evals RESULTS=/tmp/eval-$i; done
```

### 10. Six counters go

`Disclosure` has ten counters. Four earned their place. `BlindFailures` and
`DiscoveryCalls` are budgets. `DeepestLayer` told the difference between the
haiku run and the sonnet run. `HintedFailures` and `HintFollowed` produced
finding 3.

`Path`, `LayersVisited`, `DiscoveryRecoveries`, `BlindRecoveries`,
`UnknownFlagErrors`, and `RedundantDiscovery` appear in no finding and no
report column. `Path` also duplicates the raw event log at lower fidelity.
They are deleted, with `findRecovery`'s second return value and the
`DiscoveryStep` type.

### 11. Two smaller deletions

`StrayProcesses` runs `pgrep` after teardown. It guarded against a `claude`
subprocess that outlived the run. In process there is no such subprocess, and
`sb.Close()` stops the daemon.

`selectTasks` splits a comma list and builds its own unknown-name error. A
`slices.Contains` filter over the loaded tasks replaces it.

## What does not change

`classify.go` is not touched. Its shell tokenizer exists because a check
command carries a quoted CEL filter with spaces and a `&&` separator, and its
layer tables are the disclosure model. Every line of it maps to a real shape
of the CLI. It is the measurement, not overhead.

`metrics.go` keeps the four counters that budgets and findings used, and the
recovery rate.

The judge keeps its fixed `BuildEvidence`, so rubric scores stay comparable
between runs. `Runner` and `Judge` stay interfaces, because the tests use
in-memory fakes and an interface is cheaper than a scripted provider.

## Size

About 4200 lines today. About 2650 after, a cut of about a third.

Four sources, in order of size:

| Deleted                                  | Lines |
| ---------------------------------------- | ----: |
| the subprocess plumbing                  |  ~480 |
| the check vocabulary                     |  ~460 |
| `-repeat` and the medians                |  ~230 |
| six unused counters, pgrep, selectTasks  |  ~185 |

The subprocess plumbing is the part that breaks on someone else's release.
The other three are code with no consumer.

`classify.go` and the counters that budgets read are about 500 lines and are
not touched, because that is the harness.

## Steps

1. `git mv evals internal/evals`, then `git mv internal/evals/cmd/xdb-eval/main.go internal/evals/main.go` and remove the empty `cmd` tree. Every file becomes `package main`. Delete `doc.go`.
2. Write `internal/evals/go.mod` and `go get github.com/sonnes/pi-go@main`. Run `make build`.
3. Update `Makefile`, `.gitignore`, `CLAUDE.md`, and the README for the new paths.
4. Write `agent.go`: the catalog, the model table, the bash tool, and the budget hook. Its test asserts the model resolves and the budget hook stops a run.
5. Write `trace.go` against a recorded event slice. Its test asserts the same `Call` list that `trajectory_test.go` asserted, so `classify.go` and `metrics.go` keep passing untouched.
6. Cut `run.go` over to the new agent and drop the session id.
7. Cut `judge.go` over to `pi.GenerateObject`.
8. Strip the shim from `sandbox.go`.
9. Cut the checks to an exit code. Rewrite the 40 `run:` checks of the four tasks with `test` and `wc`. Delete `Expect` and its tests.
10. Cut the six counters from `metrics.go`, and `StrayProcesses` from `sandbox.go` and `run.go`.
11. Rewrite `report.go` as a tabwriter table. Drop `-repeat` and `selectTasks` from `main.go`.
12. Delete `runner.go` and `trajectory.go`.
13. `make check` and `make test`. Then one real task against OpenRouter with haiku, to confirm the cost, the turn count, and the disclosure metrics are non-zero and sane.

## Decisions

- The Claude Code subject is dropped, not kept alongside. Two subjects cost more code than today, and the request is to run on OpenRouter.
- No `replace` directive. `internal/evals` imports nothing from the root module.
- A counter stays only if a budget reads it or a finding cited it. That test
  keeps four of ten and deletes the rest.
- `classify.go` stays whole. It is the one file where the line count is the
  measurement, and cutting it would cost findings.
- Grading stays deterministic. An LLM judge scores the data model against the
  rubric. It does not decide PASS or FAIL.

## Out of scope

Process isolation. `pi`'s sandbox package confines paths, not processes, and it
is not applied to a bash tool. The current fake `HOME` and temporary directory
are what the harness has, before and after.
