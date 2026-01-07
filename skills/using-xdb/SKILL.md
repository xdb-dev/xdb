---
name: using-xdb
description: Use XDB CLI to store and manage structured data. Use when you need to persist data, create schemas for tracking information (issues, scraped data, metrics, logs), or organize data by namespace for different projects/users/contexts.
allowed-tools: Bash(xdb:*), Write
---

# Using XDB CLI for Data Management

Run `xdb --help` for command syntax and flags.

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

Flexible schema:

```bash
xdb make-schema xdb://agent.scraper/articles
```

Strict or dynamic schema with JSON definition:

```json
{
  "name": "Article",
  "mode": "strict",
  "fields": [
    { "name": "title", "type": "STRING" },
    { "name": "tags", "type": "ARRAY", "array_of": "STRING" },
    {
      "name": "metadata",
      "type": "MAP",
      "map_key": "STRING",
      "map_value": "STRING"
    },
    { "name": "author.name", "type": "STRING" },
    { "name": "author.email", "type": "STRING" }
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

## Example Use Cases

| Use Case            | Mode       | Namespace          | Schema          |
| ------------------- | ---------- | ------------------ | --------------- |
| Web scraping        | `flexible` | `agent.scraper`    | `pages`         |
| Issue tracking      | `strict`   | `project.<name>`   | `issues`        |
| Research notes      | `dynamic`  | `agent.researcher` | `notes`         |
| API cache           | `flexible` | `agent.api`        | `cache`         |
| Conversation memory | `dynamic`  | `agent.memory`     | `conversations` |
| Bookmarks           | `flexible` | `user.<id>`        | `bookmarks`     |
| Code snippets       | `strict`   | `team.<name>`      | `snippets`      |
| Meeting notes       | `dynamic`  | `org.<name>`       | `meetings`      |
| Error logs          | `dynamic`  | `project.<name>`   | `errors`        |
| Feature flags       | `strict`   | `project.<name>`   | `flags`         |

## Workflow Examples

- [Web Scraper Agent](examples/web-scraper.md) - Store scraped pages
- [Issue Tracker Agent](examples/issue-tracker.md) - Track issues with strict schema
- [Metrics Tracking Agent](examples/metrics-tracker.md) - Time-series metrics
- [Multi-User Task Manager](examples/task-manager.md) - Per-user namespaces
