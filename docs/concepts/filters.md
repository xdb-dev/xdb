---
title: Filters
description: CEL record filters for list operations on every backend, with SQL generation.
package: filter, filter/sqlgen
---

# Filters

Filters select records with [CEL](https://cel.dev/) (Common Expression Language) expressions. XDB type-checks expressions against the schema when one is available.

One filter string works on every backend. The memory, filesystem, and redis drivers evaluate the CEL expression against each record. The SQLite driver pushes the filter down as a SQL `WHERE` clause.

A filter is a predicate of the [CLI grammar](../../cmd/xdb/cli/CONTEXT.md). `--filter`, `--limit`, and `--offset` are flags of `xdb records list`. `--fields` is a flag of `xdb records list` and `xdb records get`. Run `xdb describe --filter` for the live reference of operators and functions.

## Syntax

```bash
xdb records list xdb://myapp/posts --filter 'status == "published"'
xdb records list xdb://myapp/posts --filter 'status == "published" && views >= 100' --fields _id,title --limit 10
xdb records list xdb://myapp/posts --filter 'title.contains("hello")'
xdb records list xdb://myapp/posts --filter 'status in ["active", "pending"]'
xdb records list xdb://myapp/posts --filter '!(archived == true)' --fields _id
```

## Operators

| Operator | Example                | Description      |
| -------- | ---------------------- | ---------------- |
| `==`     | `status == "active"`   | Equality         |
| `!=`     | `status != "closed"`   | Inequality       |
| `>`      | `age > 30`             | Greater than     |
| `>=`     | `age >= 18`            | Greater or equal |
| `<`      | `score < 100`          | Less than        |
| `<=`     | `score <= 99.9`        | Less or equal    |
| `&&`     | `a == 1 && b == 2`     | Logical AND      |
| `\|\|`   | `a == 1 \|\| b == 2`   | Logical OR       |
| `!`      | `!(age < 18)`          | Logical NOT      |
| `in`     | `status in ["a", "b"]` | List membership  |

## Functions

| Function     | Example                | SQL equivalent                   |
| ------------ | ---------------------- | -------------------------------- |
| `contains`   | `name.contains("oh")`  | `instr(name, ?) > 0`             |
| `startsWith` | `name.startsWith("J")` | `substr(name, 1, length(?)) = ?` |
| `endsWith`   | `name.endsWith("hn")`  | `substr(name, -length(?)) = ?`   |
| `size`       | `size(name) > 3`       | `LENGTH(name) > 3`               |

### Case Sensitivity

String matching compares bytes and is case-sensitive, the same as CEL itself.
`name.contains("hello")` does not match a stored value of `"Hello World"`.
This rule is the same on every backend. On SQLite, `contains`, `startsWith`,
and `endsWith` compile to `instr` and `substr`, not to `LIKE`. `LIKE` in
SQLite ignores ASCII case by default, which gives results different from the
in-memory CEL evaluation on the other backends.

## Schema-Aware and Schema-Free Filters

When a schema is available, the filter expression is type-checked against the
field definitions. The handling of an unknown field then depends on the
schema mode:

- `strict`: an unknown field is a compile error. The filter is rejected
  before it runs. The error names the field and lists the available fields of
  the schema in sorted order.

- `flexible` and `dynamic`: an unknown field is accepted as a
  dynamically typed variable. A record that lacks the attribute does not
  match. There is no error.

- Schema-free (no definition): every variable is dynamically typed, with
  the same result. A query on a namespace URI, or on a schema that no longer
  exists, is also schema-free.

The unknown-field check happens in `filter.Compile`, before any driver sees
the filter. The SQLite driver calls `filter.Compile` before it generates SQL.
The store facade calls `filter.Compile` before the in-memory evaluation. Both
paths use the same definition, so the check is the same on every backend.

### System Attributes

`_id`, `_version`, and `_updated` can be used in a filter in every mode. You
do not declare them, and they never trigger the unknown-field rejection of
`strict` mode:

```
_version > 5
_updated > timestamp("2026-01-01T00:00:00Z")
_id.startsWith("user-")
```

`_version` and `_updated` are stamped into every definition, so they are real
columns in a column table. `_id` is projected from the record path. It is not
stored as a tuple. It resolves to the ID column or key that every backend
already uses. See [Versioning](versioning.md).

## Relationship to AIP-160

XDB follows the field traversal, comparison, and function-call patterns described by [AIP-160](https://google.aip.dev/160), but uses CEL syntax:

| Concept     | AIP-160     | XDB (CEL)  |
| ----------- | ----------- | ---------- |
| Equality    | `=`         | `==`       |
| Logical AND | `AND`       | `&&`       |
| Logical OR  | `OR`        | `\|\|`     |
| Negation    | `NOT`, `-`  | `!`        |

CEL supports type checking before evaluation and translation to SQL. Its evaluation model is not Turing-complete.

## Go Usage

```go
// Compile once, evaluate many times.
f, err := filter.Compile(`status == "active" && age >= 18`, schemaDef)

// Evaluate against one record.
match, err := f.Match(record)

// Keep only the records that match.
results, err := f.Filter(records)
```

## SQL Generation

The `filter/sqlgen` package converts a compiled filter to parameterized SQL:

```go
wc, err := sqlgen.Generate(f, sqlgen.ColumnStrategy, "")
// wc.SQL    = `(("status" = ?) AND ("age" >= ?))`
// wc.Params = ["active", 18]
```

Column names are double-quoted in the generated SQL. Under `ColumnStrategy`,
a field that is not declared on the schema of the compiled filter returns
`sqlgen.ErrUnknownColumn` instead of a raw "no such column" error. This
happens in `dynamic` mode when a filter references a field that no record has
written yet. The SQLite driver maps this error to a query-pushdown refusal,
and the store falls back to a scan.

## Related Concepts

- [AIP-160: Filtering](https://google.aip.dev/160): Google's filtering standard

- [AIP-132: List](https://google.aip.dev/132): Standard List method design

- [AIP-158: Pagination](https://google.aip.dev/158): Page token and page size patterns

- [CEL specification](https://cel.dev/): Common Expression Language

- [Stores](stores.md): How filters integrate with the store facade
