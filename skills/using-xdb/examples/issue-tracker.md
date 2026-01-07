# Issue Tracker Agent Example

Track issues with a strict schema for consistent data. Run `xdb --help` for command syntax.

## Setup Schema

```bash
cat > /tmp/issue-schema.json <<EOF
{
  "name": "Issue",
  "mode": "strict",
  "fields": [
    {"name": "title", "type": "STRING"},
    {"name": "description", "type": "STRING"},
    {"name": "status", "type": "STRING"},
    {"name": "priority", "type": "INTEGER"},
    {"name": "assignee", "type": "STRING"},
    {"name": "labels", "type": "ARRAY", "array_of": "STRING"},
    {"name": "created_at", "type": "TIME"},
    {"name": "updated_at", "type": "TIME"}
  ]
}
EOF

xdb make-schema xdb://agent.tracker/issues --schema /tmp/issue-schema.json
```

## Create Issue

```bash
xdb put xdb://agent.tracker/issues/issue-1 <<EOF
{
  "title": "Fix authentication bug",
  "description": "Users cannot log in with SSO",
  "status": "open",
  "priority": 1,
  "assignee": "john",
  "labels": ["bug", "auth", "critical"],
  "created_at": "$(date -Iseconds)"
}
EOF
```

## Update Issue Status

```bash
# Get current issue, update status, save back
xdb get xdb://agent.tracker/issues/issue-1 -o json | \
  jq '.status = "in_progress" | .updated_at = now | todate' | \
  xdb put xdb://agent.tracker/issues/issue-1
```

## Query Issues

```bash
# List all issues
xdb ls xdb://agent.tracker/issues

# Get issue title
xdb get xdb://agent.tracker/issues/issue-1#title

# Get assignee
xdb get xdb://agent.tracker/issues/issue-1#assignee
```

## Close Issue

```bash
xdb get xdb://agent.tracker/issues/issue-1 -o json | \
  jq '.status = "closed" | .updated_at = now | todate' | \
  xdb put xdb://agent.tracker/issues/issue-1
```
