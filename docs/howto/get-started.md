---
title: Get started
description: Install the CLI, create a schema, and write and read your first record.
read_when:
  - You install XDB for the first time
  - You want a working store before you read the concepts
---

# Get started

You need Go 1.26 or later.

## Install the CLI

```bash
go install github.com/xdb-dev/xdb/cmd/xdb@latest
```

The `xdb` binary contains the CLI and the daemon.

## Start the daemon

```bash
xdb init
```

`xdb init` writes the config file to `~/.xdb/config.json` and starts the daemon. The default backend is SQLite. To use another backend, change `store.backend` in the config file. Then run `xdb daemon restart`. [Config](../concepts/config.md) lists the backends.

## Create a schema

A schema gives the fields of a record and their types.

```bash
xdb schemas create xdb://com.example/posts \
  --json '{"fields":{"title":{"type":"string"},"views":{"type":"integer"}}}'
```

The URI `xdb://com.example/posts` names the namespace `com.example` and the schema `posts`. XDB adds the `_version` and `_updated` system fields to each schema. A new schema is in strict mode. [Schemas](../concepts/schemas.md) explains the other modes.

## Write a record

```bash
xdb records create xdb://com.example/posts/p-1 \
  --json '{"title":"Hello","views":42}'
```

```json
{
  "_id": "p-1",
  "_ns": "com.example",
  "_schema": "posts",
  "_updated": "2026-09-11T12:04:06Z",
  "_version": 1,
  "title": "Hello",
  "views": 42
}
```

The store validates each value against the schema before it writes the record. Each field of the record is one tuple, for example `xdb://com.example/posts/p-1#title = "Hello"`.

## Read the record

```bash
xdb records get xdb://com.example/posts/p-1
```

To read one attribute, add it to the URI after `#`:

```bash
xdb get 'xdb://com.example/posts/p-1#title'
```

`xdb get` is an alias. It selects the resource from the depth of the URI.

To find records, give a CEL filter and a field mask:

```bash
xdb records list xdb://com.example/posts --filter 'views > 10' --fields _id,title
```

## Update with a version check

Each write increments `_version`. If an update includes `_version`, the store writes it only when the stored version is the same.

```bash
xdb records update xdb://com.example/posts/p-1 --json '{"_version":1,"views":43}'
```

If another client wrote the record first, the update fails with `CONFLICT`:

```json
{
  "code": "CONFLICT",
  "resource": "records",
  "action": "update",
  "uri": "xdb://com.example/posts/p-1",
  "hint": "re-read the record and retry with the current _version",
  "details": { "expected": "1", "got": "2" }
}
```

If you get `CONFLICT`, read the record again. Then send the update with the current `_version`. [Versioning](../concepts/versioning.md) explains the contract.

## Next steps

- Run `xdb context` to print the CLI guide for agents.
- Run `xdb describe --actions` to list the actions on each resource.
- Read [Tuples](../concepts/tuples.md) and [Records](../concepts/records.md) for the data model.
- Read [Stores](../concepts/stores.md) to embed a store in a Go service.
