# XDB CLI Context

Agent-first data layer. Model once, store anywhere. URI-addressed: `xdb://NS/SCHEMA/ID#ATTR`.

## Grammar

```
xdb <resource> <action> <URI> [--filter CEL] [--fields MASK] [--json|--file|-] [-o FMT]
```

- **resource** — `records` / `schemas` / `namespaces`
- **action** — `get | list | create | update | upsert | delete | watch`
- **URI** — the noun; depth determines resource (`ns` / `ns/schema` / `ns/schema/id`)
- **`-o`** — `json` / `ndjson` / `table` / `yaml` (auto: table on TTY, json on pipe)

**First-day moves — always do these:**

```bash
xdb describe --actions                 # what actions exist on what resources
xdb describe records.create            # parameters for one action
xdb describe --uri xdb://ns/schema     # live data schema
```

Always pass `--fields` and `--limit` to keep responses bounded.

## Minimal examples

```bash
# Read
xdb records get  xdb://ns/s/id --fields title,author
xdb records list xdb://ns/s --filter 'status=="published"' --fields id,title --limit 10

# Write — create = idempotent insert · update = patch · upsert = full replace
xdb records create xdb://ns/s/id --json '{"title":"Hello"}'
xdb records update xdb://ns/s/id --json '{"title":"Updated"}'

# Delete requires --force
xdb records delete xdb://ns/s/id --force

# Schema
xdb schemas create xdb://ns/s --json '{"fields":{"title":{"type":"string"}}}'
```

## When you need more

Everything else is retrievable from the CLI itself:

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

The CLI composes via stdin, stdout, and one error shape.

**Stdin `-` is the pipe token** — any command taking a URI or payload accepts `-` for stdin:

```bash
echo '{"title":"t"}' | xdb records create xdb://ns/s/id -
xdb records list xdb://ns/s -o ndjson | xdb batch -
```

Each NDJSON batch line is the serialized AST of one invocation: `{"resource","action","uri","payload"}`.

> **Note:** `xdb batch` accepts the input shape today but the server's `batch.execute` is not yet implemented — invocations currently return `INTERNAL` ("batch.execute not implemented"). Use single-action calls until the server side lands.

**Every error, every format, same shape:**

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

## Common flags (inherited)

| Flag                  | Purpose                                    |
| --------------------- | ------------------------------------------ |
| `--uri` / positional  | Target resource                            |
| `--json '<JSON>'`     | Inline payload                             |
| `-f, --file <PATH>`   | Payload from file                          |
| `-`                   | Read URI or payload from stdin             |
| `--filter <CEL>`      | Filter expression (list only)              |
| `--fields <MASK>`     | Field mask (`_id` always included)         |
| `--limit`, `--offset` | Pagination                                 |
| `--force`             | Required for deletes                       |
| `--cascade`           | Delete schema and all its records          |
| `--dry-run`           | Validate without writing                   |
| `-o, --output`        | Output format                              |
| `--quiet`             | Suppress output, exit code only            |
| `-c, --config`        | Config path (default `~/.xdb/config.json`) |

## Exit codes

`0` ok · `1` app error (`NOT_FOUND`, `ALREADY_EXISTS`, `SCHEMA_VIOLATION`) · `2` connection · `3` invalid argument · `4` internal

## System

```bash
xdb init              # config + data dir + daemon
xdb daemon status     # check daemon
xdb skills list       # agent skills
```

---

Note: the RPC layer addresses actions as `<resource>.<action>` (e.g. `records.create`). User-facing docs, help, and `describe` call them **actions**; the dotted form is the stable identifier.
