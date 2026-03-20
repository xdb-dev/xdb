# XDB

XDB is an agent-first data layer. Model once, store anywhere. Simple URIs, structured tuples, and a pipe-friendly CLI that agents and humans both get right on the first try.

## Why XDB?

Read about the motivation behind XDB in [Introducing XDB](https://raviatluri.in/articles/introducing-xdb).

## Core Concepts

> For in-depth documentation on each concept, see [docs/concepts](./docs/concepts/).

The XDB data model can be visualized as a tree of **Namespaces**, **Schemas**, **Records**, and **Tuples**.

```
┌─────────────────────────────────┐
│            Namespace            │
└────────────────┬────────────────┘
                 ↓
┌─────────────────────────────────┐
│             Schema              │
└────────────────┬────────────────┘
                 ↓
┌─────────────────────────────────┐
│             Record              │
└────────────────┬────────────────┘
                 ↓
┌─────────────────────────────────┐
│             Tuple               │
├─────────────────────────────────┤
│   ID | Attr | Value | Options   │
└─────────────────────────────────┘
```

### Tuple

A **Tuple** is the fundamental building block in XDB. It combines:

- ID: a string that uniquely identifies the record
- Attr: a string that identifies the attribute. It supports dot-separated nesting.
- Value: The attribute's value
- Options: Key-value pairs for metadata

![tuple.png](./docs/tuple.png)

### Record

One or more **Tuples**, with the same **ID**, make up a **Record**. Records are similar to objects, structs, or rows in a database. Records typically represent a single entity or object of domain data.

### Namespace

A **Namespace** (NS) groups one or more **Schemas**. Namespaces are typically used to organize schemas by domain, application, or tenant.

### Schema

A **Schema** defines the structure of records and groups them together. Schemas can be "strict" or "flexible". Strict schemas enforce a predefined structure on the data, while flexible schemas allow for arbitrary data. Each schema is uniquely identified by its name within a namespace.

### URI

XDB URIs are valid Uniform Resource Identifiers (URI) according to [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986). URIs are used to uniquely identify resources in XDB.

The general format of a URI is:

```
    [SCHEME]://[DOMAIN] [ "/" PATH] [ "?" QUERY] [ "#" FRAGMENT]
```

XDB URIs follow the following format:

```
    xdb:// NS [ "/" SCHEMA ] [ "/" ID ] [ "#" ATTRIBUTE ]
```

```
    xdb://com.example/posts/123-456-789#author.id
    └─┬──┘└────┬────┘└──┬─┘└─────┬─────┘└─────┬─────┘
   scheme     NS    SCHEMA      ID        ATTRIBUTE
              └───────────┬───────────┘
                       path
```

The components of the URI are:

- **NS**: The namespace.
- **SCHEMA**: The schema name.
- **ID**: The unique identifier of the record.
- **ATTRIBUTE**: The name of the attribute.
- **path**: NS, SCHEMA, and ID combined uniquely identify a record (URI without xdb://)

Valid examples:

```
Namespace:  xdb://com.example
Schema:     xdb://com.example/posts
Record:     xdb://com.example/posts/123-456-789
Attribute:  xdb://com.example/posts/123-456-789#author.id
```

## Supported Types

| Type       | PostgreSQL         | SQLite    | Description                  |
| ---------- | ------------------ | --------- | ---------------------------- |
| `string`   | `TEXT`             | `TEXT`    | UTF-8 string                 |
| `integer`  | `BIGINT`           | `INTEGER` | 64-bit signed integer        |
| `unsigned` | `BIGINT`           | `INTEGER` | 64-bit unsigned integer      |
| `float`    | `DOUBLE PRECISION` | `REAL`    | 64-bit floating point        |
| `boolean`  | `BOOLEAN`          | `INTEGER` | True or false                |
| `time`     | `TIMESTAMPTZ`      | `INTEGER` | Date and time in UTC         |
| `json`     | `JSONB`            | `TEXT`    | Arbitrary JSON data          |
| `bytes`    | `BYTEA`            | `BLOB`    | Binary data                  |
| `array`    | `[]T`              | `TEXT`    | Array of typed values        |

## Getting Started

### Installation

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
```

#### Check Version

```bash
xdb --version
```

### Quick Start

```bash
# Create schema
xdb make-schema xdb://com.example/posts --schema posts.json

# List schemas
xdb ls xdb://com.example

# Get schema
xdb get xdb://com.example/posts

# Remove schema
xdb rm xdb://com.example/posts

# Put record
xdb put xdb://com.example/posts/post-123 --file post.json

# Get record
xdb get xdb://com.example/posts/post-123

# List records
xdb ls xdb://com.example/posts

# Filter records
xdb ls xdb://com.example/posts --filter 'title.contains("Hello")'

# Remove record
xdb rm xdb://com.example/posts/post-123
```

## CLI Reference

### Commands

#### Make Schema

```bash
# Create schema from definition file
xdb make-schema xdb://com.example/posts --schema ./posts.json

# Create flexible schema (no definition file)
xdb make-schema xdb://com.example/users
```

`make-schema` creates or updates a schema at the given URI. The URI must include both NS and Schema components (e.g., `xdb://com.example/posts`).

If no schema definition file is provided, a flexible schema is created. Flexible schemas allow arbitrary data without validation. Schema definitions enforce structure and types on all Tuple and Record operations.

**Note:** You can update a schema by calling `make-schema` again with the same URI and a new schema definition.

#### List

```bash
# List schemas in namespace
xdb ls xdb://com.example

# List with pagination
xdb ls xdb://com.example/users --limit 10
xdb ls xdb://com.example/users --limit 10 --offset 20

# List all (default limit is 100)
xdb ls xdb://com.example

# Filter records (CEL expressions)
xdb ls xdb://com.example/users --filter 'age > 30'
xdb ls xdb://com.example/users --filter 'name == "alice"'

# Compound filters
xdb ls xdb://com.example/users --filter 'age >= 25 && status != "inactive"'

# String functions
xdb ls xdb://com.example/posts --filter 'title.contains("hello")'

# Combine filters with pagination
xdb ls xdb://com.example/users --filter 'age > 30' --limit 10
```

`ls` lists namespaces, schemas, or records depending on the URI. If no URI is provided, it lists all namespaces. Filters are only applied when listing records.

**Flags:**

- `--limit N`: Maximum number of results to return (default: 100)
- `--offset N`: Number of results to skip (default: 0)
- `--filter EXPR`: CEL filter expression ([AIP-160](https://google.aip.dev/160)). Operators: `==`, `!=`, `>`, `>=`, `<`, `<=`, `&&`, `||`, `!`, `in`. Functions: `.contains()`, `.startsWith()`, `.endsWith()`, `size()`.

#### Get

```bash
# Get Namespace
xdb get xdb://com.example

# Get Schema
xdb get xdb://com.example/posts

# Get Record
xdb get xdb://com.example/posts/post-123

# Get Attribute
xdb get xdb://com.example/posts/post-123#title
```

`get` retrieves a Namespace, Schema, Record, or Attribute from the given URI.

#### Put

```bash
# Put Record from JSON file
xdb put xdb://com.example/posts/post-123 --file post.json

# Put Record from stdin (JSON)
echo '{"title":"Hello","content":"World"}' | xdb put xdb://com.example/posts/post-123

# Put Record from YAML file
xdb put xdb://com.example/posts/post-123 --file post.yaml --format yaml

# Put with explicit format
echo 'title: Hello
content: World' | xdb put xdb://com.example/posts/post-123 --format yaml
```

`put` creates or updates a Record at the given URI from a JSON or YAML file, or from stdin.

**Flags:**

- `--file`, `-f`: Path to file (reads from stdin if omitted)
- `--format`: Input format: json (default) or yaml

#### Remove

```bash
# Remove record (with confirmation prompt)
xdb remove xdb://com.example/posts/post-123

# Remove without confirmation
xdb rm xdb://com.example/posts/post-123 --force

# Remove schema and all its records
xdb rm xdb://com.example/posts --force --cascade
```

`remove` (aliases: `rm`, `delete`) deletes a Record, Attribute, or Schema at the given URI.

**Flags:**

- `--force`, `-f`: Skip confirmation prompt
- `--cascade`: Delete schema and all its records (schema delete only)

#### Daemon

```bash
# Start the daemon
xdb daemon start

# Check daemon status
xdb daemon status

# Stop the daemon
xdb daemon stop

# Force stop the daemon
xdb daemon stop --force

# Restart the daemon
xdb daemon restart
```

The daemon runs an HTTP server that handles all XDB operations. By default it listens on `localhost:8147` and a Unix socket at `~/.xdb/xdb.sock`.

### Global Flags

These flags are available for all commands:

- `--output`, `-o`: Output format (json, table, yaml). Auto-detected by default (table for TTY, JSON for pipes)
- `--config`, `-c`: Path to config file (defaults to `~/.xdb/config.json`)
- `--verbose`, `-v`: Enable verbose logging (INFO level)
- `--debug`: Enable debug logging with source locations

### Output Formats

#### Automatic Format Selection

By default, XDB automatically selects the appropriate format:

- **Table format**: When output is a terminal (TTY) - human-readable tables
- **JSON format**: When output is piped or redirected - machine-parseable

#### Available Formats

- `json`: Indented JSON output (machine-readable)
- `table`: Human-readable tables with borders
- `yaml`: YAML output

#### Example

```bash
# Uses table format (interactive terminal)
xdb get xdb://com.example/users/123

# Uses JSON format (piped to jq)
xdb get xdb://com.example/users/123 | jq '.name'

# Force JSON output even in terminal
xdb get xdb://com.example/users/123 --output json

# Force table output even when piping
xdb ls xdb://com.example/users --output table
```

### Piping and Automation

XDB commands work well with Unix pipes:

```bash
# Get record and pipe to jq
xdb get xdb://com.example/posts/post-123 | jq '.title'

# List schemas and count them
xdb ls xdb://com.example | jq '. | length'

# Export records to file
xdb get xdb://com.example/posts/post-123 --output json > backup.json

# Import from file
cat backup.json | xdb put xdb://com.example/posts/post-123
```

## Configuration

XDB uses a JSON config file at `~/.xdb/config.json`. A default config is created automatically on first run.

```json
{
  "dir": "~/.xdb",
  "daemon": {
    "addr": "localhost:8147",
    "socket": "xdb.sock"
  },
  "log_level": "info",
  "store": {
    "backend": "memory"
  }
}
```

### Store Backends

- **memory** (default): In-memory store, data is lost on restart
- **sqlite**: SQLite database, stored in `<dir>/data/`
- **redis**: Redis server, requires `addr` to be configured
- **fs**: Filesystem store, stored in `<dir>/data/` by default

Example with SQLite:

```json
{
  "store": {
    "backend": "sqlite",
    "sqlite": {
      "dir": "",
      "name": "xdb.db"
    }
  }
}
```

Example with Redis:

```json
{
  "store": {
    "backend": "redis",
    "redis": {
      "addr": "localhost:6379"
    }
  }
}
```

See `xdb.example.yaml` for a full reference of all configuration options.
