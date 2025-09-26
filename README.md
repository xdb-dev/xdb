# XDB

XDB is a database library that provides a tuple-based abstraction for modeling, storing, and querying data across multiple databases. Rather than writing database-specific schemas, queries, and migrations, XDB allows developers to model their domain once and use it with one or more databases.

## Why XDB?

Read about the motivation behind XDB in [Introducing XDB](https://raviatluri.in/articles/introducing-xdb).

## Core Concepts

### Tuple

A **Tuple** is the fundamental building block in XDB. It combines:

- ID: a string array that uniquely identifies the entity
- Attr: a string array that identifies the attribute
- Value: The attribute's value
- Options: Key-value pairs for metadata

![tuple.png](./docs/tuple.png)

### Record

A **Record** is a collection of tuples that share the same ID. Records are similar to objects, structs, or rows in a database.

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
