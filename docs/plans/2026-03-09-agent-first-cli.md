# Agent-First CLI Plan

**Date:** 2026-03-09
**Status:** Draft
**Reference:** [Rewrite Your CLI for AI Agents](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)
**Inspiration:** [Google Workspace CLI](https://github.com/googleworkspace/cli) — RPC-method command structure

## Current State

- Only `core/` package exists (URI, Tuple, Record, Value, Type, Builder)
- No CLI, no store backends, no API layer
- README describes the target CLI surface (S3-like: `ls`, `get`, `put`, `rm`, `make-schema`, `daemon`)
- Module: `github.com/xdb-dev/xdb`, Go 1.26

## Design Principles

Apply all 7 principles from the blog post to XDB's CLI from inception.

### Hybrid RPC + Human-Friendly Design

The CLI has two layers:

1. **Canonical RPC commands** (`xdb <resource> <method>`) — predictable, explicit, agent-preferred
2. **Human aliases** (`get`, `put`, `ls`, `rm`) — thin wrappers for interactive use

Agents use the canonical form. Humans use whichever they prefer. Both resolve to the same code path.

---

## Command Surface

### Resources & Methods

XDB has three resources: **records**, **schemas**, **namespaces**. Each exposes CRUD-style RPC methods.

#### Records

```bash
xdb records create  --uri xdb://com.example/posts/post-123 --json '{...}'
xdb records get     --uri xdb://com.example/posts/post-123
xdb records list    --uri xdb://com.example/posts --filter "age>30"
xdb records update  --uri xdb://com.example/posts/post-123 --json '{...}'
xdb records upsert  --uri xdb://com.example/posts/post-123 --json '{...}'
xdb records delete  --uri xdb://com.example/posts/post-123
```

- `create` fails if record already exists
- `update` fails if record doesn't exist
- `upsert` creates or updates (equivalent to old `put`)
- `delete` prompts for confirmation unless `--force`

#### Schemas

```bash
xdb schemas create  --uri xdb://com.example/posts --json '{...}'
xdb schemas get     --uri xdb://com.example/posts
xdb schemas list    --uri xdb://com.example
xdb schemas update  --uri xdb://com.example/posts --json '{...}'
xdb schemas delete  --uri xdb://com.example/posts
```

#### Namespaces

```bash
xdb namespaces list
xdb namespaces get  --uri xdb://com.example
```

#### Batch

Execute multiple RPC methods in a single atomic transaction. Either all operations succeed or none do.

```bash
# Inline JSON array of operations
xdb batch --json '[
  {"method": "records.create", "uri": "xdb://com.example/posts/post-1", "body": {"title": "First"}},
  {"method": "records.create", "uri": "xdb://com.example/posts/post-2", "body": {"title": "Second"}},
  {"method": "records.get",    "uri": "xdb://com.example/posts/post-1"}
]'

# From file
xdb batch --file operations.json

# From stdin (useful for agent-generated pipelines)
cat operations.json | xdb batch

# Dry-run the entire batch
xdb batch --file operations.json --dry-run

# Stream results as NDJSON (one result per line)
xdb batch --file operations.json --output ndjson
```

Each operation in the array mirrors the canonical RPC method signature:

```json
{
  "method": "records.create",
  "uri": "xdb://com.example/posts/post-1",
  "body": { "title": "Hello", "content": "World" }
}
```

Supported methods in batch:

- **Mutations:** `records.create`, `records.update`, `records.upsert`, `records.delete`, `schemas.create`, `schemas.update`, `schemas.delete`
- **Reads:** `records.get`, `records.list`, `schemas.get`, `schemas.list`, `namespaces.get`, `namespaces.list`

Reads execute within the same transaction snapshot, so they see a consistent view of the data — including writes from earlier operations in the batch. This enables patterns like "create a record, then read it back to confirm" in a single atomic call.

**Response:**

```json
{
  "kind": "BatchResult",
  "total": 3,
  "succeeded": 3,
  "failed": 0,
  "results": [
    {
      "index": 0,
      "method": "records.create",
      "uri": "xdb://com.example/posts/post-1",
      "status": "ok"
    },
    {
      "index": 1,
      "method": "records.create",
      "uri": "xdb://com.example/posts/post-2",
      "status": "ok"
    },
    {
      "index": 2,
      "method": "records.get",
      "uri": "xdb://com.example/posts/post-1",
      "status": "ok",
      "data": { "title": "First" }
    }
  ]
}
```

**On failure (entire batch rolls back):**

```json
{
  "kind": "BatchResult",
  "total": 3,
  "succeeded": 0,
  "failed": 1,
  "rolled_back": true,
  "results": [
    {
      "index": 0,
      "method": "records.create",
      "uri": "xdb://com.example/posts/post-1",
      "status": "ok"
    },
    {
      "index": 1,
      "method": "records.create",
      "uri": "xdb://com.example/posts/post-2",
      "status": "ok"
    },
    {
      "index": 2,
      "method": "records.get",
      "uri": "xdb://com.example/posts/post-99",
      "status": "error",
      "error": { "code": "NOT_FOUND", "message": "Record does not exist" }
    }
  ]
}
```

**Dry-run** validates every operation in sequence (URI validation, schema checks, conflict detection) and reports what _would_ happen, without executing or acquiring a transaction.

**Limits:**

- Max 100 operations per batch (configurable in config file)
- Max 10MB total payload size
- Operations execute in array order within the transaction

### Human Aliases

Short-form commands that infer the resource from the URI depth:

```bash
# These pairs are equivalent:
xdb get xdb://com.example/posts/post-123       # alias → records get
xdb records get --uri xdb://com.example/posts/post-123

xdb get xdb://com.example/posts                # alias → schemas get
xdb schemas get --uri xdb://com.example/posts

xdb ls xdb://com.example                       # alias → schemas list
xdb schemas list --uri xdb://com.example

xdb ls xdb://com.example/posts                 # alias → records list
xdb records list --uri xdb://com.example/posts

xdb put xdb://com.example/posts/post-123 ...   # alias → records upsert
xdb records upsert --uri xdb://com.example/posts/post-123 ...

xdb rm xdb://com.example/posts/post-123        # alias → records delete
xdb records delete --uri xdb://com.example/posts/post-123
```

Aliases accept positional URIs (no `--uri` flag needed). The canonical commands require `--uri` for explicitness.

| Alias               | Resolves To         | URI Inference                                                         |
| ------------------- | ------------------- | --------------------------------------------------------------------- |
| `get <uri>`         | `{resource} get`    | 3 parts → records, 2 parts → schemas, 1 part → namespaces             |
| `ls <uri>`          | `{resource} list`   | 2 parts → records list, 1 part → schemas list, none → namespaces list |
| `put <uri>`         | `records upsert`    | Always records (3-part URI required)                                  |
| `rm <uri>`          | `{resource} delete` | 3 parts → records, 2 parts → schemas                                  |
| `make-schema <uri>` | `schemas create`    | Always schemas (2-part URI required)                                  |

---

## 1. Output: JSON-First, Table for Humans

### Auto-detection (already in README spec)

- **TTY** → table format (human-readable)
- **Pipe/redirect** → JSON (machine-readable, compact by default)
- `--output json|table|yaml` to override

### Agent additions

- `--output ndjson` — one JSON object per line for streaming. Available on list and batch operations.
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

### NDJSON Streaming

**Reference:** [gws-cli formatter.rs](https://github.com/googleworkspace/cli/blob/main/src/formatter.rs) — compact JSON per line for paginated responses, headers only on first page.

NDJSON (`--output ndjson`) unwraps envelopes and emits one JSON object per line. This is opt-in — JSON with envelope is the default for pipes.

#### List operations

```bash
xdb records list --uri xdb://com.example/posts --output ndjson
```

```
{"uri":"xdb://com.example/posts/post-1","data":{"title":"First Post"}}
{"uri":"xdb://com.example/posts/post-2","data":{"title":"Second Post"}}
{"uri":"xdb://com.example/posts/post-3","data":{"title":"Third Post"}}
```

No envelope, no `items` array — each record is a standalone JSON object on its own line. Consumers process line-by-line with zero buffering.

With `--page-all`, streams across all pages without waiting for the full result:

```bash
xdb records list --uri xdb://com.example/posts --output ndjson --page-all --fields id,title
```

```
{"uri":"xdb://com.example/posts/post-1","data":{"id":"post-1","title":"First"}}
{"uri":"xdb://com.example/posts/post-2","data":{"id":"post-2","title":"Second"}}
...
{"uri":"xdb://com.example/posts/post-100","data":{"id":"post-100","title":"Hundredth"}}
{"uri":"xdb://com.example/posts/post-101","data":{"id":"post-101","title":"Next Page"}}
```

Each page's items are emitted as they arrive. No metadata lines — `total` and `next_offset` are dropped. This is a deliberate tradeoff: NDJSON optimizes for streaming and composability, not completeness. Use JSON output if you need the envelope.

#### Batch operations

```bash
xdb batch --file operations.json --output ndjson
```

```
{"index":0,"method":"records.create","uri":"xdb://com.example/posts/post-1","status":"ok"}
{"index":1,"method":"records.create","uri":"xdb://com.example/posts/post-2","status":"ok"}
{"index":2,"method":"records.get","uri":"xdb://com.example/posts/post-1","status":"ok","data":{"title":"First"}}
```

Each batch result is emitted as its own line. On failure with rollback, all results are still emitted (including the error), plus a final trailer line:

```
{"index":0,"method":"records.create","uri":"xdb://com.example/posts/post-1","status":"ok"}
{"index":1,"method":"records.create","uri":"xdb://com.example/posts/post-2","status":"ok"}
{"index":2,"method":"records.get","uri":"xdb://com.example/posts/post-99","status":"error","error":{"code":"NOT_FOUND","message":"Record does not exist"}}
{"kind":"BatchTrailer","total":3,"succeeded":0,"failed":1,"rolled_back":true}
```

The trailer is always the last line and is identified by `"kind":"BatchTrailer"`. Consumers that need rollback status check the last line; consumers that only care about individual results process lines as they arrive and stop on first error.

#### Unix composability

```bash
# Count records
xdb records list --uri xdb://com.example/posts --output ndjson | wc -l

# First 5 records
xdb records list --uri xdb://com.example/posts --output ndjson --page-all | head -5

# Filter with jq
xdb records list --uri xdb://com.example/posts --output ndjson | jq 'select(.data.age > 30)'

# Pipe batch results to check for errors
xdb batch --file ops.json --output ndjson | jq 'select(.status == "error")'
```

### Formatter Design

Inspired by gws-cli's formatter, XDB's output layer handles four formats with pagination-aware behavior:

| Format | Single response | List / Batch | `--page-all` |
|--------|----------------|--------------|--------------|
| **JSON** | Pretty-printed with envelope | Pretty-printed with envelope (`items` array) | Each page as separate JSON object |
| **NDJSON** | N/A (falls back to JSON) | One line per item, no envelope | Streams across pages, one line per item |
| **Table** | Key-value rows | Column headers + data rows | Headers on first page only |
| **YAML** | YAML document | YAML document with sequence | `---` separator between pages |

Key behaviors:
- **First-page awareness** — table emits column headers only on the first page (gws-cli pattern). Subsequent pages append data rows only.
- **YAML document separators** — each page is prefixed with `---\n`, producing a valid multi-document YAML stream.
- **Compact vs pretty** — JSON uses compact form when piped, pretty when TTY. `--pretty` forces indented output regardless.

---

## 2. Raw JSON Payloads via `--json`

The canonical commands accept JSON payloads directly. No flag-per-field mapping needed:

```bash
# Inline JSON payload
xdb records create --uri xdb://com.example/posts/post-123 --json '{"title":"Hello","content":"World"}'

# File input
xdb records create --uri xdb://com.example/posts/post-123 --file post.json

# Stdin
cat post.json | xdb records create --uri xdb://com.example/posts/post-123

# Schema creation with inline JSON
xdb schemas create --uri xdb://com.example/posts --json '{"attrs":[{"name":"title","type":"string"}]}'
```

For `records list` filters, support a structured JSON query alongside the human-friendly `--filter`:

```bash
# Human-friendly
xdb records list --uri xdb://com.example/posts --filter "age>30"

# Agent-friendly: structured query
xdb records list --uri xdb://com.example/posts --query '{"filters":[{"attr":"age","op":">","value":30}],"limit":10}'
```

---

## 3. Schema Introspection at Runtime

Agents should never need external docs to understand XDB's data model. The CLI itself is the documentation.

### `xdb schema` command (gws-cli pattern)

Uses dot-notation to introspect methods and types:

```bash
# Method signature — params, request body, flags, mutating?
xdb schema records.create
xdb schema records.list
xdb schema schemas.get

# Type definitions
xdb schema Record
xdb schema Schema
xdb schema Namespace
xdb schema Value
```

#### Method introspection output

```bash
$ xdb schema records.create
```

```json
{
  "kind": "MethodDescription",
  "method": "records.create",
  "summary": "Create a new record. Fails if the record already exists.",
  "params": [
    {
      "name": "uri",
      "type": "string",
      "required": true,
      "description": "Record URI (xdb://NS/SCHEMA/ID)"
    }
  ],
  "input": {
    "accepts": ["json", "file", "stdin"],
    "description": "Record payload as JSON object"
  },
  "flags": [
    { "name": "json", "type": "string", "description": "Inline JSON payload" },
    {
      "name": "file",
      "short": "f",
      "type": "string",
      "description": "Path to input file"
    },
    {
      "name": "format",
      "type": "string",
      "default": "json",
      "enum": ["json", "yaml"]
    },
    {
      "name": "dry-run",
      "type": "bool",
      "description": "Validate without writing"
    },
    {
      "name": "output",
      "short": "o",
      "type": "string",
      "default": "auto",
      "enum": ["json", "table", "yaml", "ndjson"]
    }
  ],
  "response": { "kind": "Record" },
  "mutating": true,
  "examples": [
    "xdb records create --uri xdb://com.example/posts/post-123 --json '{\"title\":\"Hello\"}'"
  ]
}
```

#### Type introspection output

```bash
$ xdb schema Record
```

```json
{
  "kind": "TypeDescription",
  "type": "Record",
  "description": "A collection of tuples sharing the same ID within a schema.",
  "fields": [
    {
      "name": "uri",
      "type": "string",
      "description": "Record URI (xdb://NS/SCHEMA/ID)"
    },
    { "name": "id", "type": "string", "description": "Record identifier" },
    { "name": "schema", "type": "string", "description": "Schema name" },
    { "name": "namespace", "type": "string", "description": "Namespace" },
    {
      "name": "tuples",
      "type": "[]Tuple",
      "description": "Record data as tuples"
    },
    {
      "name": "created_at",
      "type": "timestamp",
      "description": "Creation time"
    },
    {
      "name": "updated_at",
      "type": "timestamp",
      "description": "Last update time"
    }
  ]
}
```

#### Data schema introspection

```bash
# Describe a user-defined schema — returns field names, types, constraints
xdb schema --uri xdb://com.example/posts
```

```json
{
  "kind": "DataSchemaDescription",
  "uri": "xdb://com.example/posts",
  "attrs": [
    { "name": "title", "type": "string", "required": true },
    { "name": "content", "type": "string", "required": false },
    { "name": "author.id", "type": "string", "required": true },
    { "name": "created_at", "type": "timestamp", "required": true }
  ],
  "strict": true
}
```

#### List all methods and types

```bash
# List all methods
xdb schema --methods

# List all types
xdb schema --types

# List supported value types (string, integer, float, etc.)
xdb schema --value-types
```

This replaces the need for agents to read docs. The CLI is the canonical reference.

---

## 4. Field Masks for Context Window Discipline

API responses (especially records with many attributes) can be large. Agents shouldn't pay tokens for fields they don't need.

```bash
# Return only specific attributes
xdb records get --uri xdb://com.example/posts/post-123 --fields title,author.id,created_at

# List with field mask — each record only contains selected attrs
xdb records list --uri xdb://com.example/posts --fields id,title --limit 20

# Combine with NDJSON for streaming filtered lists
xdb records list --uri xdb://com.example/posts --fields id,title --output ndjson --page-all
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

| Check           | Rule                                  | Example rejection                   |
| --------------- | ------------------------------------- | ----------------------------------- |
| Control chars   | Reject ASCII < 0x20                   | `xdb://com.example/posts\x00`       |
| Query params    | Reject `?` in URI path                | `xdb://com.example/posts?limit=10`  |
| Fragment abuse  | Only allow `#` for attribute position | `xdb://com.example/posts#../../etc` |
| Path traversal  | Reject `..` in any component          | `xdb://../../.ssh/keys`             |
| Double encoding | Reject `%` in NS/Schema/ID            | `xdb://com.example/%2e%2e/posts`    |
| Whitespace      | Reject spaces and tabs                | `xdb://com.example/posts /123`      |

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

All mutating methods support `--dry-run`:

```bash
# Validate the create without writing
xdb records create --uri xdb://com.example/posts/post-123 --json '{"title":"Hello"}' --dry-run
```

Output:

```json
{
  "dry_run": true,
  "valid": true,
  "method": "records.create",
  "uri": "xdb://com.example/posts/post-123",
  "changes": {
    "attrs_set": ["title"],
    "record_exists": false
  }
}
```

On validation failure:

```json
{
  "dry_run": true,
  "valid": false,
  "method": "records.create",
  "uri": "xdb://com.example/posts/post-123",
  "errors": [{ "attr": "title", "error": "expected string, got integer" }]
}
```

`--dry-run` performs:

1. URI validation
2. Input parsing and type checking
3. Schema validation (if strict schema)
4. Conflict detection (record exists / doesn't exist)
5. Returns what _would_ happen without executing

For `records.create`, dry-run also reports if the record already exists (which would cause the actual call to fail).

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
- Use `xdb schema <resource>.<method>` for method signatures
- Use `xdb schema --uri <uri>` for data schema details
- Use `xdb schema <TypeName>` for type definitions

## Rules

- ALWAYS use --dry-run before mutating operations (create, update, upsert, delete)
- ALWAYS confirm with user before delete commands
- ALWAYS use --fields on list operations to limit response size
- ALWAYS use --output json when parsing output programmatically
- PREFER canonical commands (xdb records create) over aliases (xdb put)
- Use --query for structured filters instead of --filter string parsing
- URI components must not contain: ?, #, %, .., control characters

## Resources & Methods

### records

- create — Create a new record (fails if exists)
- get — Retrieve a record by URI
- list — List records in a schema
- update — Update an existing record (fails if not exists)
- upsert — Create or update a record
- delete — Delete a record

### schemas

- create — Define a new schema
- get — Retrieve schema definition
- list — List schemas in a namespace
- update — Update schema definition
- delete — Delete a schema

### namespaces

- list — List all namespaces
- get — Get namespace details

### batch

- execute — Run multiple mutations in a single atomic transaction

## Installed Schemas

[live query — lists namespaces and schemas currently in the store]

## Common Patterns

### Create and verify a record

xdb records create --uri xdb://NS/SCHEMA/ID --json '{...}' --dry-run
xdb records create --uri xdb://NS/SCHEMA/ID --json '{...}'
xdb records get --uri xdb://NS/SCHEMA/ID --fields <relevant-fields>

### Explore data

xdb schema --uri xdb://NS/SCHEMA # understand data schema
xdb records list --uri xdb://NS/SCHEMA --fields id --limit 5 # sample IDs
xdb records get --uri xdb://NS/SCHEMA/ID # fetch one record

### Introspect a method before using it

xdb schema records.create # see params, flags, examples
```

Key design: the output is **dynamic**. It includes live data from the running store (installed schemas, namespaces) so the agent knows what data exists without having to discover it through trial and error. The resource/method sections are auto-generated from the cobra command tree, so they're always in sync with the binary.

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

| Skill       | Covers                                                                |
| ----------- | --------------------------------------------------------------------- |
| `crud`      | records create/get/list/update/upsert/delete — usage, flags, examples |
| `schema`    | schemas create/get/update, xdb schema — definition, introspection     |
| `filtering` | --filter, --query — operators, structured filters, combining          |
| `batch`     | batch command — transaction semantics, operation format, limits       |
| `bulk`      | --page-all, --output ndjson, --fields — streaming, field masks        |
| `uri`       | URI format, components, valid/invalid examples, addressing            |

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
  main.go                # Entry point

internal/cli/
  app.go                 # Root command setup, global flags, alias registration
  records.go             # records create/get/list/update/upsert/delete
  schemas.go             # schemas create/get/list/update/delete
  namespaces.go          # namespaces list/get
  batch.go               # batch execution (multi-op transactions)
  schema_inspect.go      # xdb schema (method + type introspection)
  context.go             # context command (full agent context)
  skills.go              # skills command (list + get skills)
  daemon.go              # daemon start/stop/status/restart
  aliases.go             # get/put/ls/rm → canonical command resolution

internal/cli/output/
  output.go              # Output interface + format detection
  json.go                # JSON/NDJSON formatter
  table.go               # Table formatter
  yaml.go                # YAML formatter

internal/cli/validate/
  validate.go            # URI + file path + payload validation
```

---

## Implementation Phases

### Phase 1: Foundation

1. `internal/cli/validate/` — URI hardening, file path sandboxing, payload validation
2. `internal/cli/output/` — JSON, NDJSON, table, YAML formatters with TTY detection
3. `internal/cli/app.go` — Root command with global flags + resource subcommands
4. `cmd/xdb/main.go` — Entry point

### Phase 2: Core Resource Commands

5. `records` — create, get, list, update, upsert, delete with `--json`, `--file`, `--dry-run`, `--fields`
6. `schemas` — create, get, list, update, delete with `--json`, `--dry-run`
7. `namespaces` — list, get
8. `batch` — multi-operation atomic transactions with `--json`, `--file`, `--dry-run`
9. `aliases.go` — `get`, `put`, `ls`, `rm`, `make-schema` → canonical command routing

### Phase 3: Introspection & Safety

10. `xdb schema` — method introspection (dot-notation), type definitions, data schema describe
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

| Decision          | Choice                             | Rationale                                                           |
| ----------------- | ---------------------------------- | ------------------------------------------------------------------- |
| Command structure | RPC-method (`resource method`)     | 1:1 with operations, explicit semantics, agent-predictable          |
| Human aliases     | `get/put/ls/rm` with URI inference | Keeps human DX, resolves to same code path                          |
| Create vs Update  | Separate methods                   | Agents know exactly what will happen, no upsert ambiguity           |
| Introspection     | `xdb schema` (dot-notation)        | gws-cli pattern — methods, types, data schemas from one command     |
| CLI framework     | `cobra`                            | Standard, supports subcommands, completion, good help generation    |
| Config binding    | `viper` or manual                  | Flags > config file > defaults; `XDB_CONFIG` for file location only |
| JSON output       | `encoding/json`                    | Stdlib, no external dep needed                                      |
| Table output      | `tablewriter` or custom            | Minimal dep                                                         |
| Validation        | Custom in `internal/cli/validate/` | XDB-specific rules, no generic framework needed                     |
| `--dry-run` scope | All mutating methods               | Defense-in-depth for agents                                         |
| NDJSON            | Custom writer                      | One `json.Marshal` + newline per record                             |

---

## What XDB Does NOT Need (from the blog post)

- **Response sanitization (`--sanitize`)** — XDB stores user-defined data, not third-party content with prompt injection risk. If needed later, add at the application layer, not the CLI.
- **OAuth / browser auth** — XDB is local-first (daemon on localhost). Config file is sufficient.
- **Skill files on disk / OpenClaw registry** — Skills are compiled into the binary and served via `xdb skills`. No external files to install or manage.
- **Dynamic Discovery Documents** — XDB schemas _are_ the discovery mechanism. `xdb schema` serves this purpose.
- **MCP surface** — Out of scope for now. Can be added later as a thin layer over the resource commands.
- **Personas / Recipes** — XDB is a data tool with atomic operations, not a multi-service productivity suite.
