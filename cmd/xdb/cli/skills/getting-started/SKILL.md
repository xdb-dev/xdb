---
name: getting-started
description: "Create your first schema and records"
category: recipe
---

# Getting Started

Create a schema, add records, and query them.

## Prerequisite

Initialize XDB and start the daemon (safe to re-run):

```bash
xdb init
```

## Steps

1. Create a schema:

```bash
xdb schemas create --uri xdb://myapp/todos --json '{
  "fields": {
    "title": {"type": "string"},
    "done":  {"type": "boolean"}
  }
}'
```

2. Add a record:

```bash
xdb records create --uri xdb://myapp/todos/todo-1 --json '{
  "title": "Try XDB",
  "done": false
}'
```

3. Read it back:

```bash
xdb records get --uri xdb://myapp/todos/todo-1
```

4. List records:

```bash
xdb records list --uri xdb://myapp/todos --fields title,done
```

5. Update a record (patch semantics — only supplied fields change):

```bash
xdb records update --uri xdb://myapp/todos/todo-1 --json '{"done": true}'
```

6. Delete a record:

```bash
xdb records delete --uri xdb://myapp/todos/todo-1 --force
```

## Next steps

- `xdb skills get query-and-filter` — filtering, projections, and pagination
- `xdb skills get schema-evolution` — evolving schemas safely
- `xdb skills get bulk-data` — import/export and atomic batches
- `xdb describe --actions` — every available operation
- `xdb describe --schema-format` — the full schema-definition format
