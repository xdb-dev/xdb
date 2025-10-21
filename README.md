# XDB

XDB is a database library that provides a tuple-based abstraction for modeling, storing, and querying data across multiple databases. Rather than writing database-specific schemas, queries, and migrations, XDB allows developers to model their domain once and use it with one or more databases.

## Why XDB?

Read about the motivation behind XDB in [Introducing XDB](https://raviatluri.in/articles/introducing-xdb).

## Core Concepts

The XDB data model can be visualized as a tree of **Repositories**, **Collections**, **Records**, and **Tuples**.

```
┌─────────────────────────────────┐
│           Repository            │
└────────────────┬────────────────┘
                 ↓
┌─────────────────────────────────┐
│           Collection            │
└────────────────┬────────────────┘
                 ↓
┌─────────────────────────────────┐
│             Record              │
└────────────────┬────────────────┘
                 ↓
┌─────────────────────────────────┐
│             Tuple               │
├─────────────────────────────────┤
│   ID | Attr | Value | Options   │
└─────────────────────────────────┘
```

### Tuple

A **Tuple** is the fundamental building block in XDB. It combines:

- ID: a string array that uniquely identifies the record
- Attr: a string array that identifies the attribute
- Value: The attribute's value
- Options: Key-value pairs for metadata

![tuple.png](./docs/tuple.png)

### Record

One or more **Tuples**, with the same **ID**, make up a **Record**. Records are similar to objects, structs, or rows in a database. Records typically represent a single entity or object of domain data.

### Collection

A **Collection** is a group of records with the same **Schema**.

### Repository

A **Repository** is an isolated namespace for data. It is designed to support multi-tenant storage and isolation.

### Schema

A **Schema** is a definition of your domain entities and their relationships. It is used to validate and enforce the structure of the data.

### URI

XDB URIs are valid Uniform Resource Identifiers (URI) according to [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986). URIs are used to uniquely identify resources in XDB.

The general format of a URI is:

```
    [SCHEME]://[DOMAIN] [ "/" PATH] [ "?" QUERY] [ "#" FRAGMENT]
```

XDB URIs follow the following format:

```
    xdb:// REPOSITORY [ "/" COLLECTION ] [ "/" RECORD ] [ "#" ATTRIBUTE ]
```

- The scheme is always `xdb://`.
- Repository is mandatory.
- Collection, Record, and Attribute are conditionally required.
- All components MUST use only alphanumeric (**A-Za-z0-9**), period, hyphen, underscore, and colon (**.-\_:**).
- Attributes when present must be valid JSON Path strings.

Valid examples:

```
Repository: xdb://example.com
Collection: xdb://example.com/core.domain.posts
Record:     xdb://example.com/core.domain.posts/123-456-789
Attribute:  xdb://example.com/core.domain.posts/123-456-789#author.id
```

## Supported Types

| Type      | PostgreSQL       | Description                  |
| --------- | ---------------- | ---------------------------- |
| String    | TEXT             | UTF-8 string                 |
| Integer   | BIGINT           | 64-bit signed integer        |
| Float     | DOUBLE PRECISION | 64-bit floating point number |
| Boolean   | BOOLEAN          | True or False                |
| Timestamp | TIMESTAMPZ       | Date and time in UTC         |
| JSON      | JSONB            | JSON data type               |
| Bytes     | BYTEA            | Binary data                  |

## Building Blocks

### Drivers

Drivers serve as the bridge between XDB's tuple-based model and specific database implementations. All drivers implement basic **Reader** and **Writer** capabilities, with advanced features like full-text search, aggregation, and iteration available based on the database's capabilities.

### Stores

Stores provide higher-level APIs that combine multiple drivers to support common use-cases. Store implementations satisfy capability interfaces, allowing them to be used as drivers or layered together for complex scenarios.
