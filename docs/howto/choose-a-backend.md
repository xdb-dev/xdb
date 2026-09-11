---
title: Choose a backend
description: How each driver stores a tuple, how to select a backend in the config, and the contract for a new driver.
package: store, store/xdbmemory, store/xdbfs, store/xdbredis, store/xdbsqlite
read_when:
  - You select a backend for a new deployment
  - You write a driver for another database
---

# Choose a backend

A driver writes tuples to one backend. XDB has drivers for memory, the filesystem, Redis, and SQLite. The store applies schema validation and versioning above the driver, so each backend gets the same rules.

<figure class="frame">
  <svg id="fig-backends" role="img" aria-label="One tuple fans out into a memory map, a JSON file in a directory tree, a Redis hash, a SQLite table, and a dashed box for your own database."></svg>
  <figcaption>backend storage layouts</figcaption>
</figure>

## Storage layouts

Each driver stores the tuple `xdb://com.example/posts/p-1#title = "Hello"` in a different form:

| Backend | Storage |
| --- | --- |
| Memory | `m["com.example/posts/p-1"]["title"] = "Hello"` |
| Filesystem | `com.example/posts/p-1.json` contains `{ "title": "Hello" }` |
| Redis | `HSET xdb:com.example:posts:p-1 title "Hello"` |
| SQLite | `INSERT INTO "t:com.example/posts" (_id, title) VALUES ('p-1', 'Hello')` |

Each backend maps the XDB types to its own format. SQLite uses these column types:

| Type | Go | SQLite |
| --- | --- | --- |
| `string` | `string` | `TEXT` |
| `integer` | `int64` | `INTEGER` |
| `unsigned` | `uint64` | `INTEGER` |
| `float` | `float64` | `REAL` |
| `boolean` | `bool` | `INTEGER` |
| `time` | `time.Time` | `INTEGER` |
| `json` | `json.RawMessage` | `TEXT` |
| `bytes` | `[]byte` | `BLOB` |
| `array` | `[]*Value` | `TEXT` |

## Select a backend

Set `store.backend` in `~/.xdb/config.json` to `sqlite`, `memory`, `fs`, or `redis`. Put the options for that backend in the same object:

```json
{
  "store": {
    "backend": "sqlite",
    "sqlite": { "path": "/var/lib/xdb/xdb.db" }
  }
}
```

Then restart the daemon:

```bash
xdb daemon restart
```

CAUTION: A change of backend does not copy the data from the old backend. To keep the data, export it before the change. See [Read and write records](read-and-write.md#move-many-records).

| Backend | Options |
| --- | --- |
| `sqlite` | `store.sqlite.path` (default `<datadir>/xdb.db`), `journal`, `sync`, `cache_size`, `busy_timeout` |
| `redis` | `store.redis.addr` (required), `password`, `db` |
| `fs` | `store.fs.dir` (default `<datadir>`) |
| `memory` | No options. The data stays in the memory of the daemon. |

[Config](../concepts/config.md) describes each option.

## Use a driver in Go

`store.New` puts the validation and versioning middleware around a driver:

```go
// Memory
st := store.New(xdbmemory.NewDriver())

// Filesystem
d, err := xdbfs.NewDriver("/path/to/data", xdbfs.Options{})
st := store.New(d)

// Redis
st := store.New(xdbredis.NewDriver(client))

// SQLite
d, err := xdbsqlite.NewDriver(db)
st := store.New(d)
```

## Write a driver

A driver implements `TupleReader`, `TupleWriter`, `SchemaReader`, and `SchemaWriter`. The store builds records from the tuples, converts each write to mutations, and applies validation and versioning.

<figure class="frame">
  <svg id="fig-driver" role="img" aria-label="Four required roles, TupleReader, TupleWriter, SchemaReader and SchemaWriter, stack up to make a Driver. Two optional capabilities, TxDriver and QueryDriver, sit beside it, detected by store.New, with the facade filling in when they are absent."></svg>
  <figcaption>driver interfaces</figcaption>
</figure>

Optional interfaces add capabilities. `store.New` finds them on the driver:

- `TxDriver` makes a batch atomic. Without it, the store applies the operations of a batch one at a time, and a CLI batch needs `--non-atomic`.
- `QueryDriver` evaluates filters in the database through `QueryTuples`. Without it, the store scans the tuples and filters them.

To test a new driver, register `storetest.NewDriverSuite`, `storetest.NewQuerySuite`, and the store suites. [Drivers](../concepts/drivers.md) gives the contract.
