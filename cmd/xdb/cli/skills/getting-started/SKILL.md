---
name: getting-started
description: "Create your first schema and records"
category: recipe
---

# Getting Started

Create a schema, add records, and query them.

## Steps

1. Create a schema:
```bash
xdb schemas create --uri xdb://myapp/todos --json '{
  "Fields": {
    "title":  {"Type": "string"},
    "done":   {"Type": "bool"}
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

3. List records:
```bash
xdb records list --uri xdb://myapp/todos --fields title,done
```

4. Update a record:
```bash
xdb records update --uri xdb://myapp/todos/todo-1 --json '{"done": true}'
```

5. Delete a record:
```bash
xdb records delete --uri xdb://myapp/todos/todo-1 --force
```
