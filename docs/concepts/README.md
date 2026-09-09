# Concepts

XDB stores data as [tuples](tuples.md) grouped into [records](records.md). Use these guides to understand the data model and work with it through the CLI or Go API.

The [CLI reference](../../cmd/xdb/cli/CONTEXT.md) documents commands, flags, and payloads.

## Data Model

- [Tuples](tuples.md): The smallest unit of data in XDB

- [Records](records.md): Groups of tuples that represent one entity

- [Schemas](schemas.md): Structure definitions and validation modes

- [Namespaces](namespaces.md): Logical groups of schemas

- [Versioning](versioning.md): The `_id`, `_version`, and `_updated` system attributes, and optimistic concurrency

## Addressing

- [URIs](uris.md): Unique identifiers for all XDB resources

## Type System

- [Types](types.md): Supported value types and typed accessors

## Querying

- [Filters](filters.md): CEL record filters and SQL generation

## Importing Types

- [Bring Your Own Types](bring-your-own-types.md): Import protobuf, JSON Schema, and Go struct types into a schema

## Storage & Encoding

- [Stores](stores.md): The store facade, its middleware, and construction with `store.New`

- [Drivers](drivers.md): The storage contract that a driver implements

- [Encoding](encoding.md): JSON Schema import and JSON encoding of records

## Errors

- [Errors](errors.md): Sentinels, the `[xdb/pkg]` prefix, and the error tag vocabulary

## CLI & Daemon

- [Config](config.md): Config file loading, validation, and defaults

- [Daemon](daemon.md): Daemon lifecycle
