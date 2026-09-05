---
name: bulk-data
description: "Move data at scale with export, import, and atomic batches"
category: recipe
---

# Bulk Data

Export and import records as NDJSON. Run several operations as one atomic batch.

## Export

```bash
xdb export --uri xdb://myapp/todos > todos.ndjson
xdb export --uri xdb://myapp/todos -o json          # single JSON array
xdb export --uri xdb://myapp/todos/todo-1           # one record
xdb export --uri xdb://myapp/todos --fields _id,title
```

## Import

Import reads NDJSON from stdin or from `--file`. The file must be under
the working directory. Each line needs an `_id` field (`id` is also
accepted). The default mode is upsert.

```bash
xdb import --uri xdb://myapp/todos < todos.ndjson
xdb import --uri xdb://myapp/todos --create-only < todos.ndjson
```

The summary on stdout is machine-readable:
`{"imported": N, "skipped": N, "failed": N, "first_error_line": L}`.
With `--create-only`, an existing record with different data counts as
skipped, and the stored data is kept. Import stops at the first error
and is not transactional. After an error, the lines before
`first_error_line` are already committed. Fix the input and run the
import again. In upsert mode, the retry is idempotent.

## Atomic batches

`batch` runs a list of `{op, uri, data}` operations in one transaction.
On the transactional backends (`sqlite`, `memory`), all operations
succeed or none do:

```bash
xdb batch --json '[
  {"op": "records.create", "uri": "xdb://myapp/todos/t1", "data": {"title": "A"}},
  {"op": "records.update", "uri": "xdb://myapp/todos/t1", "data": {"done": true}}
]'
```

NDJSON also works: one op per line, piped with `xdb batch -`. Allowed
ops: `records.create`, `records.update`, `records.upsert`,
`records.delete`, `schemas.create`, `schemas.update`, `schemas.delete`.
Each per-op result carries `{index, uri, status, error?}`. If one op
fails, the whole batch rolls back and the response has
`"rolled_back": true`. To validate without writing, pass `--dry-run`.
On the non-transactional backends (`fs`, `redis`), pass `--non-atomic`
for sequential best-effort execution.
