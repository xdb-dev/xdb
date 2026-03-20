# XDB CLI Context

Agent-first data layer. Model once, store anywhere. URI-addressed: `xdb://NAMESPACE/SCHEMA/ID#ATTR`.

**Before you start:** `xdb describe --uri xdb://NS/SCHEMA` to discover schema. `xdb describe <resource>.<method>` to discover method params. `xdb describe --value-types` for supported field types. Always `--fields` and `--limit` to protect context window.

## Syntax

```
xdb <resource> <method> [flags]        # structured
xdb <alias> <uri> [flags]              # human shorthand
xdb describe <resource.method | Type>  # introspection
```

## Flags

| Flag                  | Purpose                                            |
| --------------------- | -------------------------------------------------- |
| `-c, --config <PATH>` | Config file (default `~/.xdb/config.json`)         |
| `-o, --output <FMT>`  | `json` (pipes) / `table` (TTY) / `yaml` / `ndjson` |
| `--uri <URI>`         | Target resource                                    |
| `--json '<JSON>'`     | Inline payload                                     |
| `-f, --file <PATH>`   | Payload from file (stdin if omitted)               |
| `--filter <CEL>`      | CEL filter expression (AIP-160)                    |
| `--fields <MASK>`     | Field mask (`_id` always included)                 |
| `--limit`, `--offset` | Pagination                                         |
| `--dry-run`           | Validate without writing                           |
| `--force`             | Required for deletes                               |
| `--cascade`           | Delete schema and all its records                  |
| `--quiet`             | Suppress output, exit code only                    |

## Filter Syntax (CEL / AIP-160)

```bash
--filter 'status == "published"'
--filter 'status == "active" && age >= 18'
--filter 'title.contains("hello") || title.startsWith("Hi")'
--filter 'status in ["active", "pending"]'
--filter 'size(name) > 3'
--filter '!(archived == true)'
```

Operators: `==`, `!=`, `<`, `>`, `<=`, `>=`, `&&`, `||`, `!`, `in`
Functions: `.contains()`, `.startsWith()`, `.endsWith()`, `size()`

## Value Types

`string`, `integer`, `unsigned`, `float`, `boolean`, `time`, `bytes`, `json`, `array`

Use these in schema field definitions: `{"fields":{"age":{"type":"integer"}}}`. Run `xdb describe --value-types` for the full list.

## Examples

```bash
# Read
xdb records get --uri xdb://ns/schema/id --fields title,author
xdb records list --uri xdb://ns/schema --filter 'status == "published"' --fields id,title --limit 10
xdb records list --uri xdb://ns/schema --filter 'age > 30 && active == true' --fields id,name --limit 10
xdb schemas list --uri xdb://ns

# Write (create=idempotent, update=patch merge, upsert=full replace)
xdb records create --uri xdb://ns/schema/id --json '{"title":"Hello"}'
xdb records update --uri xdb://ns/schema/id --json '{"title":"Updated"}'
xdb records upsert --uri xdb://ns/schema/id --json '{"title":"Complete","author":"Bob"}'

# Bulk
xdb export --uri xdb://ns/schema --fields id,title
cat data.ndjson | xdb import --uri xdb://ns/schema
xdb import --uri xdb://ns/schema --file data.ndjson --create-only

# Schema
xdb schemas create --uri xdb://ns/schema --json '{"fields":{"title":{"type":"string"}}}'
xdb schemas delete --uri xdb://ns/schema --force --cascade  # delete schema and all records

# Introspect
xdb describe records.create       # method params
xdb describe Record               # type definition
xdb describe --methods            # all methods
xdb describe --types              # all types
xdb describe --value-types        # supported value types
xdb describe --uri xdb://ns/schema  # data schema
```

## Aliases

URI depth determines resource. Positional URI (no `--uri`):

```bash
xdb get  xdb://ns/schema/id       # records get
xdb get  xdb://ns/schema          # schemas get
xdb get  xdb://ns                 # namespaces get
xdb ls   xdb://ns/schema          # records list
xdb ls   xdb://ns                 # schemas list
xdb ls                            # namespaces list
xdb put  xdb://ns/schema/id ...   # records upsert
xdb rm   xdb://ns/schema/id --force  # records delete
xdb rm   xdb://ns/schema --force  # schemas delete
xdb make-schema xdb://ns/schema   # schemas create
```

## System

```bash
xdb init                          # config + data dir + start daemon
xdb skills list                   # available skills
xdb skills get <name>             # skill document
```

## Exit Codes

- `0` success
- `1` app error (NOT_FOUND, ALREADY_EXISTS, SCHEMA_VIOLATION)
- `2` connection error
- `3` input validation error
- `4` internal error
