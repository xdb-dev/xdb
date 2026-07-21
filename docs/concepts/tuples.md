---
title: Tuples
description: The fundamental building block of XDB — an immutable unit of path, attribute, and typed value.
package: core
---

# Tuples

A **Tuple** is the fundamental building block in XDB — an addressable fact:
`xdb://ns/schema/id#attr = value`. Every piece of data in XDB is ultimately a
tuple, and every other structure is built from them. A [Record](records.md) is
just the set of tuples that share a path.

## Structure

A tuple combines three components:

| Component | Type     | Description                                    |
| --------- | -------- | ---------------------------------------------- |
| **Path**  | `*URI`   | References the record (NS + Schema + ID)       |
| **Attr**  | `string` | Attribute name, supports dot-separated nesting |
| **Value** | `*Value` | Typed value container                          |

```
┌─────────────────────────────────────────────────┐
│                     Tuple                        │
├──────────┬──────────┬───────────────────────────┤
│   Path   │   Attr   │          Value             │
│ (URI)    │ (string) │  (typed: str, int, ...)    │
└──────────┴──────────┴───────────────────────────┘
```

A tuple is **immutable** after creation. Its path, attribute, and value cannot change.

## Creating Tuples

```go
tuple := core.NewTuple("com.example/posts/post-123", "title", "Hello World")
```

`NewTuple` panics on invalid input. The path argument is a URI path (without
the `xdb://` scheme). The value is inferred from its Go type; see
[Types](types.md) for the supported types.

Tuples are usually created through a [Record](records.md), which owns their
shared path:

```go
record := core.NewRecord("com.example", "posts", "post-123").
    Set("title", "Hello World")

tuple := record.Get("title")
```

## Accessing Data

### Path Components

```go
tuple.Path()   // *URI   — record URI (without attribute)
tuple.Attr()   // string — attribute name
tuple.URI()    // *URI   — full URI including attribute fragment
tuple.Value()  // *Value — typed value

tuple.Path().NS()     // string — namespace
tuple.Path().Schema() // string — schema name
tuple.Path().ID()     // string — record identifier
```

### Typed Value Accessors

Tuples expose `As*` methods (`AsStr()`, `AsInt()`, `AsBool()`, etc.) for
type-safe value extraction. Each returns `(T, error)`. Reading a **missing**
attribute (a nil tuple, e.g. `record.Get("tpyo")`) returns the zero value and
[`ErrAttrNotFound`](../../core/errors.go), distinguishing a typo from an empty
value. See [Types](types.md) for the full list and details.

```go
title, err := record.Get("title").AsStr()
```

## Dot-Separated Attributes

Attributes support dot notation for representing nested data (e.g., `author.name`, `profile.address.city`). When encoded to JSON, these are unfolded into nested objects. See [Encoding](encoding.md) for details.

## Tuples in Stores

Tuples are not just a data-model detail — they are the unit the storage
layer speaks. Every attr is addressable and operable through a
[Store](stores.md) via its `#attr` URI:

```go
st := store.New(xdbmemory.NewDriver())

// Merge tuples into records (creates the record if absent, leaves
// other attrs untouched). Tuples may span records.
err := st.PutTuples(ctx,
    core.NewTuple("com.example/posts/post-123", "title", "Hello"),
    core.NewTuple("com.example/posts/post-123", "rating", 4.5),
)

// Point-read one tuple. Absence — of the record or the attr — is
// core.ErrNotFound.
tuple, err := st.GetTuple(ctx,
    core.MustParseURI("xdb://com.example/posts/post-123#title"))

// Batch point reads omit absent attrs instead of erroring.
tuples, err := st.GetTuples(ctx, uris...)

// Delete individual tuples (idempotent). Removing a record's last
// tuple removes the record.
err = st.DeleteTuples(ctx,
    core.MustParseURI("xdb://com.example/posts/post-123#rating"))
```

A record springs into existence when its first tuples are put and
ceases to exist when its last tuple is deleted — records are views
over tuple sets, not containers that exist independently of them.
Schema policy applies to tuple writes like any other write: types are
checked, strict mode rejects undeclared attrs, and a `Required` attr
cannot be deleted out of a record.

The API and CLI accept attr-level URIs in the same spirit:
`xdb get xdb://com.example/posts/post-123#title` returns just that
attribute, and `xdb delete` of an attr-level URI removes just that
tuple.

## Related Concepts

- [Records](records.md) — Groups of tuples with the same path
- [Types](types.md) — The type system behind tuple values
- [URIs](uris.md) — How tuples are addressed
- [Stores](stores.md) — Tuple-level verbs on the store facade
- [Drivers](drivers.md) — Tuples as the storage contract
