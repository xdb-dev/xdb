# XDB CLI Context

XDB stores data as tuples: a path, an attribute, and a typed value. The storage backend (memory, files, Redis, or SQLite) is a config choice. Every resource has a URI: `xdb://NS/SCHEMA/ID#ATTR`.

`xdb context` prints this guide.

## Grammar

```
xdb <resource> <action> <URI> [--filter CEL] [--fields MASK] [--json|--file|-] [-o FMT]
```

- **resource**: `records` / `schemas` / `namespaces`
- **action**: `get | list | create | update | upsert | delete`. Namespaces support only `get` and `list`.
- **URI**: the noun. The depth selects the resource (`ns` / `ns/schema` / `ns/schema/id`).
- **`-o`**: `json` / `ndjson` / `table` / `yaml`. Default: `table` on a TTY, `json` on a pipe.
- **`xdb watch <URI>`** is a top-level command, not an action. It streams change events as NDJSON.

The RPC layer names each action `<resource>.<action>`, for example `records.create`. Help, `describe`, and this guide call them **actions**. The dotted form is the stable identifier.

**Discover the surface with these commands:**

```bash
xdb describe --actions                 # what actions exist on what resources
xdb describe records.create            # parameters for one action
xdb describe --uri xdb://ns/schema     # live data schema
```

On `list` calls, pass `--fields` and `--limit` so that responses stay bounded. The default limit is 20.

## Minimal examples

```bash
# Read
xdb records get  xdb://ns/s/id --fields title,author
xdb records list xdb://ns/s --filter 'status=="published"' --fields _id,title --limit 10

# Write: create fails with ALREADY_EXISTS if the record exists · update = patch · upsert = full replace
xdb records create xdb://ns/s/id --json '{"title":"Hello"}'
xdb records update xdb://ns/s/id --json '{"title":"Updated"}'

# Delete requires --force
xdb records delete xdb://ns/s/id --force

# Schema
xdb schemas create xdb://ns/s --json '{"fields":{"title":{"type":"string"}}}'
```

Every record carries `_id`, `_version` (a counter that starts at 1), and `_updated`. If a write includes `_version`, and the stored version differs, the write fails with CONFLICT.

## When you need more

The CLI can show everything else:

| Need                           | Command                            |
| ------------------------------ | ---------------------------------- |
| Action × resource matrix       | `xdb describe --actions`           |
| Parameters for an action       | `xdb describe <resource>.<action>` |
| Type definition                | `xdb describe <Type>`              |
| Live schema for a URI          | `xdb describe --uri <uri>`         |
| CEL filter operators/functions | `xdb describe --filter`            |
| Error code catalog             | `xdb describe --errors`            |
| Supported value types          | `xdb describe --value-types`       |

## Composition

The CLI composes through stdin, stdout, and one error shape.

**Stdin `-` is the pipe token.** Every command that takes a URI or a payload accepts `-` to read it from stdin:

```bash
echo '{"title":"t"}' | xdb records create xdb://ns/s/id -
echo '{"op":"records.upsert","uri":"xdb://ns/s/id","data":{"title":"t"}}' | xdb batch -
```

Each NDJSON batch line is one operation: `{"op":"records.create","uri":"...","data":{...}}`. Allowed ops: `records.create`, `records.update`, `records.upsert`, `records.delete`, `schemas.create`, `schemas.update`, `schemas.delete`. A batch is atomic on the `sqlite` and `memory` backends. On the `fs` and `redis` backends, pass `--non-atomic`. For details, run `xdb skills get bulk-data`.

**Every error, in every format, has the same shape:**

```json
{
  "code": "NOT_FOUND",
  "message": "...",
  "resource": "records",
  "action": "get",
  "uri": "xdb://...",
  "hint": "..."
}
```

## Common flags

`-c, --config` is a root flag. The other flags are declared per action. To see the flags of one action, run `xdb <resource> <action> --help`.

| Flag                  | Purpose                                    |
| --------------------- | ------------------------------------------ |
| `--uri` / positional  | Target resource                            |
| `--json '<JSON>'`     | Inline payload                             |
| `-f, --file <PATH>`   | Payload from file                          |
| `-`                   | Read URI or payload from stdin             |
| `--filter <CEL>`      | Filter expression (`records list` only)    |
| `--fields <MASK>`     | Field mask (`_id` always included)         |
| `--limit`, `--offset` | Pagination                                 |
| `--force`             | Required for deletes                       |
| `--cascade`           | Delete schema and all its records          |
| `--dry-run`           | Validate without writing                   |
| `-o, --output`        | Output format                              |
| `--quiet`             | Suppress output, exit code only            |
| `-c, --config`        | Config path (default `~/.xdb/config.json`) |

## Exit codes

`0` ok · `1` app error (`NOT_FOUND`, `ALREADY_EXISTS`, `CONFLICT`, `SCHEMA_VIOLATION`, `NOT_IMPLEMENTED`) · `2` connection · `3` invalid argument · `4` internal

## System

```bash
xdb init              # config + daemon
xdb daemon status     # daemon state (exit 2 when stopped)
xdb skills            # list agent skills
xdb skills get <name> # print one skill
```

## Output shapes

- `records list` and `schemas list` with `-o json` or `-o yaml` return a page
  envelope `{"items": [...], "total": N, "next_offset": M}`. On the last page,
  `next_offset` is omitted. `-o ndjson` streams bare items. To fetch every
  page, pass `--page-all`.
- `namespaces get` returns `{"data": "<ns>", "schemas": [...], "total_schemas": N}`.
  To discover all state, walk `namespaces list` -> `namespaces get` ->
  `records list xdb://ns`.
- `daemon status` exits 2 when the daemon is stopped and 0 when it runs.
  Scripts can gate with `xdb daemon status --quiet && ...`.
