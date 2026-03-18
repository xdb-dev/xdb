# Agent-First CLI Plan

**Date:** 2026-03-09
**Status:** Draft
**Reference:** [Rewrite Your CLI for AI Agents](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)
**Inspiration:** [Google Workspace CLI](https://github.com/googleworkspace/cli) — RPC-method command structure

## Current State

- `core/` package exists (URI, Tuple, Record, Value, Type, Builder)
- `schema/` package exists (Def, Attr, validation)
- `store/` interfaces exist (RecordStore, SchemaStore, NamespaceReader, Store, BatchExecutor, HealthChecker)
- 4 store backends implemented: `xdbmemory`, `xdbfs`, `xdbredis`, `xdbsqlite`
- `encoding/xdbjson/` exists (JSON encoder/decoder for records)
- `tests/` shared test suites exist for store implementations
- No CLI, no daemon, no API layer
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

- `create` is **idempotent** — if the record already exists, returns the existing record (no error). On new record, returns the created record. Safe to retry after timeout
- `update` uses **patch semantics** — only supplied fields are modified, unspecified fields are preserved. Fails if record doesn't exist (`NOT_FOUND`). Returns the full record after merge
- `upsert` creates or full-replaces (equivalent to old `put`). Returns the record on success. Use when you want to set the complete state regardless of what existed before
- `delete` is **idempotent** — succeeds even if the record doesn't exist (no `NOT_FOUND` error). Always requires `--force` (both TTY and non-TTY). Without `--force`, exits with error

**Verb semantics summary:**

| Verb | Exists? | Behavior | Idempotent? |
|------|---------|----------|-------------|
| `create` | No | Create record | Yes |
| `create` | Yes | Return existing record (no error) | Yes |
| `update` | No | `NOT_FOUND` error | N/A |
| `update` | Yes | Patch: merge supplied fields over existing | Yes |
| `upsert` | No | Create record (full state) | Yes |
| `upsert` | Yes | Full replace (all fields) | Yes |
| `delete` | No | Success (no-op) | Yes |
| `delete` | Yes | Delete record | Yes |

**When to use each:**
- `create` — "I'm making something new, but don't fail if I already made it" (retry-safe)
- `update` — "Change these specific fields on an existing record" (partial merge)
- `upsert` — "Set this record to exactly this state, I don't care what was there" (full replace)
- `delete` — "Make sure this record is gone" (retry-safe)

#### Schemas

```bash
xdb schemas create  --uri xdb://com.example/posts --json '{...}'
xdb schemas get     --uri xdb://com.example/posts
xdb schemas list    --uri xdb://com.example
xdb schemas update  --uri xdb://com.example/posts --json '{...}'
xdb schemas delete  --uri xdb://com.example/posts --force
xdb schemas delete  --uri xdb://com.example/posts --cascade  # deletes schema + all records
```

- `create` is **idempotent** — if the schema already exists, returns the existing schema (no error)
- `update` uses **patch semantics** — merges supplied attrs over existing definition. Fails if schema doesn't exist
- `delete` is **idempotent** — succeeds even if schema doesn't exist. Refuses if records exist; use `--cascade` to delete schema + all records
- `delete` always requires `--force` (same as record delete)

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

#### Watch

Stream change notifications as NDJSON. Turns XDB from a passive store into an event source for agent workflows.

```bash
# Watch all changes in a schema
xdb watch --uri xdb://com.example/posts --output ndjson

# Watch a specific record
xdb watch --uri xdb://com.example/posts/post-123 --output ndjson
```

Output (one line per event):

```
{"type":"create","uri":"xdb://com.example/posts/post-1","data":{"title":"Hello"}}
{"type":"update","uri":"xdb://com.example/posts/post-1","data":{"title":"Updated"}}
{"type":"delete","uri":"xdb://com.example/posts/post-1"}
```

**Lifecycle events:**

- `{"type":"connected"}` — emitted on initial connection and after reconnect
- `{"type":"reconnecting"}` — emitted when connection lost, before retry
- `{"type":"deleted","uri":"xdb://com.example/posts"}` — schema was deleted; watch exits with code 0

**Reconnection:** On daemon restart or connection loss, the client retries with exponential backoff (1s, 2s, 4s, max 30s). A `reconnecting` event is emitted so consumers know.

**Limits:** Max 10 watchers per schema, max 50 total (configurable in config file). Excess connections get an error: `"too many watchers (max 50)"`.

#### Import / Export

Bulk data movement for loading and dumping datasets.

```bash
# Import NDJSON file (auto-chunks into batches of 100)
xdb import --uri xdb://com.example/posts --file data.ndjson

# Import from stdin
cat data.ndjson | xdb import --uri xdb://com.example/posts

# Export all records as NDJSON
xdb export --uri xdb://com.example/posts --output ndjson > backup.ndjson

# Export with field mask
xdb export --uri xdb://com.example/posts --fields id,title --output ndjson
```

**Import behavior:**

- Reads NDJSON (one JSON object per line) or JSON array
- Auto-chunks into batches of 100 operations, each sent as `batch.execute`
- Each chunk is atomic (all-or-nothing), but chunks are independent
- Progress to stderr: `Imported 100/10000... 200/10000...`
- On chunk failure: reports which chunk failed and how many records committed before it
- Uses `records.upsert` by default; `--create-only` flag uses `records.create` (fails on duplicates)

**Export behavior:**

- Thin wrapper over `records.list --page-all --output ndjson`
- Streams across all pages without buffering

#### Init

First-run scaffolding command:

```bash
xdb init    # Creates ~/.xdb/, config file, starts daemon
```

Creates `~/.xdb/` directory, writes default `config.json`, and starts the daemon. Idempotent — safe to run multiple times.

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

### Exit Codes

| Code | Meaning | Example |
|------|---------|---------|
| 0 | Success | Record created, list returned |
| 1 | Application error | NOT_FOUND, ALREADY_EXISTS, SCHEMA_VIOLATION |
| 2 | Connection error | Daemon not running, timeout |
| 3 | Input validation error | INVALID_URI, bad JSON, mutual exclusivity |
| 4 | Internal error | Store failure, unexpected daemon error |

### Auto-detection (already in README spec)

- **TTY** → table format (human-readable)
- **Pipe/redirect** → JSON (machine-readable, compact by default)
- `--output json|table|yaml` to override

### Agent additions

- `--output ndjson` — one JSON object per line for streaming. Available on list and batch operations.
- `--quiet` — suppress all stdout output. Only exit code matters. Useful for shell conditionals: `if xdb records get --uri ... --quiet; then ...`
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

| Format     | Single response              | List / Batch                                 | `--page-all`                            |
| ---------- | ---------------------------- | -------------------------------------------- | --------------------------------------- |
| **JSON**   | Pretty-printed with envelope | Pretty-printed with envelope (`items` array) | Each page as separate JSON object       |
| **NDJSON** | N/A (falls back to JSON)     | One line per item, no envelope               | Streams across pages, one line per item |
| **Table**  | Key-value rows               | Column headers + data rows                   | Headers on first page only              |
| **YAML**   | YAML document                | YAML document with sequence                  | `---` separator between pages           |

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

**Input mutual exclusivity:** `--json`, `--file`, and stdin are mutually exclusive. If more than one is provided, error: `"Error: specify only one of --json, --file, or stdin"` (exit code 3).

For `records list` filters, support a structured JSON query alongside the human-friendly `--filter`:

```bash
# Human-friendly
xdb records list --uri xdb://com.example/posts --filter "age>30"

# Agent-friendly: structured query
xdb records list --uri xdb://com.example/posts --query '{"filters":[{"attr":"age","op":">","value":30}],"limit":10}'
```

`--filter` and `--query` are mutually exclusive. If both are provided, error: `"Error: specify only one of --filter or --query"` (exit code 3).

---

## 3. Schema Introspection at Runtime

Agents should never need external docs to understand XDB's data model. The CLI itself is the documentation.

### `xdb describe` command (gws-cli pattern)

Uses dot-notation to introspect methods and types:

```bash
# Method signature — params, request body, flags, mutating?
xdb describe records.create
xdb describe records.list
xdb describe schemas.get

# Type definitions
xdb describe Record
xdb describe Schema
xdb describe Namespace
xdb describe Value
```

#### Method introspection output

```bash
$ xdb describe records.create
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
$ xdb describe Record
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
xdb describe --uri xdb://com.example/posts
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
xdb describe --methods

# List all types
xdb describe --types

# List supported value types (string, integer, float, etc.)
xdb describe --value-types
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

When the record already exists (idempotent create):

```json
{
  "dry_run": true,
  "valid": true,
  "method": "records.create",
  "uri": "xdb://com.example/posts/post-123",
  "changes": {
    "record_exists": true,
    "action": "return_existing"
  }
}
```

For `records.update` (patch), dry-run shows the merge diff:

```json
{
  "dry_run": true,
  "valid": true,
  "method": "records.update",
  "uri": "xdb://com.example/posts/post-123",
  "changes": {
    "fields_modified": ["title"],
    "fields_unchanged": ["content", "author.id", "created_at"]
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
4. Existence check (reports `record_exists` for create, `NOT_FOUND` for update)
5. For update: reports which fields would be modified vs unchanged
6. Returns what _would_ happen without executing

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
- Use `xdb describe <resource>.<method>` for method signatures
- Use `xdb describe --uri <uri>` for data schema details
- Use `xdb describe <TypeName>` for type definitions

## Rules

- ALWAYS use --dry-run before mutating operations (create, update, upsert, delete)
- ALWAYS confirm with user before delete commands
- ALWAYS use --fields on list operations to limit response size
- ALWAYS use --output json when parsing output programmatically
- PREFER canonical commands (xdb records create) over aliases (xdb put)
- PREFER `update` for modifying specific fields (patch merge). Use `upsert` only when you want to set the complete record state
- Use --query for structured filters instead of --filter string parsing
- URI components must not contain: ?, #, %, .., control characters

## Resources & Methods

### records

- create — Create a new record. Idempotent: returns existing if already exists
- get — Retrieve a record by URI
- list — List records in a schema
- update — Patch: merge supplied fields over existing record (fails if not exists)
- upsert — Full replace: set the complete record state (creates if not exists)
- delete — Delete a record. Idempotent: succeeds even if not found

### schemas

- create — Define a new schema. Idempotent: returns existing if already exists
- get — Retrieve schema definition
- list — List schemas in a namespace
- update — Patch: merge supplied attrs over existing schema definition
- delete — Delete a schema. Idempotent: succeeds even if not found

### namespaces

- list — List all namespaces
- get — Get namespace details

### batch

- execute — Run multiple mutations in a single atomic transaction

## Installed Schemas

[live query — lists namespaces and schemas currently in the store]

## Common Patterns

### Create a record (safe to retry)

xdb records create --uri xdb://NS/SCHEMA/ID --json '{...}' --dry-run
xdb records create --uri xdb://NS/SCHEMA/ID --json '{...}'

### Modify specific fields on a record

xdb records update --uri xdb://NS/SCHEMA/ID --json '{"title":"New Title"}' --dry-run
xdb records update --uri xdb://NS/SCHEMA/ID --json '{"title":"New Title"}'

### Set a record to an exact state (full replace)

xdb records upsert --uri xdb://NS/SCHEMA/ID --json '{full record...}'

### Explore data

xdb describe --uri xdb://NS/SCHEMA # understand data schema
xdb records list --uri xdb://NS/SCHEMA --fields id --limit 5 # sample IDs
xdb records get --uri xdb://NS/SCHEMA/ID # fetch one record

### Introspect a method before using it

xdb describe records.create # see params, flags, examples
```

Key design: the output is **dynamic**. It includes live data from the running store (installed schemas, namespaces) so the agent knows what data exists without having to discover it through trial and error. The resource/method sections are auto-generated from the urfave/cli command tree, so they're always in sync with the binary.

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
| `schema`    | schemas create/get/update, xdb describe — definition, introspection   |
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
  "log": { "level": "info", "file": "xdb.log" },
  "store": { "backend": "sqlite", "sqlite": { "name": "xdb.db" } },
  "watch": { "max_per_schema": 10, "max_total": 50 }
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

The CLI/daemon is a **separate Go module** (`github.com/xdb-dev/xdb/cmd/xdb`) to keep the root module clean for library consumers. The root module contains only library packages (`core/`, `schema/`, `store/`, `client/`, `filter/`). CLI-specific dependencies (urfave/cli, etc.) don't pollute library imports.

### Root module (github.com/xdb-dev/xdb) — library

```
core/              # URI, Tuple, Record, Value, Type (existing)
schema/            # Schema definitions and validation (existing)
store/             # Store interfaces + backends (existing)
encoding/xdbjson/  # JSON encoder/decoder (existing)
client/            # Go client library (JSON-RPC client, wire types)
filter/            # Filter parsing — minimal grammar (attr OP value)
```

### CLI/daemon module (github.com/xdb-dev/xdb/cmd/xdb)

```
cmd/xdb/
  main.go                    # Entry point

  internal/cli/
    app.go                   # Root command setup, global flags, alias registration
    records.go               # records create/get/list/update/upsert/delete
    schemas.go               # schemas create/get/list/update/delete
    namespaces.go            # namespaces list/get
    batch.go                 # batch execution (multi-op transactions)
    watch.go                 # watch command (streaming changes)
    import_export.go         # import/export commands (bulk data)
    init.go                  # init command (first-run scaffolding)
    describe.go              # xdb describe (method + type introspection)
    context.go               # context command (full agent context)
    skills.go                # skills command (list + get skills)
    daemon.go                # daemon start/stop/status/restart
    aliases.go               # get/put/ls/rm → canonical command resolution

  internal/cli/output/
    output.go                # Output interface + format detection
    json.go                  # JSON/NDJSON formatter
    table.go                 # Table formatter
    yaml.go                  # YAML formatter

  internal/cli/validate/
    validate.go              # URI + file path + payload validation

  internal/rpc/
    request.go               # JSON-RPC 2.0 request/response types
    handler.go               # HTTP handler (parse, dispatch, respond)
    router.go                # Method → handler mapping
    errors.go                # Error codes and constructors

  internal/service/
    service.go               # Service struct, constructor
    records.go               # records.* method handlers
    schemas.go               # schemas.* method handlers
    namespaces.go            # namespaces.* method handlers
    batch.go                 # batch.execute handler
    watch.go                 # watch pubsub fan-out
    introspect.go            # schema.describe_* handlers
    system.go                # system.* handlers

  internal/daemon/
    daemon.go                # Daemon lifecycle (start, stop, listeners)
    pid.go                   # PID file management
```

### Security: File permissions

- `~/.xdb/` directory: `0700`
- `~/.xdb/xdb.sock` socket: `0600`
- `~/.xdb/config.json`: `0600`
- `~/.xdb/xdb.log`: `0600`

---

## Pre-Work (store interface changes)

Before CLI implementation begins:

1. **Add `Close() error` to `store.Store` interface** — required for daemon shutdown. All 4 backends implement it (SQLite closes DB, Redis closes pool, memory/fs are no-ops).
2. **Add `Fields []string` to `store.ListQuery`** — store-level field projection. SQLite implements as SELECT column list; others filter post-fetch.

## Implementation Phases

### Phase 1: Foundation

1. `filter/` package — minimal filter grammar (attr OP value), AST, evaluation. Extend later with AND/OR/parens
2. `internal/cli/validate/` — URI hardening, file path sandboxing, payload validation
3. `internal/cli/output/` — JSON, NDJSON, table, YAML formatters with TTY detection
4. `internal/cli/app.go` — Root command with global flags + resource subcommands
5. `cmd/xdb/main.go` — Entry point
6. `xdb init` — first-run scaffolding (~/.xdb/, config, daemon start)

### Phase 2: Core Resource Commands

7. `records` — create, get, list, update, upsert, delete with `--json`, `--file`, `--dry-run`, `--fields`, `--quiet`
8. `schemas` — create, get, list, update, delete with `--json`, `--dry-run`, `--cascade`
9. `namespaces` — list, get
10. `batch` — multi-operation atomic transactions with `--json`, `--file`, `--dry-run`
11. `import` / `export` — bulk data movement (NDJSON, auto-chunking)
12. `aliases.go` — `get`, `put`, `ls`, `rm`, `make-schema` → canonical command routing

### Phase 3: Introspection, Safety & Streaming

13. `xdb describe` — method introspection (dot-notation), type definitions, data schema describe
14. `xdb watch` — streaming change notifications (NDJSON, reconnection, lifecycle events)
15. Structured error responses (JSON error envelope with exit codes)
16. Input hardening test suite (fuzz with agent-typical hallucinations)

### Phase 4: Agent Surfaces

17. `xdb context` command (dynamic agent context generation)
18. `xdb skills` / `xdb skills get <name>` (compiled-in skill documents)
19. Daemon with Unix socket + HTTP, structured logging to `~/.xdb/xdb.log`

### Phase 5: Distribution

20. `goreleaser` config for GitHub releases with prebuilt binaries
21. Homebrew tap (`xdb-dev/tap/xdb`)
22. `go install github.com/xdb-dev/xdb/cmd/xdb@latest` verification

---

## Key Decisions

| Decision               | Choice                             | Rationale                                                           |
| ---------------------- | ---------------------------------- | ------------------------------------------------------------------- |
| Command structure      | RPC-method (`resource method`)     | 1:1 with operations, explicit semantics, agent-predictable          |
| Human aliases          | `get/put/ls/rm` with URI inference | Keeps human DX, resolves to same code path                          |
| Create vs Update       | Separate methods, distinct semantics | `create` = idempotent insert, `update` = patch merge, `upsert` = full replace |
| Introspection          | `xdb describe` (dot-notation)       | gws-cli pattern — methods, types, data schemas from one command     |
| CLI framework          | `urfave/cli v3`                    | Simple API, built-in completion, sufficient for XDB's command tree  |
| Module structure       | `cmd/xdb/` as separate Go module   | Root module stays clean for library consumers (no CLI deps)         |
| Config binding         | Manual JSON + flags                | Flags > config file > defaults; `XDB_CONFIG` for file location only |
| JSON output            | `encoding/json`                    | Stdlib, no external dep needed                                      |
| Table output           | `tablewriter` or custom            | Minimal dep                                                         |
| Validation             | Custom in `internal/cli/validate/` | XDB-specific rules, no generic framework needed                     |
| Filter parsing         | `filter/` package in root module   | Minimal grammar (attr OP value), reusable across CLI and service    |
| Field masking          | Store-level projection             | `Fields []string` in `ListQuery`; backends implement column select  |
| `--dry-run` scope      | All mutating methods               | Defense-in-depth for agents                                         |
| NDJSON                 | Custom writer                      | One `json.Marshal` + newline per record                             |
| Delete safety          | Idempotent + always require `--force` | Idempotent (no error if gone) for retry safety; `--force` for intent |
| Schema delete + records| Error unless `--cascade`           | Refuses if records exist; `--cascade` deletes schema + all records  |
| Mutation responses     | Return full record                 | All mutations return the full record (after merge for update)       |
| Idempotency           | create + delete are idempotent     | Retry-safe: create returns existing, delete succeeds if gone        |
| Update semantics      | Patch (partial merge)              | Only supplied fields change; tuple model naturally supports merge   |
| Import chunking        | Auto-batch of 100                  | Each chunk atomic, chunks independent. Progress to stderr           |
| Distribution           | go install + goreleaser + Homebrew | Full distribution: `brew install`, GitHub releases, `go install`    |

---

## What XDB Does NOT Need (from the blog post)

- **Response sanitization (`--sanitize`)** — XDB stores user-defined data, not third-party content with prompt injection risk. If needed later, add at the application layer, not the CLI.
- **OAuth / browser auth** — XDB is local-first (daemon on localhost). Config file is sufficient.
- **Skill files on disk / OpenClaw registry** — Skills are compiled into the binary and served via `xdb skills`. No external files to install or manage.
- **Dynamic Discovery Documents** — XDB schemas _are_ the discovery mechanism. `xdb describe` serves this purpose.
- **MCP surface** — Explicitly excluded from scope. Can be added later as a thin layer over the resource commands.
- **Personas / Recipes** — XDB is a data tool with atomic operations, not a multi-service productivity suite.
- **Embedded mode** — The CLI always talks to a running daemon. No in-process store fallback. If the daemon isn't running, error: `"daemon not running — start it with: xdb daemon start"` (exit code 2).
- **Prometheus metrics** — Structured access logs are sufficient for a local daemon. Add metrics when XDB supports remote/multi-user.
- **Audit / transaction log** — Records have `created_at`/`updated_at`. Full mutation history deferred to a future plan.
