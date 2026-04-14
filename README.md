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

| Type       | PostgreSQL         | SQLite    | Description             |
| ---------- | ------------------ | --------- | ----------------------- |
| `string`   | `TEXT`             | `TEXT`    | UTF-8 string            |
| `integer`  | `BIGINT`           | `INTEGER` | 64-bit signed integer   |
| `unsigned` | `BIGINT`           | `INTEGER` | 64-bit unsigned integer |
| `float`    | `DOUBLE PRECISION` | `REAL`    | 64-bit floating point   |
| `boolean`  | `BOOLEAN`          | `INTEGER` | True or false           |
| `time`     | `TIMESTAMPTZ`      | `INTEGER` | Date and time in UTC    |
| `json`     | `JSONB`            | `TEXT`    | Arbitrary JSON data     |
| `bytes`    | `BYTEA`            | `BLOB`    | Binary data             |
| `array`    | `[]T`              | `TEXT`    | Array of typed values   |

## Getting Started

### Installation

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
xdb --version
xdb init    # creates config, data dir, starts the daemon
```

## Using xdb from the CLI

The xdb CLI is a small, regular language for reading and writing data. Every invocation has the same shape:

```
xdb <resource> <action> <URI> [--filter CEL] [--fields MASK] [--json|--file|-] [-o FMT]
```

One grammar, one predicate language, one output protocol — applied uniformly to every resource. Agents learn it once; humans get shorthand on top.

### Primitives

| Primitive  | Purpose                                                                    | Example                            |
| ---------- | -------------------------------------------------------------------------- | ---------------------------------- |
| `resource` | What you're operating on: `records`, `schemas`, `namespaces`               | `records`                          |
| `action`   | Closed set: `get`, `list`, `create`, `update`, `upsert`, `delete`, `watch` | `upsert`                           |
| URI        | The noun — `xdb://NS/SCHEMA/ID#ATTR`. Depth determines the resource.       | `xdb://com.example/posts/p-1`      |
| `--filter` | CEL predicate ([AIP-160](https://google.aip.dev/160))                      | `--filter 'status == "published"'` |
| `--fields` | Projection (field mask)                                                    | `--fields id,title`                |
| payload    | JSON from `--json`, `--file`, or stdin `-`                                 | `--json '{"title":"Hello"}'`       |
| `-o`       | Output format: `json`, `ndjson`, `table`, `yaml`                           | `-o json`                          |

### Canonical example

One URI, walked through the full action set:

```bash
# Define the schema
xdb schemas create xdb://com.example/posts --json '{"fields":{"title":{"type":"string"}}}'

# Write data
xdb records create xdb://com.example/posts/p-1 --json '{"title":"Hello"}'
xdb records update xdb://com.example/posts/p-1 --json '{"title":"Updated"}'
xdb records upsert xdb://com.example/posts/p-1 --json '{"title":"Full replace"}'

# Read data
xdb records get  xdb://com.example/posts/p-1 --fields title
xdb records list xdb://com.example/posts --filter 'title.contains("Hello")' --fields id,title --limit 10

# Delete data
xdb records delete xdb://com.example/posts/p-1 --force
```

### `describe` — the CLI reference, built in

`describe` introspects every part of the CLI:

```bash
xdb describe records.create    # action signature
xdb describe Record            # type definition
xdb describe --uri xdb://ns/schema  # live data schema
xdb describe --actions         # action × resource matrix
xdb describe --filter          # CEL operators and functions
xdb describe --errors          # error code catalog
xdb describe --value-types     # supported value types
```

### Composition

Commands compose through stdin, stdout, and one error shape.

**Stdin `-` is the explicit pipe token.** Any command that takes a URI or payload accepts `-` to read it from stdin:

```bash
echo '{"title":"t"}' | xdb records create xdb://com.example/posts/p-1 -
xdb records list xdb://com.example/posts -o ndjson \
  | xdb batch -     # each line is {"resource":"...","action":"...","uri":"...","payload":{...}}
```

**Errors are structured.** Every error, in every format, has the same shape:

```json
{
  "code": "NOT_FOUND",
  "message": "record not found",
  "resource": "records",
  "action": "get",
  "uri": "xdb://...",
  "hint": "try xdb records list xdb://..."
}
```

**Output format is automatic on a TTY, JSON on a pipe.** Override with `-o`:

```bash
xdb records get xdb://com.example/posts/p-1            # table (TTY)
xdb records get xdb://com.example/posts/p-1 | jq .     # JSON (pipe)
xdb records get xdb://com.example/posts/p-1 -o yaml    # explicit
```

### Shorthand

URI depth dispatches to the right resource. These are macros — each expands to the full form:

| Shorthand         | Expands to                                      |
| ----------------- | ----------------------------------------------- |
| `xdb get <uri>`   | `records/schemas/namespaces get` (by URI depth) |
| `xdb ls  <uri>`   | `records/schemas/namespaces list`               |
| `xdb put <uri>`   | `records/schemas upsert`                        |
| `xdb rm  <uri>`   | `records/schemas delete` (requires `--force`)   |
| `xdb make-schema` | `schemas create`                                |

Prefer the full form in scripts and agent prompts; reach for shorthand at a prompt.

### Global flags

- `--config`, `-c`: Path to config file (default `~/.xdb/config.json`)
- `--output`, `-o`: Output format (`json`, `ndjson`, `table`, `yaml`)
- `--verbose`, `-v`: INFO-level logging
- `--debug`: Debug logging with source locations

### Daemon

The daemon runs a JSON-RPC server that handles all operations; the CLI is a thin client. `xdb init` starts it; most users never invoke it directly.

```bash
xdb daemon start
xdb daemon status
xdb daemon stop
xdb daemon restart
```

> For the full grammar reference — including the action × resource matrix, error codes, and agent-targeted guidance — see [cmd/xdb/cli/CONTEXT.md](cmd/xdb/cli/CONTEXT.md).

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
