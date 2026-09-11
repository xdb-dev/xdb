---
name: query-and-filter
description: "Filter, project, and paginate record listings"
category: recipe
---

# Query and Filter

`records list` is the main read path. It supports CEL filters, field
projections, and pagination.

## Filters

Filters are CEL expressions. `xdb describe --filter` shows the full
grammar. String functions are case-sensitive.

```bash
xdb records list --uri xdb://myapp/todos --filter 'done == false'
xdb records list --uri xdb://myapp/todos --filter 'priority >= 2 && done == false'
xdb records list --uri xdb://myapp/todos --filter 'title.contains("urgent")'
xdb records list --uri xdb://myapp/todos --filter 'title in ["Try XDB", "Write docs"]'
```

Under a `strict` schema, a filter on an undeclared field fails with
INVALID_ARGUMENT. The error names the field and lists the declared
fields.

## Time ranges

Compare a `time` field against `timestamp("...")`. The argument is an
RFC 3339 time. A comparison against a plain string fails, because a
time and a string are different types.

```bash
xdb records list --uri xdb://myapp/txns \
  --filter 'date >= timestamp("2026-08-01T00:00:00Z") && date < timestamp("2026-09-01T00:00:00Z")'
```

`_updated` is a time field on every record, so the same form selects
the records written in a period.

## Absent fields

`has(attr)` asks whether the record holds the attribute. Negate it to
find the records that do not, such as the unassigned issues.

```bash
xdb records list --uri xdb://myapp/issues --filter '!has(assignee)'
xdb records list --uri xdb://myapp/issues --filter 'has(assignee) && done == false'
```

`_attrs` holds the attribute names of the record, so `"assignee" in
_attrs` is the same test.

## Projections

`--fields` limits the returned attributes. `_id` is always included.
System attributes start with an underscore: `_id`, `_version`,
`_updated`. Request `--fields _id,title`, not `id`.

```bash
xdb records list --uri xdb://myapp/todos --fields _id,title -o ndjson
```

## Pagination

`-o json` returns `{"items": [...], "total": N, "next_offset": M}`. To
continue, pass the returned `next_offset` as `--offset`. To fetch
everything, pass `--page-all`. `-o ndjson` streams bare items.

```bash
xdb records list --uri xdb://myapp/todos --limit 20 -o json
xdb records list --uri xdb://myapp/todos --page-all -o ndjson
```

## Scripting

Exit codes are stable: 0 success, 1 app error (NOT_FOUND,
ALREADY_EXISTS, CONFLICT, SCHEMA_VIOLATION, NOT_IMPLEMENTED), 2 daemon
unreachable, 3 invalid input, 4 internal.

```bash
xdb records get --uri xdb://myapp/todos/todo-1 --quiet
```

Exit 0 means that the record exists. Exit 1 means that it does not.
