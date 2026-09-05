---
title: URIs
description: RFC 3986 compliant unique resource identifiers for all XDB resources.
package: core
---

# URIs

XDB **URIs** are valid Uniform Resource Identifiers as defined in [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986). A URI uniquely identifies every resource in XDB: namespaces, schemas, records, and attributes. Every level of a URI addresses a real thing. A namespace groups schemas. A schema groups records. A path names a set of tuples. `#attr` names one tuple.

A `URI` is an immutable value type. Construct one with `NewURI`, `ParseURI`, or `ParsePath`.

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

# Attribute — a single value in a record
xdb://com.example/posts/123-456-789#title
```

The more components a URI has, the more specific the reference is. The `Depth()` method reports this specificity: 1 for a namespace, 2 for a schema, and 3 for a record. The attribute fragment does not change the depth.

## URIs in the CLI

The URI is the **noun** of the [CLI grammar](../../cmd/xdb/cli/CONTEXT.md). Every action has the form `xdb <resource> <action> <URI> [flags]`. The URI depth selects the resource. The action set is not the same for every resource:

- Records support `get`, `list`, `create`, `update`, `upsert`, and `delete`.
- Schemas support `get`, `list`, `create`, `update`, and `delete`.
- Namespaces support only `list` and `get`.

`xdb watch <uri>` is a top-level command, not a resource action. Run `xdb describe --actions` for the live matrix of actions and resources.

The shorthand commands `xdb get`, `xdb ls`, and `xdb rm` also select the resource from the URI depth. For example, `xdb get xdb://com.example/posts/123` is the same as `xdb records get xdb://com.example/posts/123`.

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

Accessors return plain strings. An absent component is the empty string.

```go
uri.NS()         // string — namespace
uri.Schema()     // string — schema ("" for a namespace URI)
uri.ID()         // string — record ID ("" for a namespace or schema URI)
uri.Attr()       // string — attribute ("" if there is no fragment)
uri.Depth()      // int    — 1 (namespace), 2 (schema), or 3 (record)
uri.Path()       // string — path without the scheme
uri.String()     // string — full URI with the scheme
uri.SchemaURI()  // *URI   — URI with only NS + Schema
uri.RecordURI()  // *URI   — URI with NS + Schema + ID (attribute dropped)
uri.RecordPath() // string — the ns/schema/id record key (attribute dropped)
```

## Constructing URIs

### NewURI

`NewURI(ns, parts...)` builds a namespace, schema, or record URI from its
parts. The first part is the schema and the second part is the ID. Both parts
are optional:

```go
nsURI, err := core.NewURI("com.example")                     // xdb://com.example
schemaURI, err := core.NewURI("com.example", "posts")        // xdb://com.example/posts
recordURI, err := core.NewURI("com.example", "posts", "123") // xdb://com.example/posts/123
uri := core.MustNewURI("com.example", "posts", "123")        // panics on invalid input
```

### Parsing

```go
uri, err := core.ParseURI("xdb://com.example/posts/123#title")
uri := core.MustParseURI("xdb://com.example/posts/123#title")
```

NS, Schema, and Attribute cannot contain `/`. An ID can contain `/`. Trailing
path segments join into the ID. Every valid URI round-trips:
`ParseURI(u.String())` equals `u`.

## Equality

URIs are comparable value types. `==` on the dereferenced pointers compares
the components:

```go
a := core.MustParseURI("xdb://com.example/posts/123")
b := core.MustParseURI("xdb://com.example/posts/123")
*a == *b // true
```

## JSON Serialization

A URI serializes to JSON as a quoted string, and parses back from one:

```json
"xdb://com.example/posts/123"
```

## Related Concepts

- [Namespaces](namespaces.md) — The NS component
- [Schemas](schemas.md) — The Schema component
- [Records](records.md) — The ID component
- [Tuples](tuples.md) — The Attribute fragment
