---
title: Tuples
description: The smallest unit of data in XDB — an immutable path, attribute, and typed value.
package: core
---

# Tuples

A **Tuple** is the smallest unit of data in XDB. It is an addressable fact:
`xdb://ns/schema/id#attr = value`. Every piece of data in XDB is a tuple, and
every larger structure is built from tuples. A [Record](records.md) is the set
of tuples that share one path.

## Structure

A tuple has three components:

| Component | Type     | Description                                     |
| --------- | -------- | ----------------------------------------------- |
| **Path**  | `*URI`   | The record URI (NS + Schema + ID)               |
| **Attr**  | `string` | The attribute name. Dots separate nested names  |
| **Value** | `*Value` | The typed value                                 |

```
┌────────────────────────────────────────────────┐
│                     Tuple                      │
├──────────┬──────────┬──────────────────────────┤
│   Path   │   Attr   │          Value           │
│  (URI)   │ (string) │ (typed: str, int, ...)   │
└──────────┴──────────┴──────────────────────────┘
```

A tuple is **immutable** after creation. Its path, attribute, and value cannot change.

## Creating Tuples

```go
tuple := core.NewTuple("com.example/posts/post-123", "title", "Hello World")
```

`NewTuple` panics on invalid input. The path argument is a URI path without
the `xdb://` scheme. XDB infers the value type from the Go type. See
[Types](types.md) for the supported types.

Usually you create tuples through a [Record](records.md). The record owns the
shared path:

```go
record := core.NewRecord("com.example", "posts", "post-123").
    Set("title", "Hello World")

tuple := record.Get("title")
```

## Accessing Data

### Path Components

```go
tuple.Path()   // *URI   — record URI (without the attribute)
tuple.Attr()   // string — attribute name
tuple.URI()    // *URI   — full URI, including the attribute fragment
tuple.Value()  // *Value — typed value

tuple.Path().NS()     // string — namespace
tuple.Path().Schema() // string — schema name
tuple.Path().ID()     // string — record identifier
```

### Typed Value Accessors

A tuple has `As*` methods (`AsStr()`, `AsInt()`, `AsBool()`, and more) that
return the value with its type. Each method returns `(T, error)`. If the
attribute is missing, the tuple is nil, for example `record.Get("tpyo")`. The
`As*` methods on a nil tuple return the zero value and
[`ErrAttrNotFound`](../../core/errors.go). This makes a typo different from an
empty value. See [Types](types.md) for the full list of accessors.

```go
title, err := record.Get("title").AsStr()
```

## Dot-Separated Attributes

An attribute name can contain dots to represent nested data, for example `author.name` or `profile.address.city`. The JSON encoder unfolds these attributes into nested objects. See [Encoding](encoding.md) for details.

## Tuples in Stores

Tuples are not only a data-model detail. They are the unit that the storage
layer operates on. Every attribute is addressable through a
[Store](stores.md) with its `#attr` URI:

```go
st := store.New(xdbmemory.NewDriver())

// Patch tuples into records. The store creates the record if it is
// absent and leaves other attributes unchanged. Tuples can span records.
err := st.PutTuples(ctx,
    core.NewTuple("com.example/posts/post-123", "title", "Hello"),
    core.NewTuple("com.example/posts/post-123", "rating", 4.5),
)

// Point-read one tuple. If the record or the attribute is absent, the
// error is core.ErrNotFound.
tuple, err := st.GetTuple(ctx,
    core.MustParseURI("xdb://com.example/posts/post-123#title"))

// Batch point reads omit absent attributes and return no error for them.
tuples, err := st.GetTuples(ctx, uris...)

// Delete single tuples. This is idempotent. When the last tuple of a
// record is deleted, the record is deleted.
err = st.DeleteTuples(ctx,
    core.MustParseURI("xdb://com.example/posts/post-123#rating"))
```

A record starts to exist when its first tuples are written. It stops existing
when its last tuple is deleted. Records are views over tuple sets, not
containers that exist on their own. Schema policy applies to tuple writes in
the same way as to other writes. Types are checked, `strict` mode rejects
undeclared attributes, and a `required` attribute cannot be deleted from a
record.

The API and the CLI accept attribute-level URIs in the same way.
`xdb records get xdb://com.example/posts/post-123#title` returns only that
attribute. `xdb records delete xdb://com.example/posts/post-123#title --force`
deletes only that tuple.

## Related Concepts

- [Records](records.md) — Groups of tuples with the same path
- [Types](types.md) — The type system behind tuple values
- [URIs](uris.md) — How tuples are addressed
- [Stores](stores.md) — Tuple-level methods on the store facade
- [Drivers](drivers.md) — Tuples as the storage contract
