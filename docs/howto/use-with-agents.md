---
title: Use XDB from an agent
description: Discover the CLI at runtime, parse its errors, and pipe JSON between commands.
package: cmd/xdb/cli
read_when:
  - You give an agent access to the xdb CLI
  - You write a script that parses the output of xdb
---

# Use XDB from an agent

Each resource command has the same grammar:

```
xdb <resource> <action> <URI> [--filter CEL] [--fields MASK] [--json|--file|-] [-o FMT]
```

<figure class="frame">
  <svg id="fig-grammar" role="img" aria-label="The CLI grammar, xdb resource action URI with filter, fields, payload and output flags, each annotated."></svg>
  <figcaption>the grammar</figcaption>
</figure>

## Discover the CLI

`xdb context` prints the CLI guide as Markdown. `xdb describe` returns structured reference data.

```bash
xdb context                                  # the CLI guide
xdb describe --actions                       # the actions on each resource
xdb describe records.create                  # the parameters of one action
xdb describe --uri xdb://com.example/posts   # the live schema
xdb describe --filter                        # CEL operators and functions
xdb describe --errors                        # the error codes
xdb describe --value-types                   # the value types
```

The CLI also contains skills: Markdown recipes for `getting-started`, `bulk-data`, `query-and-filter`, and `schema-evolution`. The skills are in the binary, so they work offline.

```bash
xdb skills                  # list the skills
xdb skills get bulk-data    # print one skill
```

## Parse errors

Each error has the same shape in each output format:

```json
{
  "code": "NOT_FOUND",
  "message": "[xdb/api] records.get xdb://com.example/posts/nope: [xdb/core] not found",
  "resource": "records",
  "action": "get",
  "uri": "xdb://com.example/posts/nope",
  "hint": "try: xdb records list xdb://com.example/posts"
}
```

The exit code gives the class of the error:

| Exit code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Application error: `NOT_FOUND`, `ALREADY_EXISTS`, `CONFLICT`, `SCHEMA_VIOLATION`, or `NOT_IMPLEMENTED` |
| `2` | Connection error |
| `3` | Invalid argument |
| `4` | Internal error |

[Errors](../concepts/errors.md) explains each code.

## Pipe JSON

On a terminal, the default output is a table. In a pipe, the default output is JSON. `-o` selects `json`, `ndjson`, `table`, or `yaml`.

`-` reads the URI or the payload from stdin:

```bash
echo '{"title":"Hello"}' | xdb records create xdb://com.example/posts/p-1 -
xdb records get xdb://com.example/posts/p-1 | jq -r .title
```

<figure class="frame">
  <svg id="fig-pipe" role="img" aria-label="echo pipes JSON into xdb records create, which pipes JSON into jq or xdb batch."></svg>
  <figcaption>json in, json out</figcaption>
</figure>

`xdb batch` reads one operation from each line of NDJSON:

```bash
echo '{"op":"records.upsert","uri":"xdb://com.example/posts/p-2","data":{"title":"Hi"}}' | xdb batch -
```

A batch is atomic on the `sqlite` and `memory` backends. On the `fs` and `redis` backends, pass `--non-atomic`.

## Use the aliases

An alias selects the resource from the depth of the URI. Aliases make shell commands shorter. In a script, the full form is easier to read.

| Alias | Runs |
| --- | --- |
| `xdb get <uri>` | `get` on the namespace, schema, or record |
| `xdb ls [uri]` | `list`. Without a URI, it lists the namespaces. |
| `xdb put <uri>` | `records upsert` |
| `xdb rm <uri> --force` | `delete` on the record or schema |
| `xdb make-schema <uri>` | `schemas create` |

## Find the stored data

To find all the data, start at the namespaces:

```bash
xdb namespaces list
xdb namespaces get xdb://com.example
xdb records list xdb://com.example
```

`namespaces get` returns the schemas in the namespace:

```json
{ "namespace": "com.example", "schemas": ["xdb://com.example/posts"], "total_schemas": 1 }
```

## Check the daemon

The CLI sends each command to the daemon. `xdb daemon status` exits with code 2 when the daemon is stopped, so a script can stop before the first command:

```bash
xdb daemon status --quiet && xdb records list xdb://com.example/posts
```
