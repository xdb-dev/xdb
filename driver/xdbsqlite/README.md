# SQLite Driver for XDB

There are two drivers for SQLite:

## KVStore

KVStore is a key-value store for SQLite. It stores tuples in SQLite tables as key-value pairs. Each kind is stored in a separate table.

Each table has:

- `key` is an unique key of the tuple
- `id` is the id of the tuple
- `attr` is attribute name
- `value` is the value of the attribute

## SQLStore

SQLStore is a store that stores tuples as columns & rows in tables. The table schema is defined by the record schema.
