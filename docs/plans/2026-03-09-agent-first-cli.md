# Agent-First CLI Plan

**Date:** 2026-03-09
**Status:** Draft
**Reference:** [Rewrite Your CLI for AI Agents](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)

## Current State

- Only `core/` package exists (URI, Tuple, Record, Value, Type, Builder)
- No CLI, no store backends, no API layer
- README describes the target CLI surface (S3-like: `ls`, `get`, `put`, `rm`, `make-schema`, `daemon`)
- Module: `github.com/xdb-dev/xdb`, Go 1.26

## Design Principles

Apply all 7 principles from the blog post to XDB's CLI from inception.

---

## 1. Output: JSON-First, Table for Humans

### Auto-detection (already in README spec)

- **TTY** → table format (human-readable)
- **Pipe/redirect** → JSON (machine-readable, compact by default)
- `--output json|table|yaml` to override

### Agent additions

- `--output ndjson` — one JSON object per line for streaming list results. Enables agents to process records incrementally without buffering entire result sets.
- Compact JSON by default when piped; `--output json --pretty` for indented.
- All JSON output uses a consistent envelope:

```json
{
  "kind": "Record",
  "uri": "xdb://com.example/posts/post-123",
  "data": { ... }
}
```

Lists use:

```json
{
  "kind": "RecordList",
  "uri": "xdb://com.example/posts",
  "items": [ ... ],
  "next_offset": 10,
  "total": 42
}
```

This lets agents reliably parse without guessing structure.

---

## 2. Raw JSON Payloads via `--json`

The `put` command already accepts JSON files/stdin. Extend this pattern:

```bash
# Current (file/stdin) — keep this
xdb put xdb://com.example/posts/post-123 --file post.json

# New: inline JSON payload
xdb put xdb://com.example/posts/post-123 --json '{"title":"Hello","content":"World"}'

# Schema creation with inline JSON
xdb make-schema xdb://com.example/posts --json '{"attrs":[{"name":"title","type":"string"}]}'
```

The `--json` flag accepts the full record/schema payload directly. No flag-per-field mapping needed. Agents construct JSON naturally; this avoids translation loss.

For `ls` filters, also support a JSON filter syntax alongside the human-friendly `--filter`:

```bash
# Human-friendly (keep)
xdb ls xdb://com.example/posts --filter "age>30"

# Agent-friendly: structured filter
xdb ls xdb://com.example/posts --query '{"filters":[{"attr":"age","op":">","value":30}],"limit":10}'
```

---

## 3. Schema Introspection at Runtime

Agents should never need external docs to understand XDB's data model. The CLI itself is the documentation.

### `xdb describe` command

```bash
# Describe a schema — returns field names, types, constraints
xdb describe xdb://com.example/posts
```

Output:

```json
{
  "kind": "SchemaDescription",
  "uri": "xdb://com.example/posts",
  "attrs": [
    {"name": "title", "type": "string", "required": true},
    {"name": "content", "type": "string", "required": false},
    {"name": "author.id", "type": "string", "required": true},
    {"name": "created_at", "type": "timestamp", "required": true}
  ],
  "strict": true
}
```

```bash
# Describe a command — returns parameters, flags, examples
xdb describe --command put
```

Output:

```json
{
  "kind": "CommandDescription",
  "name": "put",
  "summary": "Create or update a record",
  "args": [
    {"name": "uri", "type": "string", "required": true, "description": "Record URI (xdb://NS/SCHEMA/ID)"}
  ],
  "flags": [
    {"name": "file", "short": "f", "type": "string", "description": "Path to input file"},
    {"name": "json", "type": "string", "description": "Inline JSON payload"},
    {"name": "format", "type": "string", "default": "json", "enum": ["json", "yaml"]},
    {"name": "dry-run", "type": "bool", "description": "Validate without writing"}
  ],
  "examples": [
    "xdb put xdb://com.example/posts/123 --json '{\"title\":\"Hello\"}'"
  ],
  "mutating": true
}
```

