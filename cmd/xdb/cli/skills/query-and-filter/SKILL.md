---
name: query-and-filter
description: "Filter, project, and paginate record listings"
category: recipe
---

# Query and Filter

`records list` is the core read path: CEL filters, field projections,
and pagination.

## Filters

Filters use CEL expressions (`xdb describe --filter` shows the full
grammar). String functions are case-sensitive.

```bash
xdb records list --uri xdb://myapp/todos --filter 'done == false'
xdb records list --uri xdb://myapp/todos --filter 'priority >= 2 && done == false'
xdb records list --uri xdb://myapp/todos --filter 'title.contains("urgent")'
xdb records list --uri xdb://myapp/todos --filter 'title in ["Try XDB", "Write docs"]'
```

Under a strict-mode schema, filtering on an undeclared field fails with
INVALID_ARGUMENT naming the field and listing the available ones.

## Projections

`--fields` limits the returned attributes; `_id` is always included.
System fields use a leading underscore (`_id`, `_ns`, `_schema`) —
request `--fields _id,title`, not `id`.

```bash
xdb records list --uri xdb://myapp/todos --fields _id,title -o ndjson
```

## Pagination

`-o json` returns `{"items": [...], "total": N, "next_offset": M}`;
pass `--offset` with the returned `next_offset` to continue, or
`--page-all` to fetch everything. `-o ndjson` streams bare items.

```bash
xdb records list --uri xdb://myapp/todos --limit 20 -o json
xdb records list --uri xdb://myapp/todos --page-all -o ndjson
```

## Scripting

Exit codes are stable: 0 success, 1 domain error (NOT_FOUND, CONFLICT,
SCHEMA_VIOLATION), 2 daemon unreachable, 3 invalid input, 4 internal.

```bash
xdb records get --uri xdb://myapp/todos/todo-1 --quiet
```

Exit 0 means the record exists; 1 means it does not.
