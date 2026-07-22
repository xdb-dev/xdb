---
title: Filters
description: CEL-based record filtering for list operations across all store backends.
package: filter
---

# Filters

Filters narrow down record list results using [CEL](https://cel.dev/) (Common Expression Language) expressions. The filter design follows [Google's AIP-160 filtering standard](https://google.aip.dev/160), which defines a unified filtering language across Google APIs. XDB uses CEL as the evaluation engine for strict type-safety and predictable performance.

A single filter string works across all store backends — in-memory stores evaluate CEL against records directly, while SQL stores push filters down as WHERE clauses.

A filter is a **predicate primitive** of the [CLI grammar](../../cmd/xdb/cli/CONTEXT.md) — it composes with `--fields`, `--limit`, and `--offset` on any `list` action. Run `xdb describe --filter` for the live operator/function reference.

## Syntax

```bash
xdb records list xdb://myapp/posts --filter 'status == "published"'
xdb records list xdb://myapp/posts --filter 'status == "published" && views >= 100' --fields id,title --limit 10
xdb records list xdb://myapp/posts --filter 'title.contains("hello")'
xdb records list xdb://myapp/posts --filter 'status in ["active", "pending"]'
xdb records list xdb://myapp/posts --filter '!(archived == true)' --fields id
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

| Function     | Example                | SQL equivalent                        |
| ------------ | ----------------------- | -------------------------------------- |
| `contains`   | `name.contains("oh")`  | `instr(name, ?) > 0`                   |
| `startsWith` | `name.startsWith("J")` | `substr(name, 1, length(?)) = ?`       |
| `endsWith`   | `name.endsWith("hn")`  | `substr(name, -length(?)) = ?`         |
| `size`       | `size(name) > 3`       | `LENGTH(name) > 3`                     |

### Case sensitivity

String matching is byte-wise case-sensitive, matching CEL's own semantics —
`name.contains("hello")` does not match a stored value of `"Hello World"`.
This holds uniformly across every backend. On SQLite, this is why `contains`/
`startsWith`/`endsWith` compile to `instr`/`substr` rather than `LIKE`:
SQLite's `LIKE` is ASCII case-insensitive by default, which would otherwise
diverge from the in-memory CEL evaluation used by the other backends.

## Schema-aware vs flexible mode

When a schema is available, filter expressions are type-checked against
field definitions. Unknown-field handling then depends on the schema's
mode:

- **Strict**: an unknown field is a compile error — the filter is rejected
  before it runs, naming the field and listing the schema's available
  fields (sorted).
- **Flexible** and **dynamic**: an unknown field is accepted as a
  dynamically typed variable. Records that lack the attribute simply don't
  match — no error.
- **No schema** (flexible mode with no field definitions): every variable
  is dynamically typed, same as above.

Unknown-field rejection is enforced at the same point regardless of
backend: SQLite rejects it during `filter.Compile` (before any SQL is
generated), and non-strict backends (or a strict schema evaluated through
the in-memory fallback) enforce it identically since the same `filter.Compile`
call governs both paths.

## Relationship to AIP-160

XDB follows the [AIP-160](https://google.aip.dev/160) filtering standard conceptually — field traversal, comparison operators, and function calls all match AIP-160 patterns. The implementation uses CEL's stricter syntax conventions:

| Concept    | AIP-160     | XDB (CEL)  |
| ---------- | ----------- | ---------- |
| Equality   | `=`         | `==`       |
| Logical AND | `AND`      | `&&`       |
| Logical OR  | `OR`       | `\|\|`     |
| Negation   | `NOT`, `-`  | `!`        |

CEL was chosen over AIP-160's looser syntax for compile-time type checking, safe evaluation (non-Turing-complete), and direct SQL generation.

## Go usage

```go
// Compile once, evaluate many times.
f, err := filter.Compile(`status == "active" && age >= 18`, schemaDef)

// Evaluate against a record.
match, err := filter.Match(f, record)

// Filter a slice of records.
results, err := filter.Records(f, records)
```

## SQL generation

The `filter/sqlgen` package converts compiled filters to parameterized SQL:

```go
wc, err := sqlgen.Generate(f, sqlgen.ColumnStrategy, "")
// wc.SQL    = `("status" = ? AND "age" >= ?)`
// wc.Params = ["active", 18]
```

Column names are double-quoted in the generated SQL. Under
[ColumnStrategy], a field not declared on the compiled filter's schema
(dynamic mode, referencing a field no record has written yet) yields
`sqlgen.ErrUnknownColumn` rather than a raw "no such column" error; stores
map this to a query-pushdown refusal so the caller falls back to a scan.

## Related

- [AIP-160: Filtering](https://google.aip.dev/160) — Google's filtering standard
- [AIP-132: List](https://google.aip.dev/132) — Standard List method design
- [AIP-158: Pagination](https://google.aip.dev/158) — Page token and page size patterns
- [CEL specification](https://cel.dev/) — Common Expression Language
- [Stores](stores.md) — How filters integrate with store backends
