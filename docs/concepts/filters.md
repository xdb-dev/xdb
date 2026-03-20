---
title: Filters
description: CEL-based record filtering for list operations across all store backends.
package: filter
---

# Filters

Filters narrow down record list results using [CEL](https://cel.dev/) (Common Expression Language) expressions. A single filter string works across all store backends — in-memory stores evaluate against records directly, while SQL stores push filters down as WHERE clauses.

## Syntax

```bash
xdb records list --uri xdb://myapp/posts --filter 'status == "published"'
xdb records list --uri xdb://myapp/posts --filter 'status == "published" && views >= 100'
xdb records list --uri xdb://myapp/posts --filter 'title.contains("hello")'
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

| Function     | Example                | SQL equivalent     |
| ------------ | ---------------------- | ------------------ |
| `contains`   | `name.contains("oh")`  | `LIKE '%oh%'`      |
| `startsWith` | `name.startsWith("J")` | `LIKE 'J%'`        |
| `endsWith`   | `name.endsWith("hn")`  | `LIKE '%hn'`       |
| `size`       | `size(name) > 3`       | `LENGTH(name) > 3` |

## Schema-aware vs flexible mode

When a schema is available, filter expressions are type-checked against field definitions. When no schema exists (flexible mode), all variables are dynamically typed. Unknown fields evaluate to false rather than causing errors.

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
// wc.SQL    = "(status = ? AND age >= ?)"
// wc.Params = ["active", 18]
```