```bash
# Describe supported types
xdb describe --types
```

```bash
# Describe URI format
xdb describe --uri
```

This replaces the need for agents to read docs. The CLI is the canonical reference.

---

## 4. Field Masks for Context Window Discipline

API responses (especially records with many attributes) can be large. Agents shouldn't pay tokens for fields they don't need.

```bash
# Return only specific attributes
xdb get xdb://com.example/posts/post-123 --fields title,author.id,created_at

# List with field mask — each record only contains selected attrs
xdb ls xdb://com.example/posts --fields id,title --limit 20

# Combine with NDJSON for streaming filtered lists
xdb ls xdb://com.example/posts --fields id,title --output ndjson --page-all
```

The `--fields` flag accepts a comma-separated list of attribute names. The response only includes those attributes, reducing token consumption.

The `--page-all` flag with `--output ndjson` streams all pages as individual JSON objects, enabling incremental processing:

```
{"id":"post-1","title":"First Post"}
{"id":"post-2","title":"Second Post"}
...
```

---

## 5. Input Hardening Against Hallucinations

The agent is not a trusted operator. XDB URIs are the primary input surface and the main attack vector.

### URI Validation (defense-in-depth)

| Check | Rule | Example rejection |
|-------|------|-------------------|
| Control chars | Reject ASCII < 0x20 | `xdb://com.example/posts\x00` |
| Query params | Reject `?` in URI path | `xdb://com.example/posts?limit=10` |
| Fragment abuse | Only allow `#` for attribute position | `xdb://com.example/posts#../../etc` |
| Path traversal | Reject `..` in any component | `xdb://../../.ssh/keys` |
| Double encoding | Reject `%` in NS/Schema/ID | `xdb://com.example/%2e%2e/posts` |
| Whitespace | Reject spaces and tabs | `xdb://com.example/posts /123` |

### File Path Validation

- `--file` paths: canonicalize with `filepath.Abs` + `filepath.Clean`
- Reject paths outside CWD (sandbox)
- Reject symlinks that escape sandbox
- Reject `/dev/`, `/proc/`, `/etc/` prefixes

### Value Validation

- JSON payloads: validate against schema (if strict) before write
- Reject payloads > configurable max size (default 1MB)
- Reject deeply nested JSON (default max depth: 20)

### Error Messages

Validation errors must be specific and machine-parseable:

```json
{
  "error": {
    "code": "INVALID_URI",
    "message": "URI path contains path traversal sequence '..'",
    "field": "uri",
    "value": "xdb://../../.ssh/keys"
  }
}
```

---

## 6. `--dry-run` for Mutation Safety

All mutating commands (`put`, `rm`, `make-schema`) support `--dry-run`:

```bash
# Validate the put without writing
xdb put xdb://com.example/posts/post-123 --json '{"title":"Hello"}' --dry-run
```

Output:

```json
{
  "dry_run": true,
  "valid": true,
  "operation": "put",
  "uri": "xdb://com.example/posts/post-123",
  "changes": {
    "attrs_set": ["title"],
    "attrs_removed": [],
    "record_exists": false
  }
}
```

On validation failure:

```json
{
  "dry_run": true,
  "valid": false,
  "operation": "put",
  "uri": "xdb://com.example/posts/post-123",
  "errors": [
    {"attr": "title", "error": "expected string, got integer"}
  ]
}
```

