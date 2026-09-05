# XDB

XDB is an agent-first data layer. Model once, store anywhere. It has simple URIs, structured tuples, and a pipe-friendly CLI that agents and humans get right on the first try.

## Why XDB?

Read about the motivation behind XDB in [Introducing XDB](https://raviatluri.in/articles/introducing-xdb).

## Core Concepts

> For in-depth documentation on each concept, see [docs/concepts](./docs/concepts/).

The XDB data model is a tree of **Namespaces**, **Schemas**, **Records**, and **Tuples**.

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
│       Path | Attr | Value       │
└─────────────────────────────────┘
```

### Tuple

A **Tuple** is the fundamental building block in XDB. It has three parts:

- Path: the namespace, the schema, and the ID of the record. Together they identify the record.
- Attr: a string that identifies the attribute. It supports dot-separated nesting.
- Value: the value of the attribute.

### Record

One or more **Tuples** with the same **path** (NS + Schema + ID) make up a **Record**. The ID alone does not group tuples. The full path does. A record _is_ its tuples. It adds no data of its own. When at least one tuple exists at a path, the record at that path exists.

This is the tuple-first framing of XDB: the tuple is the primitive, and every larger structure is built from tuples. A record is similar to an object, a struct, or a row in a database. It usually represents one entity of domain data.

### Namespace

A **Namespace** (NS) groups one or more **Schemas**. Namespaces usually organize schemas by domain, application, or tenant.

### Schema

A **Schema** defines the structure of records and groups them together. Declared fields always type-check. A schema has one of three modes, and the mode governs only undeclared fields. `strict` (the default) rejects undeclared fields. `flexible` accepts undeclared fields as-is. `dynamic` infers undeclared fields and adds them to the schema. Each schema has a unique name within its namespace.

### URI

XDB URIs are valid Uniform Resource Identifiers (URI) according to [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986). A URI identifies one resource in XDB.

The general format of a URI is:

```
    [SCHEME]://[DOMAIN] [ "/" PATH] [ "?" QUERY] [ "#" FRAGMENT]
```

XDB URIs have this format:

```
    xdb://NS [ "/" SCHEMA ] [ "/" ID ] [ "#" ATTRIBUTE ]
```

```
    xdb://com.example/posts/123-456-789#author.id
    └─┬──┘└────┬────┘└──┬─┘└─────┬─────┘└─────┬─────┘
   scheme     NS    SCHEMA      ID        ATTRIBUTE
              └───────────┬───────────┘
                       path
```

The components of the URI are:

- **NS**: the namespace.
- **SCHEMA**: the schema name.
- **ID**: the unique identifier of the record.
- **ATTRIBUTE**: the name of the attribute.
- **path**: NS, SCHEMA, and ID together. The path identifies one record (the URI without `xdb://`).

Valid examples:

```
Namespace:  xdb://com.example
Schema:     xdb://com.example/posts
Record:     xdb://com.example/posts/123-456-789
Attribute:  xdb://com.example/posts/123-456-789#author.id
```

## Supported Types

| Type       | SQLite    | Description             |
| ---------- | --------- | ----------------------- |
| `string`   | `TEXT`    | UTF-8 string            |
| `integer`  | `INTEGER` | 64-bit signed integer   |
| `unsigned` | `INTEGER` | 64-bit unsigned integer |
| `float`    | `REAL`    | 64-bit floating point   |
| `boolean`  | `INTEGER` | True or false           |
| `time`     | `INTEGER` | Date and time in UTC    |
| `json`     | `TEXT`    | Arbitrary JSON data     |
| `bytes`    | `BLOB`    | Binary data             |
| `array`    | `TEXT`    | Array of typed values   |

## Getting Started

### Installation

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
xdb init    # creates the config and starts the daemon
```

## Using XDB from the CLI

The `xdb` CLI is a small, regular language for reading and writing data. Every invocation has the same shape:

```
xdb <resource> <action> <URI> [--filter CEL] [--fields MASK] [--json|--file|-] [-o FMT]
```

One grammar, one predicate language, and one output protocol apply to every resource. Agents learn the grammar once. Humans get shorthand on top.

### Primitives

| Primitive  | Purpose                                                                             | Example                            |
| ---------- | ----------------------------------------------------------------------------------- | ---------------------------------- |
| `resource` | What you operate on: `records`, `schemas`, `namespaces`                             | `records`                          |
| `action`   | Closed set: `get`, `list`, `create`, `update`, `upsert`, `delete`                   | `upsert`                           |
| URI        | The noun: `xdb://NS/SCHEMA/ID#ATTR`. The depth selects the resource.                | `xdb://com.example/posts/p-1`      |
| `--filter` | CEL predicate ([AIP-160](https://google.aip.dev/160))                               | `--filter 'status == "published"'` |
| `--fields` | Projection (field mask)                                                             | `--fields _id,title`               |
| payload    | JSON from `--json`, `--file`, or stdin `-`                                          | `--json '{"title":"Hello"}'`       |
| `-o`       | Output format: `json`, `ndjson`, `table`, `yaml`                                    | `-o json`                          |

Namespaces support only `get` and `list`. `xdb watch <URI>` is a top-level command, not an action. It streams change events as NDJSON.

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
xdb records list xdb://com.example/posts --filter 'title.contains("Hello")' --fields _id,title --limit 10

# Delete data
xdb records delete xdb://com.example/posts/p-1 --force
```

`create` fails with `ALREADY_EXISTS` if the record exists. `update` is a patch: only the fields in the payload change. `upsert` replaces the whole record.

### `describe`: the built-in CLI reference

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

**Stdin `-` is the explicit pipe token.** Every command that takes a URI or a payload accepts `-` to read it from stdin:

```bash
echo '{"title":"t"}' | xdb records create xdb://com.example/posts/p-1 -
echo '{"op":"records.upsert","uri":"xdb://com.example/posts/p-2","data":{"title":"t2"}}' \
  | xdb batch -     # one {"op":"...","uri":"...","data":{...}} operation per line
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

**The output format is a table on a TTY and JSON on a pipe.** Override it with `-o`:

```bash
xdb records get xdb://com.example/posts/p-1            # table (TTY)
xdb records get xdb://com.example/posts/p-1 | jq .     # JSON (pipe)
xdb records get xdb://com.example/posts/p-1 -o yaml    # explicit
```

### Shorthand

The URI depth dispatches to the right resource. These commands are macros. Each one expands to the full form:

| Shorthand         | Expands to                                                    |
| ----------------- | ------------------------------------------------------------- |
| `xdb get <uri>`   | `records/schemas/namespaces get` (by URI depth)               |
| `xdb ls [uri]`    | `records/schemas/namespaces list`. Without a URI, it lists namespaces. |
| `xdb put <uri>`   | `records upsert` (record URI only)                            |
| `xdb rm  <uri>`   | `records/schemas delete` (requires `--force`)                 |
| `xdb make-schema` | `schemas create`                                              |

Use the full form in scripts and agent instructions. Use the shorthand at an interactive shell.

### Global flags

- `--config`, `-c`: path to the config file (default `~/.xdb/config.json`)
- `--output`, `-o`: output format (`json`, `ndjson`, `table`, `yaml`)
- `--verbose`, `-v`: enable verbose logging
- `--debug`: enable debug logging

### Daemon

The daemon runs a JSON-RPC server that handles all operations. The CLI is a thin client. `xdb init` starts the daemon. Most users never run the daemon commands directly.

```bash
xdb daemon start
xdb daemon status
xdb daemon stop
xdb daemon restart
```

> For the full grammar reference, with the action × resource matrix, the error codes, and the agent guidance, see [cmd/xdb/cli/CONTEXT.md](cmd/xdb/cli/CONTEXT.md).

## Config

XDB reads a JSON config file at `~/.xdb/config.json`. If the file does not exist, `xdb init` and `xdb daemon start` create it with defaults. If the file is missing, every other command uses the same defaults in memory.

The default config, as `xdb init` writes it:

```json
{
  "dir": "~/.xdb",
  "daemon": {
    "socket": "xdb.sock"
  },
  "log_level": "info",
  "store": {
    "backend": "sqlite"
  }
}
```

### Backends

- **sqlite** (default): a SQLite database file, at `<dir>/data/xdb.db` by default. Keys under `store.sqlite`: `path`, `journal`, `sync`, `cache_size`, `busy_timeout`.
- **memory**: an in-memory backend. When the daemon stops, the data is lost.
- **fs**: a filesystem backend, under `<dir>/data` by default. Key under `store.fs`: `dir`.
- **redis**: a Redis server. `store.redis.addr` is required. Optional keys under `store.redis`: `password`, `db`.

Example with SQLite:

```json
{
  "store": {
    "backend": "sqlite",
    "sqlite": {
      "path": "/var/lib/xdb/xdb.db",
      "journal": "wal"
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

Run `xdb describe --config` for the full reference of all config keys and their defaults.
