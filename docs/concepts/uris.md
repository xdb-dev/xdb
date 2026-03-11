---
title: URIs
description: RFC 3986 compliant unique resource identifiers for all XDB resources.
package: core
---

# URIs

XDB **URIs** are valid Uniform Resource Identifiers following [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986). Every resource in XDB — namespaces, schemas, records, and attributes — is uniquely identified by a URI.

## Format

```
xdb://NS[/SCHEMA][/ID][#ATTRIBUTE]
```

```
xdb://com.example/posts/123-456-789#author.id
└─┬──┘└────┬────┘└──┬─┘└─────┬─────┘└─────┬─────┘
scheme     NS    SCHEMA      ID        ATTRIBUTE
           └───────────┬───────────┘
                     path
```

| Component     | Required | Description                             |
| ------------- | -------- | --------------------------------------- |
| **Scheme**    | Yes      | Always `xdb://`                         |
| **NS**        | Yes      | [Namespace](namespaces.md) identifier   |
| **Schema**    | No       | [Schema](schemas.md) name               |
| **ID**        | No       | [Record](records.md) identifier         |
| **Attribute** | No       | [Tuple](tuples.md) attribute (fragment) |

## URI Levels

Each level of the URI identifies a different resource type:

```bash
# Namespace — groups schemas
xdb://com.example

# Schema — groups records
xdb://com.example/posts

# Record — a single entity
xdb://com.example/posts/123-456-789

# Attribute — a single value within a record
xdb://com.example/posts/123-456-789#title
```

The more components present, the more specific the reference. Commands like `get`, `ls`, and `rm` use the URI level to determine what operation to perform.

## Paths

A **path** is the URI without the `xdb://` scheme:

```
com.example/posts/123-456-789
```

Paths are used internally for storage keys and as arguments to `NewTuple` and `ParsePath`.

```go
// Parse a full URI (scheme required)
uri, err := core.ParseURI("xdb://com.example/posts/123")

// Parse a path (no scheme)
uri, err := core.ParsePath("com.example/posts/123")
```

## Component Access

```go
uri.NS()        // *NS     — namespace
uri.Schema()    // *Schema — schema (nil if namespace-only URI)
uri.ID()        // *ID     — record ID (nil if schema-only URI)
uri.Attr()      // *Attr   — attribute (nil if no fragment)
uri.Path()      // string  — path without scheme
uri.String()    // string  — full URI with scheme
uri.SchemaURI() // *URI    — URI with only NS + Schema
```

## Constructing URIs

### Builder (recommended)

```go
uri := core.New().
    NS("com.example").
    Schema("posts").
    ID("123").
    MustURI()
// xdb://com.example/posts/123
```

### Parsing

```go
uri, err := core.ParseURI("xdb://com.example/posts/123#title")
uri := core.MustParseURI("xdb://com.example/posts/123#title")
```

## Equality

Two URIs are equal if all their components match:

```go
a := core.MustParseURI("xdb://com.example/posts/123")
b := core.MustParseURI("xdb://com.example/posts/123")
a.Equals(b) // true
```

## JSON Serialization

URIs serialize to and from JSON as quoted strings:

```json
"xdb://com.example/posts/123"
```

## Related Concepts

- [Namespaces](namespaces.md) — The NS component
- [Schemas](schemas.md) — The Schema component
- [Records](records.md) — The ID component
- [Tuples](tuples.md) — The Attribute fragment
