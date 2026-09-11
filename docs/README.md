---
title: Overview
description: The tuple model, the package map, and a first record in Go.
read_when:
  - You open the XDB docs for the first time
  - You decide which package to start with
---

# Overview

XDB stores data as tuples. A tuple is a path, an attribute, and a typed value:

```
xdb://com.example/posts/p-1#title = "Hello"
```

A record is the set of tuples at one path. You read and write records from Go, over JSON-RPC, or with the `xdb` CLI. The same calls work on each backend: memory, files, Redis, or SQLite.

## Packages

| Package | Contents |
| --- | --- |
| `core` | Tuples, records, URIs, and typed values |
| `schema` | Schema definitions and validation modes |
| `store` | The store facade. It validates each write against its schema and sets the record version. |
| `store/xdbmemory`, `store/xdbfs`, `store/xdbredis`, `store/xdbsqlite` | The drivers. Each driver writes tuples to one backend. |
| `encoding/xdbjson`, `encoding/xdbproto`, `encoding/xdbstruct` | Conversion between tuples and JSON, protobuf messages, or Go structs |
| `filter` | CEL filters for record lists |
| `api`, `rpc` | The JSON-RPC server. It has one method for each action, for example `records.create`. |
| `cmd/xdb` | The CLI and the daemon. The CLI sends each command to the daemon over JSON-RPC. |

## Your first record

This program keeps a record in memory and reads one attribute back.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func main() {
	ctx := context.Background()
	st := store.New(xdbmemory.NewDriver())

	post := core.NewRecord("com.example", "posts", "p-1").
		Set("title", "Hello").
		Set("views", 42)

	if err := st.CreateRecord(ctx, post); err != nil {
		log.Fatal(err)
	}

	got, err := st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/p-1"))
	if err != nil {
		log.Fatal(err)
	}

	title, err := got.Get("title").AsStr()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(title)
}
```

To use the CLI instead, read [Get started](howto/get-started.md).

## Find a page

- [Get started](howto/get-started.md): install the CLI and write your first record.
- [Concepts](concepts/README.md): tuples, records, schemas, stores, and the rest of the design.
- [CLI reference](../cmd/xdb/cli/CONTEXT.md): the commands, flags, and payloads. `xdb context` prints the same guide.
- [Go API](https://pkg.go.dev/github.com/xdb-dev/xdb): the exact signatures and fields.
