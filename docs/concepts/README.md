# Concepts

This directory documents the core concepts of XDB. The data model covers URIs, tuples, records, schemas, namespaces, and types. The infrastructure around the data model covers stores, drivers, encoding, config, and the daemon.

The [CLI](../../cmd/xdb/cli/CONTEXT.md) exposes these concepts as the parts of a small language. A URI is the noun. Records, schemas, and namespaces are what a URI resolves to. Types define the values that filters and payloads operate on.

## Data Model

- [Tuples](tuples.md) — The smallest unit of data in XDB
- [Records](records.md) — Groups of tuples that represent one entity
- [Schemas](schemas.md) — Structure definitions and validation modes
- [Namespaces](namespaces.md) — Logical groups of schemas
- [Versioning](versioning.md) — The `_id`, `_version`, and `_updated` system attributes, and optimistic concurrency

## Addressing

- [URIs](uris.md) — Unique identifiers for all XDB resources

## Type System

- [Types](types.md) — Supported value types and typed accessors

## Querying

- [Filters](filters.md) — CEL record filters and SQL generation

## Importing Types

- [Bring Your Own Types](bring-your-own-types.md) — Import protobuf, JSON Schema, and Go struct types into a schema

## Storage & Encoding

- [Stores](stores.md) — The store facade, its middleware, and construction with `store.New`
- [Drivers](drivers.md) — The storage contract that a driver implements
- [Encoding](encoding.md) — JSON encoding and decoding of records

## CLI & Daemon

- [Config](config.md) — Config file loading, validation, and defaults
- [Daemon](daemon.md) — Daemon lifecycle
