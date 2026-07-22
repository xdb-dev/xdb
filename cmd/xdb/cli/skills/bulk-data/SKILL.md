---
name: bulk-data
description: "Move data at scale with export, import, and atomic batches"
category: recipe
---

# Bulk Data

NDJSON round-trips and atomic multi-operation batches.

## Export

```bash
xdb export --uri xdb://myapp/todos > todos.ndjson
xdb export --uri xdb://myapp/todos -o json          # single JSON array
xdb export --uri xdb://myapp/todos/todo-1           # one record
xdb export --uri xdb://myapp/todos --fields _id,title
```

## Import

Import reads NDJSON from stdin or `--file` (the file must live under
the working directory). Each line needs an `_id` (or `id`) field; the
default mode is upsert.

```bash
xdb import --uri xdb://myapp/todos < todos.ndjson
xdb import --uri xdb://myapp/todos --create-only < todos.ndjson
```

The summary on stdout is machine-readable:
`{"imported": N, "skipped": N, "failed": N, "first_error_line": L}`.
Under `--create-only`, an existing record with different data is
counted as skipped and the local data is kept. Import is fail-fast and
not transactional: on error, lines before `first_error_line` are
already committed — fix the input and re-run (upsert mode makes the
retry idempotent).

## Atomic batches

`batch` runs `{op, uri, data}` operations in one transaction — all or
nothing on transactional backends (sqlite, memory):

```bash
xdb batch --json '[
  {"op": "records.create", "uri": "xdb://myapp/todos/t1", "data": {"title": "A"}},
  {"op": "records.update", "uri": "xdb://myapp/todos/t1", "data": {"done": true}}
]'
```

NDJSON works too (one op per line, pipe with `xdb batch -`). Allowed
ops: records.create/update/upsert/delete, schemas.create/update/delete.
Per-op results carry `{index, uri, status, error?}`; a failure rolls
back the whole batch (`"rolled_back": true`). Validate first with
`--dry-run`; on non-transactional backends pass `--non-atomic` for
sequential best-effort execution.
