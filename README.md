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

- ID: a string that uniquely identifies the record
- Attr: a string that identifies the attribute. It supports dot-separated nesting.
- Value: The attribute's value
- Options: Key-value pairs for metadata

![tuple.png](./docs/tuple.png)

### Record

One or more **Tuples**, with the same **ID**, make up a **Record**. Records are similar to objects, structs, or rows in a database. Records typically represent a single entity or object of domain data.

### Collection

A **Collection** is a collection of records with the same **Schema**. Collections are identified by their schema name and are unique within a repository.

### Repository

A **Repository** is a data repository holding one or more **Collections**. Repositories are typically used to group collections by domain, application, or tenant. Repositories are identified by their name (referred to as "NS" in the URI).

### Schema

A **Schema** is a definition of your domain entities and their relationships. Schemas can be "strict" or "flexible". Strict schemas enforce a predefined structure on the data, while flexible schemas allow for arbitrary data.

### URI

XDB URIs are valid Uniform Resource Identifiers (URI) according to [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986). URIs are used to uniquely identify resources in XDB.

The general format of a URI is:

```
    [SCHEME]://[DOMAIN] [ "/" PATH] [ "?" QUERY] [ "#" FRAGMENT]
```

XDB URIs follow the following format:

```
    xdb:// NS [ "/" SCHEMA ] [ "/" ID ] [ "#" ATTRIBUTE ]
```

```
    xdb://com.example/posts/123-456-789#author.id
    └─┬──┘└────┬────┘└──┬─┘└─────┬─────┘└─────┬─────┘
   scheme     NS    SCHEMA      ID        ATTRIBUTE
              └───────────┬───────────┘
                      RECORD KEY
                        (rkey)
```

The components of the URI are:

- **NS**: The name of the repository.
- **SCHEMA**: The name of the collection.
- **ID**: The unique identifier of the record.
- **ATTRIBUTE**: The name of the attribute.
- **RECORD KEY (rkey)**: NS, SCHEMA, and ID combined uniquely identify a record

Valid examples:

```
Repository: xdb://com.example
Collection: xdb://com.example/posts
Record:     xdb://com.example/posts/123-456-789
Attribute:  xdb://com.example/posts/123-456-789#author.id
```

## Supported Types

| Type      | PostgreSQL       | SQLite  | Description                  |
| --------- | ---------------- | ------- | ---------------------------- |
| String    | TEXT             | TEXT    | UTF-8 string                 |
| Integer   | BIGINT           | INTEGER | 64-bit signed integer        |
| Float     | DOUBLE PRECISION | REAL    | 64-bit floating point number |
| Boolean   | BOOLEAN          | INTEGER | True or False                |
| Timestamp | TIMESTAMPZ       | INTEGER | Date and time in UTC         |
| JSON      | JSONB            | TEXT    | JSON data type               |
| Bytes     | BYTEA            | BLOB    | Binary data                  |

## Building Blocks

### Drivers

Drivers serve as the bridge between XDB's tuple-based model and specific database implementations. All drivers implement basic **Reader** and **Writer** capabilities, with advanced features like full-text search, aggregation, and iteration available based on the database's capabilities.

### Stores

Stores provide higher-level APIs that combine multiple drivers to support common use-cases. Store implementations satisfy capability interfaces, allowing them to be used as drivers or layered together for complex scenarios.
