---
title: Read and write records
description: Get, list, create, update, and delete records, with version checks, dry runs, bulk data, and change streams.
package: api, cmd/xdb/cli
read_when:
  - You write a script or an agent that changes records
  - A write fails with CONFLICT or SCHEMA_VIOLATION
  - You move many records at once
---

# Read and write records

Go, JSON-RPC, and the CLI send reads and writes to the same store. The store applies the same validation and versioning to each client. The examples use the CLI and the `xdb://com.example/posts` schema from [Get started](get-started.md).

<figure class="frame">
  <svg id="fig-doors" role="img" aria-label="Go, JSON-RPC and CLI arrows converge on one store, which talks to a driver."></svg>
  <figcaption>clients share a store</figcaption>
</figure>

## Read

Read one record by its URI. `--fields` selects the fields in the output.

```bash
xdb records get xdb://com.example/posts/p-1 --fields title,views
```

List the records that match a CEL filter. `--limit` sets the page size. The default limit is 20.

```bash
xdb records list xdb://com.example/posts --filter 'views > 10' --fields _id,title --limit 10
```

With `-o json` or `-o yaml`, a list returns a page:

```json
{ "items": [{ "_id": "p-1", "title": "Hello" }], "total": 1 }
```

If there are more records, the page also has `next_offset`. To get the next page, give that value to `--offset`. To get all the pages, pass `--page-all`. `-o ndjson` writes one record on each line.

On SQLite, the store compiles the filter to SQL. On the other backends, the store scans the records and filters them. [Filters](../concepts/filters.md) lists the operators.

## Write

| Command | Result |
| --- | --- |
| `records create` | Creates the record. If the record exists with the same data, the command succeeds. If the data is different, it fails with `CONFLICT`. |
| `records update` | Changes only the fields in the payload |
| `records upsert` | Replaces the record |
| `records delete --force` | Deletes the record |

```bash
xdb records update xdb://com.example/posts/p-1 --json '{"views":43}'
xdb records upsert xdb://com.example/posts/p-1 --json '{"title":"Hi"}'
xdb records delete xdb://com.example/posts/p-1 --force
```

To delete a schema and its records, add `--cascade`:

```bash
xdb schemas delete xdb://com.example/posts --force --cascade
```

CAUTION: `--cascade` deletes each record in the schema.

## Check the version before a write

Each record has a `_version` that starts at 1. Each write increments it and returns the new version. If an update includes a `_version` that is not 0, the store writes only when the stored version is the same:

```bash
xdb records update xdb://com.example/posts/p-1 --json '{"_version":1,"views":43}'
```

For a delete, give the version with `--if-version`:

```bash
xdb records delete xdb://com.example/posts/p-1 --force --if-version 7
```

```json
{
  "code": "CONFLICT",
  "message": "records.delete xdb://com.example/posts/p-1: record is at version 1, not 7: [xdb/core] revision conflict",
  "resource": "records",
  "action": "delete",
  "uri": "xdb://com.example/posts/p-1"
}
```

If you get `CONFLICT`, read the record again. Then send the write with the current version.

<figure class="frame">
  <svg id="fig-cas" role="img" aria-label="Two agents read version 3. Agent A writes with version 3 and succeeds, moving to 4. Agent B writes with version 3 and gets a conflict, re-reads version 4, writes again and succeeds."></svg>
  <figcaption>version checks on transactional backends</figcaption>
</figure>

Memory and SQLite do the check and the write in one transaction. On the filesystem and Redis backends, another write can occur between the check and the write. [Versioning](../concepts/versioning.md) gives the full contract.

## Validate a write without writing

`--dry-run` validates the payload against the schema and writes nothing:

```bash
xdb records update xdb://com.example/posts/p-1 --json '{"views":"lots"}' --dry-run
```

```json
{
  "code": "SCHEMA_VIOLATION",
  "message": "[xdb/core] schema violation: field \"views\": cannot decode as INTEGER [expected=INTEGER, got=STRING, reason=decode_failed, field=views]",
  "resource": "records",
  "action": "update",
  "uri": "xdb://com.example/posts/p-1",
  "hint": "run xdb describe --uri <schema-uri> to inspect the schema",
  "details": { "expected": "INTEGER", "field": "views", "got": "STRING", "reason": "decode_failed" }
}
```

If the payload is valid, the result shows the record that the write would store:

```json
{
  "dry_run": true,
  "valid": true,
  "would": "update",
  "record": {
    "_id": "p-1",
    "_ns": "com.example",
    "_schema": "posts",
    "_updated": "2026-09-11T12:39:22Z",
    "_version": 1,
    "title": "Hello",
    "views": 43
  }
}
```

## Move many records

Export the records of a schema as NDJSON. Then import the file:

```bash
xdb export --uri xdb://com.example/posts > posts.ndjson
xdb import --uri xdb://com.example/posts -f posts.ndjson
```

```json
{ "imported": 1, "skipped": 0, "failed": 0 }
```

Import upserts each record. With `--create-only`, import skips the records that exist with different data.

`xdb batch` applies a stream of operations. Each line of NDJSON is one operation:

```bash
xdb batch - < ops.ndjson
```

```json
{"op":"records.upsert","uri":"xdb://com.example/posts/p-2","data":{"title":"Hi"}}
```

The operations are `records.create`, `records.update`, `records.upsert`, `records.delete`, `schemas.create`, `schemas.update`, and `schemas.delete`. A batch is atomic on the `sqlite` and `memory` backends. On the `fs` and `redis` backends, pass `--non-atomic`.

## Watch changes

`xdb watch` streams the changes to a namespace, a schema, or a record as NDJSON:

```bash
xdb watch xdb://com.example/posts
```

```json
{"ts":"...","type":"record.create","uri":"xdb://com.example/posts/p-2","version":1}
{"ts":"...","type":"record.update","uri":"xdb://com.example/posts/p-1","version":4}
```

Each record event includes the version. A delete event has the last version before the delete. If the versions of one record have a gap, the stream missed a write. Delivery is at most once, and the stream does not replay old events.