`--dry-run` performs:
1. URI validation
2. Input parsing and type checking
3. Schema validation (if strict schema)
4. Conflict detection (record exists / doesn't exist)
5. Returns what *would* happen without executing

---

## 7. Agent Context & Skills

Two commands: `xdb context` for full agent context injection, `xdb skills` for skill discovery and fetching.

### `xdb context`

Returns a complete, self-contained context document that can be piped directly into an agent's system prompt or CLAUDE.md. This is the **primary integration point** — any agent framework calls this once and gets everything it needs.

```bash
# Get full agent context (markdown to stdout)
xdb context

# Pipe into a file for CLAUDE.md inclusion
xdb context >> CLAUDE.md

# Includes live data: installed schemas, available backends
xdb context
```

Output is markdown — designed for LLM consumption, not machine parsing:

```markdown
# XDB CLI — Agent Context

## Quick Reference
- All data is addressed by URI: xdb://NS/SCHEMA/ID#ATTR
- Use `xdb describe --command <cmd>` for command details
- Use `xdb describe <uri>` for schema details

## Rules
- ALWAYS use --dry-run before mutating operations (put, rm, make-schema)
- ALWAYS confirm with user before rm commands
- ALWAYS use --fields on list operations to limit response size
- ALWAYS use --output json when parsing output programmatically
- Use --query for structured filters instead of --filter string parsing
- URI components must not contain: ?, #, %, .., control characters

## Commands
[auto-generated from cobra command tree — name, summary, flags, examples, mutating?]

## Installed Schemas
[live query — lists namespaces and schemas currently in the store]

## Common Patterns

### Create and verify a record
xdb put xdb://NS/SCHEMA/ID --json '{...}' --dry-run
xdb put xdb://NS/SCHEMA/ID --json '{...}'
xdb get xdb://NS/SCHEMA/ID --fields <relevant-fields>

### Explore data
xdb describe xdb://NS/SCHEMA         # understand schema
xdb ls xdb://NS/SCHEMA --fields id --limit 5  # sample IDs
xdb get xdb://NS/SCHEMA/ID           # fetch one record
```

Key design: the output is **dynamic**. It includes live data from the running store (installed schemas, namespaces) so the agent knows what data exists without having to discover it through trial and error. The command sections are auto-generated from the cobra command tree, so they're always in sync with the binary.

### `xdb skills`

Skills are individual, focused documents about specific XDB capabilities. `xdb skills` provides discovery and fetching.

```bash
# List all available skills
xdb skills

# Output:
# NAME          DESCRIPTION
# crud          Create, read, update, delete records
# schema        Define and manage schemas
# filtering     Query and filter records
# bulk          Pagination, streaming, NDJSON patterns
# uri           URI format and addressing

# Fetch a specific skill (markdown to stdout)
xdb skills get crud

# JSON output for programmatic use
xdb skills --output json
xdb skills get crud --output json
```

Skills are **compiled into the binary** — no external files to install or symlink. Each skill is a focused markdown document covering one capability surface:

| Skill | Covers |
|-------|--------|
| `crud` | put, get, ls, rm — usage, flags, examples, gotchas |
| `schema` | make-schema, describe — schema definition, introspection |
| `filtering` | --filter, --query — operators, structured filters, combining |
| `bulk` | --page-all, --output ndjson, --fields — streaming, field masks |
| `uri` | URI format, components, valid/invalid examples, addressing |

#### Why compiled-in, not files on disk

- **Always in sync** with the binary version — no stale skill files
- **Zero setup** — no `npx skills add`, no symlinks, no `~/.openclaw/skills/`
- **Discoverable** — `xdb skills` lists what's available; no filesystem scanning
- **Portable** — works on any machine with the binary, no extra install step

#### How agents use skills

An agent framework can:
1. Run `xdb context` at conversation start for the full picture
2. Run `xdb skills get <name>` on-demand when it needs deeper guidance on a specific topic (e.g., before constructing a complex filter)

This is pull-based discovery — the agent asks the CLI what it can do, rather than requiring pre-installed context files.

---

## 8. Configuration: Config File + Flags

Two layers only. No env var config surface.

### Config file (`~/.xdb/config.json`)

Persistent settings. Created automatically on first run.

```json
{
  "dir": "~/.xdb",
  "daemon": { "addr": "localhost:8147", "socket": "xdb.sock" },
  "log_level": "info",
  "store": { "backend": "sqlite", "sqlite": { "name": "xdb.db" } }
}
```

### Flags (per-invocation overrides)

- `--config`, `-c`: Path to config file (default `~/.xdb/config.json`)
- `--output`, `-o`: Output format (json, table, yaml, ndjson)
- `--verbose`, `-v`: Enable verbose logging
- `--debug`: Enable debug logging

### One env var: `XDB_CONFIG`

Points at a non-default config file location. Same convention as `KUBECONFIG`. Avoids passing `--config` on every invocation in CI/containers.

### Precedence

```
flags > config file > defaults
XDB_CONFIG only changes which config file is loaded.
```

No parallel env var config surface. CI/containers generate or mount a config file.

---

## Package Structure

```
cmd/xdb/
  main.go              # Entry point

internal/cli/
  app.go               # Root command setup, global flags
  get.go               # get command
  put.go               # put command
  ls.go                # ls command
  rm.go                # rm command
  make_schema.go       # make-schema command
  describe.go          # describe command (introspection)
  context.go           # context command (full agent context)
  skills.go            # skills command (list + get skills)
  daemon.go            # daemon start/stop/status/restart

internal/cli/output/
  output.go            # Output interface + format detection
  json.go              # JSON/NDJSON formatter
  table.go             # Table formatter
  yaml.go              # YAML formatter

internal/cli/validate/
  validate.go          # URI + file path + payload validation
```

---

## Implementation Phases

### Phase 1: Foundation
1. `internal/cli/validate/` — URI hardening, file path sandboxing, payload validation
2. `internal/cli/output/` — JSON, NDJSON, table, YAML formatters with TTY detection
3. `internal/cli/app.go` — Root command with global flags + env var binding
4. `cmd/xdb/main.go` — Entry point

### Phase 2: Core Commands
5. `get` command with `--fields` support
6. `put` command with `--json`, `--file`, `--dry-run`
7. `ls` command with `--filter`, `--query`, `--fields`, `--page-all`, `--output ndjson`
8. `rm` command with `--force`, `--dry-run`, confirmation prompt
9. `make-schema` command with `--json`, `--dry-run`

### Phase 3: Introspection & Safety
10. `describe` command (schema, commands, types, URI format)
11. Structured error responses (JSON error envelope)
12. Input hardening test suite (fuzz with agent-typical hallucinations)

### Phase 4: Agent Surfaces
13. `xdb context` command (dynamic agent context generation)
14. `xdb skills` / `xdb skills get <name>` (compiled-in skill documents)
15. Daemon with Unix socket + HTTP

### Phase 5: Store Backends
16. Store interface in `store/` package
17. Memory backend
18. SQLite backend
19. Redis backend
20. Filesystem backend

---

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI framework | `cobra` | Standard, supports subcommands, completion, good help generation |
| Config binding | `viper` or manual | Flags > config file > defaults; `XDB_CONFIG` for file location only |
| JSON output | `encoding/json` | Stdlib, no external dep needed |
| Table output | `tablewriter` or custom | Minimal dep |
| Validation | Custom in `internal/cli/validate/` | XDB-specific rules, no generic framework needed |
| `--dry-run` scope | All mutating commands | Defense-in-depth for agents |
| NDJSON | Custom writer | One `json.Marshal` + newline per record |

---

## What XDB Does NOT Need (from the blog post)

- **Response sanitization (`--sanitize`)** — XDB stores user-defined data, not third-party content with prompt injection risk. If needed later, add at the application layer, not the CLI.
- **OAuth / browser auth** — XDB is local-first (daemon on localhost). Config file is sufficient.
- **Skill files on disk / OpenClaw registry** — Skills are compiled into the binary and served via `xdb skills`. No external files to install or manage.
- **Discovery Document** — XDB schemas *are* the discovery mechanism. `xdb describe` serves this purpose.
- **MCP surface** — Out of scope for now. Can be added later as a thin layer over the CLI commands.
