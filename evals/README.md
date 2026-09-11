# Evals

Agent task evaluations for the xdb CLI. An eval gives a headless agent a
real-world task in a sandbox, in phases. The harness grades the result and
measures friction. A failed eval is a product finding. Do not change the
task to make it pass.

The design is in [docs/plans/2026-09-11-evals-framework.md](../docs/plans/2026-09-11-evals-framework.md).

## Run

```
make evals                         # every task, one run each
make evals TASK=household-ledger   # one task
make evals REPEAT=3                # pass rate and medians
make evals MODEL=haiku             # cheap smoke run
make evals RUBRIC=1                # also score each run against its rubric
```

The target builds `bin/xdb` and then runs `evals/cmd/xdb-eval`. It needs
the `claude` CLI on PATH and an account that is logged in. Each run costs
money.

Results go to `evals/results/<timestamp>/`. Git ignores this directory.

```
summary.md                       # the table
summary.json
<task>/
  trajectory.01-<phase>.jsonl    # one stream-json file per phase
  result.json                    # checks, metrics, disclosure block
```

## Write a task

Each task has one directory under `evals/tasks/`:

```
evals/tasks/<name>/
  task.yaml
  fixtures/        # the harness copies these files into the work directory of the agent
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
- A `run` check takes these assertions: `exit`, `exit_nonzero`,
  `stdout_contains`, `stderr_contains`, `stdout_empty`, `json`,
  `ndjson_count`, `ndjson_ids`, and `error`.
- Later phases add curveballs: a wrong amount, a duplicate, a schema
  change, a stale version.
- Put the expected numbers in a comment at the top of the file. Then the
  next person can compute them again when a fixture changes.

See `evals/tasks/household-ledger/task.yaml` for an example.

## Subject inputs

The system prompt holds the sandbox facts and the output of `xdb context`
from the sandbox binary. The task prompt holds only the task. The harness
disables the settings, hooks, and skills of the subject. The `xdb` on its
PATH is a shim that points `HOME` at the sandbox. The repo path does not
appear in its environment.

## Read the results

The summary table has one row per task:

| Column    | Meaning                                                    |
| --------- | ---------------------------------------------------------- |
| Result    | PASS or FAIL, or `k/n PASS` with `REPEAT`                   |
| Checks    | Checks passed over checks run                              |
| Turns     | Agent turns, summed over phases                            |
| xdb calls | Shell commands that ran xdb                                |
| Failed    | xdb calls that exited non-zero                             |
| Blind     | Failed calls on an action that the agent did not look up   |
| Deepest   | Deepest disclosure layer visited (L1 help to L4 skills)    |
| Recovery  | Share of failed calls with a later success on the action   |
| Cost      | USD                                                        |
| Rubric    | Average judge score out of 5, with `RUBRIC=1`              |

The `disclosure` block in `result.json` has the full path and the counters
that tell the layers apart. A high `blind_failures` count with few
`discovery_recoveries` means that the agent guesses instead of reading, or
that the errors do not point at the docs. Recoveries at layer 4 mean that
the lower layers lack something that the skill has. A `redundant_discovery`
count means that the agent did not trust the context guide on an action it
already shows.
