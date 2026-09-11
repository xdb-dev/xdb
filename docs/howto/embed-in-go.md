---
title: Embed XDB in Go
description: Build a store in your own service and serve it over JSON-RPC.
package: store, api, rpc
read_when:
  - You use XDB as a library instead of through the daemon
  - You want remote clients to call your store over JSON-RPC
---

# Embed XDB in Go

The `xdb` daemon is a Go program that builds a store and serves it over JSON-RPC. Your service can do the same.

## Build a store

`store.New` puts schema validation and versioning around a driver. `store.WithSchemaCache` keeps the schema definitions in memory.

```go
d, err := xdbsqlite.NewDriver(db)
if err != nil {
	return err
}
st := store.New(d, store.WithSchemaCache())
```

[Choose a backend](choose-a-backend.md) shows the other drivers.

## Read and write

Write a record, or write one tuple:

```go
rec := core.NewRecord("com.example", "posts", "p-1").
	Set("title", "Hello").
	Set("views", 42)
if err := st.CreateRecord(ctx, rec); err != nil {
	return err
}

err := st.PutTuples(ctx,
	core.NewTuple("com.example/posts/p-1", "views", 43),
)
```

List the records that match a CEL filter:

```go
page, err := st.ListRecords(ctx, &store.Query{
	URI:    core.MustParseURI("xdb://com.example/posts"),
	Filter: `views > 10`,
	Limit:  10,
})
```

`page.Items` holds the records, and `page.Total` gives the count of matches. To get the next page, set `Offset` to `page.NextOffset`.

Encode a record as JSON with only some of its fields. The output always includes `_id`:

```go
data, err := xdbjson.Unmarshal(rec, xdbjson.WithFields("title", "views"))
// {"_id":"p-1","title":"Hello","views":42}
```

`store.TX` runs a function in one transaction. If the function returns an error, `Run` rolls back each change:

```go
tx, ok := st.(store.TX)
if ok {
	err = tx.Run(ctx, func(tx store.Store) error {
		return tx.PutTuples(ctx,
			core.NewTuple("com.example/posts/p-1", "views", 44),
		)
	})
}
```

[Drivers](../concepts/drivers.md) lists the backends that support transactions.

## Serve JSON-RPC

Register a handler for each method on a router. The router is an `http.Handler`:

```go
records := api.NewRecordService(st)
schemas := api.NewSchemaService(st)

r := rpc.NewRouter()
rpc.RegisterHandler(r, "records.create", records.Create)
rpc.RegisterHandler(r, "records.get", records.Get)
rpc.RegisterHandler(r, "records.list", records.List)
rpc.RegisterHandler(r, "schemas.create", schemas.Create)

err := http.ListenAndServe(":8147", r)
```

A client sends a JSON-RPC 2.0 request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "records.list",
  "params": {
    "uri": "xdb://com.example/posts",
    "filter": "views > 10",
    "fields": ["title", "views"],
    "limit": 10
  }
}
```

The daemon also serves the `introspect.methods`, `introspect.method`, `introspect.types`, and `introspect.type` methods. They return the catalog of methods and types. A JSON-RPC error code tells a conflict from a missing resource, so a client does not parse the error message. [Errors](../concepts/errors.md) lists the codes.
