# Evals

Agent task evaluations for the xdb CLI. An eval gives an agent a real-world
task in a sandbox, in phases. The harness grades the result and measures
friction. A failed eval is a product finding. Do not change the task to make
it pass.

The design is in
[docs/plans/2026-09-11-evals-framework.md](../../docs/plans/2026-09-11-evals-framework.md).
The move onto pi.Agent is in
[docs/plans/2026-09-11-evals-on-pi-agent.md](../../docs/plans/2026-09-11-evals-on-pi-agent.md).

## Run

```
make evals                                        # every task
make evals TASK=household-ledger                  # one task
make evals MODEL=anthropic/claude-haiku-4.5       # a cheaper model
make evals RUBRIC=1                               # also score against the rubric
```

The target builds `bin/xdb` and then runs the harness. Set
`OPENROUTER_API_KEY` first. Each run costs money.

To compare runs, run the command more than once with a different results
directory, and read the result files:

```
for i in 1 2 3; do make evals RESULTS=/tmp/eval-$i; done
```

Results go to `internal/evals/results/<timestamp>/`. Git ignores this
directory.

```
summary.txt                  # the table
summary.json
<task>/
  trace.01-<phase>.jsonl     # one agent event per line, per phase
  result.json                # checks, metrics, disclosure block
```

## Models

`agent.go` holds the model table. Each row is an OpenRouter slug and its
price. To add a model, add a row. A model with no price reports a cost of
zero, and the cost budget then never trips.

## Write a task

Each task has one directory under `tasks/`:

```
tasks/<name>/
  task.yaml
  fixtures/     # the harness copies these files into the work directory of the agent
```

Rules for `task.yaml`:

- The `name` must match the directory name.
- The prompt names the namespace and the domain nouns. It does not name
  fields, types, flags, or commands. The agent designs the shape.
- Write `$NS` in prompts and check commands. The harness replaces it with
  the task namespace.
- Prefer `answer` checks. They grade what the agent can read back out of
  its own store, and they do not depend on the shape that it chose. Use
  `run` checks for facts that an answer cannot carry, such as record counts
  and schema existence.
- A `run` check is a shell command. Exit code 0 passes. Count with `wc -l`
  and compare with `test`. Read a field with `jq`.
- Later phases add curveballs: a wrong amount, a duplicate, a schema
  change, a stale version.
- Put the expected numbers in a comment at the top of the file. Then the
  next person can compute them again when a fixture changes.

See `tasks/household-ledger/task.yaml` for an example.

## Subject inputs

The system prompt holds the sandbox facts and the output of `xdb context`
from the sandbox binary. The task prompt holds only the task. The agent has
one tool: a shell that runs in the sandbox, with a private `HOME` and the
sandbox `bin` first on `PATH`. The repo path does not appear in its
environment.

## Read the results

The table has one row per task:

| Column   | Meaning                                                  |
| -------- | -------------------------------------------------------- |
| RESULT   | PASS or FAIL                                             |
| CHECKS   | Checks passed over checks run                            |
| TURNS    | Agent turns, summed over phases                          |
| XDB      | Shell commands that ran xdb                              |
| FAILED   | xdb calls that exited non-zero                           |
| BLIND    | Failed calls on an action that the agent did not look up |
| DEEPEST  | Deepest disclosure layer reached (L1 help to L4 skills)   |
| RECOVERY | Share of failed calls with a later success on the action |
| COST     | USD                                                      |

A high `blind_failures` count means that the agent guesses instead of
reading, or that the errors do not point at the docs. A deepest layer of 4
means that the lower layers lack something that the skill has. A
`hinted_failures` count with a low `hint_followed` means that the error
hints do not say a thing the agent can act on.

The event log holds every command the agent ran, in order. Read it when a
number needs a reason.
