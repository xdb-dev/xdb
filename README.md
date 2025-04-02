# XDB

XDB is a Go library providing a tuple-based abstraction for modeling, storing, and querying data. It is designed to be a lightweight, flexible, and efficient alternative to implementing database-specific models, queries, and repository layers.

The primary use-case for XDB is making it easier to build and maintain fully managed data services.

## Tuple

Tuple is the fundamental data structure of XDB. It is based on N-quads. XDB breaks down any kind of data into tuples.

Tuples are grouped into "Records" which are similar to objects/structs/rows in a database. Records are organized by "kind" - similar to tables and "id" - similar to primary keys.

A typical tuple looks like this:

![tuple.png](./docs/tuple.png)

- Kind: Type/kind of the record where the tuple belongs to
- ID: Unique identifier of the record
- Name: Name of the attribute. It is similar to column names in a database
- Value: Value of the attribute. It can be any one of the supported types
- Options: Options are key-value pairs used by different components & operations

## Edge

Edge is a special kind of tuple that defines a unidirectional relationship between two records. Edges are used to model relationships between records.

## Record

Record is a collection of tuples grouped by kind and id. It is similar to a row in a database table. Records are uniquely identified by combination of kind and id.

## Key

Key is a generic identifier used to uniquely identify records, tuples and edges.

## Types

XDB supports the following types:

| Type      | PostgreSQL       | Description                  |
| --------- | ---------------- | ---------------------------- |
| String    | TEXT             | UTF-8 string                 |
| Integer   | BIGINT           | 64-bit signed integer        |
| Float     | DOUBLE PRECISION | 64-bit floating point number |
| Boolean   | BOOLEAN          | True or False                |
| Timestamp | TIMESTAMPZ       | Date and time in UTC         |
| JSON      | JSONB            | JSON data type               |
| Bytes     | BYTEA            | Binary data                  |

### Hybrid Types

| Type  | PostgreSQL | Description                                                      |
| ----- | ---------- | ---------------------------------------------------------------- |
| Point | POINT      | Latitude and Longitude                                           |
| Money | JSONB      | Currency and Amount                                              |
| File  | JSONB      | File/Image metadata. Actual file is stored in a separate storage |

## Drivers

Drivers are interfaces for storing, querying, searching and iterating records, tuples and edges. Think of drivers as "capabilities" implemented by different databases. Drivers can be chained, layered, and combined to provide features or simply to work around limitations of a specific database.

## Stores

Stores manage storage, retrieval, indexing, and querying of records, tuples, and edges. Stores use drivers to interact with databases.
