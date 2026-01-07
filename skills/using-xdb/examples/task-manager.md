# Multi-User Task Manager Example

Each user gets their own namespace for isolated task management. Run `xdb --help` for command syntax.

## Setup

```bash
# Use current user for namespace
USER_NS="user.$(whoami)"

# Create tasks schema for this user
xdb make-schema "xdb://$USER_NS/tasks"
```

## Nanoid Helper

```bash
nanoid() { openssl rand -base64 12 | tr -dc 'a-zA-Z0-9' | head -c 21; }
```

## Add Task

```bash
xdb put "xdb://$USER_NS/tasks/$(nanoid)" <<EOF
{
  "title": "Review PR #123",
  "due": "2024-01-20",
  "done": false,
  "tags": ["code-review", "urgent"]
}
EOF
```

## List Tasks

```bash
# List all tasks for current user
xdb ls "xdb://$USER_NS/tasks"
```

## Complete Task

```bash
# Mark task as done (use ID from list output)
xdb get "xdb://$USER_NS/tasks/V1StGXR8Z5jdHi9" -o json | \
  jq '.done = true' | \
  xdb put "xdb://$USER_NS/tasks/V1StGXR8Z5jdHi9"
```

## Delete Task

```bash
xdb rm "xdb://$USER_NS/tasks/V1StGXR8Z5jdHi9" --force
```

## With Strict Schema

```bash
cat > /tmp/task-schema.json <<EOF
{
  "name": "Task",
  "mode": "strict",
  "fields": [
    {"name": "title", "type": "STRING"},
    {"name": "description", "type": "STRING"},
    {"name": "due", "type": "STRING"},
    {"name": "done", "type": "BOOLEAN"},
    {"name": "priority", "type": "INTEGER"},
    {"name": "tags", "type": "ARRAY", "array_of": "STRING"},
    {"name": "created_at", "type": "TIME"}
  ]
}
EOF

xdb make-schema "xdb://$USER_NS/tasks" --schema /tmp/task-schema.json
```

## Team Shared Tasks

```bash
# Use team namespace for shared tasks
TEAM_NS="team.engineering"

xdb make-schema "xdb://$TEAM_NS/tasks"

xdb put "xdb://$TEAM_NS/tasks/sprint-task-1" <<EOF
{
  "title": "Deploy v2.0",
  "assignee": "$(whoami)",
  "due": "2024-01-25",
  "done": false
}
EOF
```
