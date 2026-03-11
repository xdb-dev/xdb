---
title: Tuples
description: The fundamental building block of XDB — an immutable unit of path, attribute, and typed value.
package: core
---

# Tuples

A **Tuple** is the fundamental building block in XDB. Every piece of data in XDB is ultimately stored as a tuple.

## Structure

A tuple combines four components:

| Component | Type    | Description                                       |
| --------- | ------- | ------------------------------------------------- |
| **Path**  | `*URI`  | References the record (NS + Schema + ID)          |
| **Attr**  | `*Attr` | Attribute name, supports dot-separated nesting    |
| **Value** | `*Value`| Typed value container                             |

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

### Using the Builder (recommended)

The builder provides a fluent API for constructing tuples:

```go
tuple := core.New().
    NS("com.example").
    Schema("posts").
    ID("post-123").
    MustTuple("title", "Hello World")
```

Typed builder methods avoid reflection overhead:

```go
tuple := core.New().
    NS("com.example").
    Schema("posts").
    ID("post-123").
    Str("title", "Hello World")
```

Available typed methods: `Bool()`, `Int()`, `Uint()`, `Float()`, `Str()`, `Bytes()`, `Time()`, `JSON()`.

### Using NewTuple

```go
tuple := core.NewTuple("com.example/posts/post-123", "title", "Hello World")
```

`NewTuple` panics on invalid input. The path argument is a URI path (without the `xdb://` scheme).

## Accessing Data

### Path Components

```go
tuple.NS()     // *NS     — namespace
tuple.Schema() // *Schema — schema name
tuple.ID()     // *ID     — record identifier
tuple.Attr()   // *Attr   — attribute name
tuple.URI()    // *URI    — full URI including attribute fragment
tuple.Path()   // *URI    — record URI (without attribute)
```

### Typed Value Accessors

Tuples expose `As*` methods (`AsStr()`, `AsInt()`, `AsBool()`, etc.) for type-safe value extraction. All accessors are nil-safe and return `(T, error)`. See [Types](types.md) for the full list and details.

```go
title, err := tuple.AsStr()
```

## Dot-Separated Attributes

Attributes support dot notation for representing nested data (e.g., `author.name`, `profile.address.city`). When encoded to JSON, these are unfolded into nested objects. See [Encoding](encoding.md) for details.

## Related Concepts

- [Records](records.md) — Groups of tuples with the same path
- [Types](types.md) — The type system behind tuple values
- [URIs](uris.md) — How tuples are addressed
