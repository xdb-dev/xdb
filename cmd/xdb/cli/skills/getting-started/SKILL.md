---
name: getting-started
description: "Create your first schema and records"
category: recipe
---

# Getting Started

Create a schema, add records, and query them.

## Prerequisite

Initialize XDB and start the daemon. You can run this command again at any time:

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

3. Read the record:

```bash
xdb records get --uri xdb://myapp/todos/todo-1
```

4. List the records:

```bash
xdb records list --uri xdb://myapp/todos --fields title,done
```

5. Update a record. An update is a patch: only the fields in the payload change.

```bash
xdb records update --uri xdb://myapp/todos/todo-1 --json '{"done": true}'
```

6. Delete a record:

```bash
xdb records delete --uri xdb://myapp/todos/todo-1 --force
```

## Next steps

- `xdb skills get query-and-filter`: filters, projections, and pagination
- `xdb skills get schema-evolution`: safe schema changes
- `xdb skills get bulk-data`: import, export, and atomic batches
- `xdb describe --actions`: every available action
- `xdb describe --schema-format`: the full JSON format of a definition
