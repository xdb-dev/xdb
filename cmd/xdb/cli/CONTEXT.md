# XDB CLI (`xdb`) Context

XDB is an agent-first data layer. Model once, store anywhere. All data is addressed by URI: `xdb://NAMESPACE/SCHEMA/ID#ATTR`.

## Rules of Engagement for Agents

* **Schema Discovery:** *If you don't know the data structure, run `xdb describe --uri xdb://NS/SCHEMA` first to inspect the schema before creating records.*
* **Context Window Protection:** *Record lists can be large. ALWAYS use `--fields` to select only the attributes you need, and `--limit` to cap result size.*
* **Dry-Run Safety:** *Always use `--dry-run` for mutating operations (create, update, upsert, delete) to validate before actual execution.*
* **Introspection First:** *Run `xdb describe <resource>.<method>` to inspect any method's parameters before using it.*

## Core Syntax

```bash
xdb <resource> <method> [flags]
```

Use `--help` at any level:

```bash
xdb --help
xdb records --help
xdb records create --help
```

### Key Flags

- `--uri <URI>`: Target resource URI (e.g., `xdb://com.example/posts/post-1`).
- `--json '<JSON>'`: Inline JSON payload for create/update/upsert.
- `--file <PATH>`, `-f`: Read payload from a file instead of `--json`.
- `--fields <MASK>`: Comma-separated field mask (critical for AI context window efficiency).
- `--output <FMT>`, `-o`: Output format: `json` (default for pipes), `table` (default for TTY), `yaml`, `ndjson`.
- `--dry-run`: Validate without writing.
- `--force`: Required for all delete operations.
- `--quiet`: Suppress output; only exit code matters.
- `--limit`, `--offset`: Pagination controls for list operations.

## Usage Patterns

### 1. Reading Data (get / list)

Always use `--fields` to minimize tokens.

```bash
# Get a single record
xdb records get --uri xdb://com.example/posts/post-1 --fields title,author

# List records with filter
xdb records list --uri xdb://com.example/posts --filter "status=published" --fields id,title --limit 10

# List schemas in a namespace
xdb schemas list --uri xdb://com.example
```

### 2. Writing Data (create / update / upsert)

Use `--json` for the request body. Always `--dry-run` first.

```bash
# Create a record (idempotent — safe to retry)
xdb records create --uri xdb://com.example/posts/post-1 --json '{"title":"Hello","author":"Alice"}' --dry-run
xdb records create --uri xdb://com.example/posts/post-1 --json '{"title":"Hello","author":"Alice"}'

# Update specific fields (patch merge — unspecified fields preserved)
xdb records update --uri xdb://com.example/posts/post-1 --json '{"title":"Updated Title"}'

# Full replace (upsert — sets complete state)
xdb records upsert --uri xdb://com.example/posts/post-1 --json '{"title":"Complete","author":"Bob"}'
```

### 3. Streaming & Bulk (ndjson / import / export)

Use `--output ndjson` for streaming, `import`/`export` for bulk.

```bash
# Export all records as NDJSON
xdb export --uri xdb://com.example/posts --fields id,title

# Import from NDJSON
cat records.ndjson | xdb import --uri xdb://com.example/posts

# Stream list as NDJSON for piping
xdb records list --uri xdb://com.example/posts --output ndjson | jq 'select(.title != null)'
```

### 4. Schema Introspection

If unsure about parameters or data structure, introspect:

```bash
# Inspect a method
xdb describe records.create

# Inspect a type
xdb describe Record

# List all methods
xdb describe --methods

# Describe a data schema
xdb describe --uri xdb://com.example/posts
```

### 5. Human Aliases

Aliases infer the resource from URI depth:

```bash
xdb get xdb://com.example/posts/post-1       # → records get
xdb get xdb://com.example/posts              # → schemas get
xdb ls xdb://com.example/posts               # → records list
xdb ls xdb://com.example                     # → schemas list
xdb ls                                       # → namespaces list
xdb put xdb://com.example/posts/post-1 ...   # → records upsert
xdb rm xdb://com.example/posts/post-1 --force # → records delete
```
