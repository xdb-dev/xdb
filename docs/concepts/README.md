# Concepts

This directory contains documentation for XDB's core concepts. Each document covers a single concept in depth, including its purpose, structure, and usage.

## Data Model

- [Tuples](tuples.md) — The fundamental building block of XDB
- [Records](records.md) — Groups of tuples representing a single entity
- [Schemas](schemas.md) — Structure definitions and validation modes
- [Namespaces](namespaces.md) — Logical grouping of schemas

## Addressing

- [URIs](uris.md) — Unique resource identifiers for all XDB resources

## Type System

- [Types](types.md) — Supported value types and typed accessors

## Querying

- [Filters](filters.md) — CEL-based record filtering with SQL generation

## Storage & Encoding

- [Stores](stores.md) — Storage interfaces and implementations
- [Encoding](encoding.md) — JSON encoding and decoding of records

## CLI & Daemon

- [Configuration](config.md) — Config file loading, validation, and defaults
- [Daemon](daemon.md) — Background daemon lifecycle management
