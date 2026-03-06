---
name: using-xdb
description: Use XDB CLI to store and manage structured data. Use when you need to persist data, create schemas for tracking information (issues, scraped data, metrics, logs), or organize data by namespace for different projects/users/contexts.
allowed-tools: Bash(xdb:*), Write
---

# Using XDB CLI for Data Management

## Install

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
```

## Configuration

XDB auto-creates `~/.xdb/config.json` on first run. Default backend is SQLite (`~/.xdb/data/xdb.db`). Available backends: `sqlite` (default), `memory`, `redis`, `fs`.

## URI Format

```
xdb://NAMESPACE/SCHEMA/ID#ATTRIBUTE
```

| Component | Required     | Example                                       |
| --------- | ------------ | --------------------------------------------- |
| Namespace | Yes          | `agent.scraper`, `user.john`, `project.myapp` |
| Schema    | For data ops | `articles`, `issues`, `tasks`                 |
| ID        | For records  | `article-123`, `issue-1`                      |
| Attribute | Optional     | `#title`, `#status`, `#author.name`           |

## Namespace Strategy

| Pattern          | Use Case            | Example                             |
| ---------------- | ------------------- | ----------------------------------- |
| `agent.<name>`   | Agent-specific data | `agent.scraper`, `agent.researcher` |
| `user.<id>`      | Per-user data       | `user.john`, `user.admin`           |
| `project.<name>` | Project data        | `project.webapp`, `project.api`     |
| `team.<name>`    | Team shared data    | `team.engineering`, `team.support`  |
| `org.<name>`     | Organization data   | `org.acme`, `org.internal`          |

## Schema Definition Format

**Modes:**

| Mode       | Behavior                                                |
| ---------- | ------------------------------------------------------- |
| `flexible` | Accepts any attributes (default, no schema file needed) |
| `strict`   | Only allows attributes defined in schema                |
| `dynamic`  | Auto-infers and adds new fields from data               |

Schema JSON definition (for `strict` or `dynamic` modes):

```json
{
  "name": "Bug",
  "mode": "strict",
  "fields": [
    { "name": "title", "type": "STRING" },
    { "name": "severity", "type": "STRING" },
    { "name": "tags", "type": "ARRAY", "array_of": "STRING" }
  ]
}
```

**Types:** `STRING`, `INTEGER`, `UNSIGNED`, `FLOAT`, `BOOLEAN`, `TIME`, `BYTES`, `ARRAY`, `MAP`
**Nested fields:** Use dot notation (`author.name`, `stats.views`)

## Nanoid Helper

Generate unique IDs:

```bash
nanoid() { openssl rand -base64 12 | tr -dc 'a-zA-Z0-9' | head -c 21; }
```

## Command Reference

### make-schema (alias: ms)

```bash
# Create flexible schema (no definition needed)
xdb make-schema xdb://NS/SCHEMA

# Create schema from definition file
xdb make-schema xdb://NS/SCHEMA -s <file>
```

### get

```bash
# Get namespace, schema, record, or attribute
xdb get xdb://NS
xdb get xdb://NS/SCHEMA
xdb get xdb://NS/SCHEMA/ID
xdb get xdb://NS/SCHEMA/ID#ATTRIBUTE
```

### put

```bash
# Put record from file
xdb put xdb://NS/SCHEMA/ID -f <path>

# Put record from stdin
echo '{"key":"value"}' | xdb put xdb://NS/SCHEMA/ID

# Put from YAML
xdb put xdb://NS/SCHEMA/ID -f data.yaml --format yaml
```

### list (alias: ls)

```bash
# List schemas in namespace
xdb ls xdb://NS

# List records in schema
xdb ls xdb://NS/SCHEMA

# Paginate results
xdb ls xdb://NS/SCHEMA --limit 10 --offset 20
```

### remove (aliases: rm, delete)

```bash
# Remove with confirmation prompt
xdb rm xdb://NS/SCHEMA/ID

# Remove without confirmation
xdb rm xdb://NS/SCHEMA/ID -f

# Remove schema
xdb rm xdb://NS/SCHEMA --force
```

### daemon

```bash
xdb daemon start              # Start in background
xdb daemon stop [--force]     # Stop (force kill if needed)
xdb daemon status [--json]    # Show status
xdb daemon restart [--force]  # Restart
xdb daemon logs [-f] [-n 50]  # View/follow logs
```

### Global Flags

| Flag        | Alias | Description                                            |
| ----------- | ----- | ------------------------------------------------------ |
| `--output`  | `-o`  | Output format: `json`, `table`, `yaml` (auto-detected) |
| `--config`  | `-c`  | Path to config file                                    |
| `--verbose` | `-v`  | Enable verbose logging                                 |
| `--debug`   |       | Enable debug logging with source locations             |

## Example Use Cases

| Use Case       | Mode       | Example URI                          |
| -------------- | ---------- | ------------------------------------ |
| Web scraping   | `flexible` | `xdb://agent.scraper/pages`          |
| Issue tracking | `strict`   | `xdb://project.webapp/bugs`          |
| Research notes | `dynamic`  | `xdb://agent.researcher/notes`       |
| Bookmarks      | `flexible` | `xdb://user.john/bookmarks`          |
| Feature flags  | `strict`   | `xdb://project.myapp/flags`          |
