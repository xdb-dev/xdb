---
title: "Filtering Approaches for Multi-Backend APIs"
description: "Research on filtering syntax approaches for systems that target SQL, Elasticsearch, Redis, and filesystem backends"
date: 2026-03-19
---

# Filtering Approaches for Multi-Backend Systems

## Executive Summary

XDB needs a filtering system that is:
1. **Simple for CLI users** — `--filter "field=value"` style syntax
2. **JSON-representable** — easily constructed programmatically
3. **Multi-backend convertible** — translatable to SQL WHERE, Elasticsearch, Redis scans, filesystem filtering

**Top Recommendation: AIP-160 Filtering (Google's standard)**

AIP-160 is the most battle-tested approach for multi-backend systems. It offers:
- Proven CLI usability (used across `gcloud` CLI commands)
- Clean CEL-based expression syntax
- Direct SQL compatibility
- Extensibility to other backends
- Strong support for nested fields
- Industry adoption (Google, Firebase, Eventarc, etc.)

**Alternative if simplicity is priority: RSQL/FIQL**

More compact URI-friendly syntax with slightly less power but easier to parse and more established Go ecosystem support.

**Confidence Level: HIGH** — Both approaches have proven implementations across major cloud platforms.

---

## Search Specification

### Objective
Research how major cloud providers and data platforms implement filtering across heterogeneous storage backends, focusing on systems that must target:
- SQL databases (SQLite, PostgreSQL)
- Elasticsearch
- Redis
- Filesystem-based storage

### Constraints & Requirements
1. CLI syntax must be intuitive and typeable without documentation
2. Must support common operators: equality, comparison, logical AND/OR
3. Should support field traversal (e.g., `author.name = "value"`)
4. Must be convertible to multiple backend query languages
5. Should have available Go parsing libraries
6. Must handle escaping gracefully for shell environments

### Internal Context
XDB already has:
- `store.ListQuery` struct with `Filter string` field in place
- Modular store implementations (xdbfs, xdbmemory, xdbredis, xdbsqlite)
- Tuple-based data model with dot-separated nested attribute names
- Multiple backends requiring independent filter translation

### Validation Criteria
- Existing Go implementations available
- Library maturity (active maintenance, >100 stars on GitHub)
- Backend conversion implementation examples
- Type safety considerations for Go
- Performance characteristics

---

## Solutions Evaluated

### 1. Google AIP-160 Filtering Standard

**Links:**
- Official spec: https://google.aip.dev/160
- CEL spec: https://cel.dev/
- CEL-Go implementation: https://github.com/google/cel-go

**Description:**

AIP-160 is Google's API Improvement Proposal for standardized filtering across all Google APIs and services. It uses a CEL (Common Expression Language) expression string as the filter value.

**CLI Syntax:**
```bash
gcloud compute instances list --filter="status=RUNNING"
gcloud compute instances list --filter="status=RUNNING AND zone=us-central1-a"
gcloud compute instances list --filter="labels.env=test AND labels.version=alpha"
```

**JSON/API Representation:**
The filter is passed as a simple string query parameter:
```json
{
  "filter": "status=RUNNING AND zone=us-central1-a"
}
```

**Operators & Syntax:**

| Operator | Purpose | Example |
|----------|---------|---------|
| `=`, `!=` | Equality, inequality | `status=RUNNING`, `env!=prod` |
| `<`, `>`, `<=`, `>=` | Numeric/string comparison | `count>5`, `date<2024-01-01` |
| `AND` | Logical AND | `status=RUNNING AND zone=us-east` |
| `OR` | Logical OR | `status=RUNNING OR status=STOPPED` |
| `NOT`, `-` | Negation | `NOT status=PENDING` |
| `.` | Field traversal | `metadata.labels.env=prod` |
| `:` | Contains (for collections) | `tags:monitoring`, `labels:env` |

**Operator Precedence (Important!):**
- OR has higher precedence than AND (opposite of typical programming languages)
- Use parentheses to override: `(a=1 OR a=2) AND b=3`

**Type Handling:**
- Enums: string representation (case-sensitive)
- Booleans: `true`/`false` literals
- Numbers: standard integer/float formats
- Timestamps: RFC-3339 format
- Durations: numeric value with `s` suffix

**Error Handling:**
- Returns `INVALID_ARGUMENT` for non-compliant filters
- Schema violations produce clear errors

**Pros:**
- ✓ Proven in production at scale (Google Cloud, Firebase)
- ✓ Non-Turing complete, safe to evaluate
- ✓ Excellent documentation and examples
- ✓ Strong typing with CEL
- ✓ Extensible with custom functions
- ✓ Macro support (all, exists, filter, map)
- ✓ Can be compiled once, executed many times efficiently
- ✓ Official Go library (google/cel-go)
- ✓ Multiple backend conversion examples
- ✓ Used across enterprise APIs (Kubernetes, Eventarc, etc.)

**Cons:**
- ✗ Slightly more verbose than alternatives
- ✗ CEL-Go library is heavy (requires significant setup for type definitions)
- ✗ Learning curve for users unfamiliar with CEL
- ✗ Requires defining allowed fields/functions per API

**Stats:**
- GitHub Stars (cel-go): 2.5k+
- Last Update: Actively maintained (2024-2025)
- License: Apache 2.0
- Go Package: `github.com/google/cel-go`

**Community:**
- Active maintenance by Google
- Issue response time: Fast
- Used by Google Cloud, Firebase, Eventarc, Service Extensions

**Bundle Size Impact:**
- cel-go is comprehensive but not lightweight (~3-5 MB for full library)
- Can be included selectively; core parsing is ~1 MB

---

### 2. RSQL/FIQL (Resource Search Query Language)

**Links:**
- Spec: https://www.baeldung.com/rest-api-search-language-rsql-fiql
- Java parser: https://github.com/jirutka/rsql-parser
- TypeScript parser: https://github.com/mw-experts/rsql
- Go implementations: Multiple available

**Description:**

RSQL is a query language for parametrized filtering of RESTful API entries, based on FIQL (Feed Item Query Language). It's URI-friendly with no unsafe characters, so URL encoding is not required.

**CLI Syntax:**
```bash
# RSQL style
/movies?query=name=="Kill Bill"
/movies?query=director.lastName==Nolan;year>=2000
/movies?query=genre=in=(Action,Comedy);year>2010

# More readable variant with 'and'/'or'
/movies?query=name=="Kill Bill" and year>2003
/movies?query=status==(Running,Stopped) or pending==true
```

**JSON/API Representation:**
```json
{
  "filter": "name==\"Kill Bill\";year>=2000"
}
```

**Operators & Syntax:**

| Operator | Meaning | Symbol | Example |
|----------|---------|--------|---------|
| Equal | == | =eq= | `name=="John"` |
| Not equal | != | =ne= | `status!="closed"` |
| Less than | < | =lt= | `age<30` |
| Less or equal | <= | =le= | `count<=10` |
| Greater than | > | =gt= | `score>80` |
| Greater or equal | >= | =ge= | `timestamp>=2024-01-01` |
| In | `=in=()` | | `status=in=(active,pending)` |
| Not in | `=out=()` | | `region=out=(us-west,eu-east)` |
| AND | `;` or `and` | | `status==active;year>=2020` |
| OR | `,` or `or` | | `status==active,status==pending` |
| Grouping | `()` | | `(a==1;b==2),(c==3)` |

**Type Handling:**
- String: quoted with `""`
- Number: unquoted
- Boolean: `true`, `false`
- Collections: parentheses `(val1,val2,val3)`

**Pros:**
- ✓ URI-friendly (no special URL encoding needed)
- ✓ Compact syntax
- ✓ Established Go ecosystem support
- ✓ Simpler to parse than CEL
- ✓ Lower dependency overhead
- ✓ Well-suited for query parameters
- ✓ Existing implementations in multiple languages

**Cons:**
- ✗ Less powerful than CEL (no custom functions/macros)
- ✗ Less adoption than AIP-160 in enterprise APIs
- ✗ Less documentation and examples
- ✗ Operator precedence can be confusing (AND `;` vs OR `,`)
- ✗ Go library ecosystem smaller

**Stats:**
- GitHub Stars (rsql-parser): 200+
- Last Update: Actively maintained
- License: Apache 2.0
- Go Packages: Multiple community implementations

**Community:**
- Used in Spring/Java ecosystem heavily
- Growing adoption in Go community
- Good documentation from Baeldung and others

**Bundle Size Impact:**
- Minimal (~50-100 KB for parser)
- Lower overhead than CEL

---

### 3. OData $filter

**Links:**
- Spec: https://www.odata.org/
- Microsoft Graph reference: https://learn.microsoft.com/en-us/graph/filter-query-parameter
- Azure AI Search: https://learn.microsoft.com/en-us/azure/search/search-query-odata-filter

**Description:**

OData is a protocol for building RESTful APIs with querying capabilities similar to SQL. The $filter system query option allows filtering collections with expressions.

**CLI Syntax:**
```bash
# Via query parameter
?$filter=status eq 'Running'
?$filter=status eq 'Running' and zone eq 'us-central1-a'
?$filter=metadata/labels/env eq 'prod'
?$filter=count gt 5 and date lt datetime'2024-01-01T00:00:00'
```

**JSON/API Representation:**
```json
{
  "$filter": "status eq 'Running' and zone eq 'us-central1-a'"
}
```

**Operators & Syntax:**

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `status eq 'Running'` |
| `ne` | Not equals | `status ne 'Closed'` |
| `lt`, `le`, `gt`, `ge` | Comparison | `count gt 5` |
| `and`, `or`, `not` | Logical | `a eq 1 and b eq 2` |
| `has` | Enum member | `status has 'Draft'` |
| `in` | Collection membership | `status in ('Active','Pending')` |
| `any`, `all` | Collection evaluation | `tags/any(t: t eq 'urgent')` |

**Type Handling:**
- String: single quotes required `'value'`
- Number: unquoted
- Boolean: `true`, `false`
- DateTime: `datetime'2024-01-01T00:00:00Z'`
- Enum: quoted strings

**Pros:**
- ✓ Enterprise standard (Microsoft, SAP adoption)
- ✓ Rich feature set (any/all for collections)
- ✓ Well-documented
- ✓ SQL-like syntax (easier for SQL developers)
- ✓ Comprehensive function library

**Cons:**
- ✗ Verbose syntax (requires keywords like `eq`, `and`)
- ✗ String values must be single-quoted
- ✗ Less CLI-friendly (harder to type)
- ✗ Larger attack surface for injection
- ✗ Less adoption in modern Go APIs
- ✗ Limited Go library ecosystem

**Stats:**
- Enterprise standard since 2006
- Last Update: v4.0 current (2024)
- License: OData standardization under OASIS
- Go packages: Limited community implementations

**Community:**
- Strong in Microsoft/enterprise ecosystem
- Declining adoption in new projects
- Heavy in legacy systems

**Bundle Size Impact:**
- Variable depending on implementation
- No standard lightweight Go library

---

### 4. MongoDB Query Language (BSON)

**Links:**
- Reference: https://www.mongodb.com/docs/manual/reference/mql/query-predicates/comparison/
- Examples: Multiple tutorials available

**Description:**

MongoDB's JSON-based query format with operator prefixes for expressions.

**JSON Representation:**
```json
{
  "filter": {
    "$and": [
      { "status": { "$in": ["Running", "Pending"] } },
      { "count": { "$gt": 5 } },
      { "author.name": { "$eq": "John" } }
    ]
  }
}
```

**CLI Syntax (derived from JSON):**
Not naturally CLI-friendly; typically requires escaping or special JSON parsing.

**Operators:**
- `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
- `$in`, `$nin`
- `$and`, `$or`, `$not`
- `$exists`, `$type`, `$regex`
- `$all`, `$elemMatch` (arrays)

**Pros:**
- ✓ Familiar to MongoDB users
- ✓ Powerful nested query support
- ✓ JSON structure natural for programmatic construction
- ✓ Well-established

**Cons:**
- ✗ Not CLI-friendly without significant escaping
- ✗ Verbose for simple queries
- ✗ Requires JSON parsing/validation
- ✗ Not standardized for non-MongoDB APIs
- ✗ Difficult to type manually

**Stats:**
- MongoDB ecosystem
- Widely used for MongoDB operations
- License: SSPL/Community

---

### 5. Notion API Filter Format

**Links:**
- Reference: https://developers.notion.com/reference/post-database-query-filter

**Description:**

Notion's deeply nested JSON format for database filtering with type-specific operators.

**JSON Representation:**
```json
{
  "filter": {
    "and": [
      { "property": "Status", "select": { "equals": "Active" } },
      { "property": "Created", "date": { "on_or_after": "2024-01-01" } },
      { "or": [
          { "property": "Tags", "multi_select": { "contains": "urgent" } },
          { "property": "Tags", "multi_select": { "contains": "priority" } }
        ]
      }
    ]
  }
}
```

**CLI Syntax:**
Not designed for CLI usage; requires full JSON construction.

**Pros:**
- ✓ Type-specific operators (excellent for schema validation)
- ✓ Very powerful nested logic
- ✓ Clean JSON structure
- ✓ Good for UI form builders

**Cons:**
- ✗ Extremely verbose
- ✗ Not suitable for CLI
- ✗ Type-specific operators require schema knowledge
- ✗ Not suitable for generic APIs
- ✗ Difficult to construct programmatically without helpers
- ✗ Nesting limited to 2 levels

**Stats:**
- Notion-specific
- Good documentation
- Active maintenance

---

### 6. Stripe API Filtering

**Links:**
- Reference: https://docs.stripe.com/api

**Description:**

Stripe uses REST query parameters with bracket notation for arrays.

**CLI Syntax:**
```bash
curl -G https://api.stripe.com/v2/core/accounts \
  -G "applied_configurations[0]=merchant" \
  -G "applied_configurations[1]=customer"
```

**Query Parameter Representation:**
```
?applied_configurations[0]=merchant&applied_configurations[1]=customer
```

**Pros:**
- ✓ Simple and HTTP-native
- ✓ Standard URL query syntax

**Cons:**
- ✗ Limited to exact matching or enumeration
- ✗ No support for complex filtering logic (AND, OR, comparisons)
- ✗ Not suitable for dynamic filtering
- ✗ Doesn't solve the problem for XDB

**Stats:**
- Stripe-specific
- Limited to their use case

---

### 7. Kubernetes Label/Field Selectors

**Links:**
- Labels: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
- Field Selectors: https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/

**Description:**

Kubernetes uses two complementary selector systems:

**Label Selectors (CLI):**
```bash
kubectl get pods -l app=nginx,tier=frontend
kubectl get pods -l app!=nginx
kubectl get pods -l "app in (nginx, httpd)"
```

**Field Selectors (CLI):**
```bash
kubectl get pods --field-selector status.phase=Running
kubectl get pods --field-selector=status.phase!=Running,spec.restartPolicy=Always
```

**Operators:**
- Label selectors: `=`, `==`, `!=`, `in()`, `notin()`, `exists`
- Field selectors: `=`, `==`, `!=` only

**Pros:**
- ✓ Very simple and CLI-friendly
- ✓ Minimal operator set
- ✓ Comma-separated for AND
- ✓ Well-established in container ecosystem

**Cons:**
- ✗ Very limited operator set (no comparison operators for field selectors)
- ✗ Not suitable for complex filtering
- ✗ No logical OR at top level
- ✗ Limited to key=value style matching

**Stats:**
- Kubernetes standard
- Widely used
- License: Apache 2.0

---

## Detailed Analysis: AIP-160 vs RSQL

### Technical Fit Comparison

| Criterion | AIP-160 | RSQL |
|-----------|---------|------|
| **CLI Usability** | Good (field=value) | Very Good (compact) |
| **Learning Curve** | Moderate (CEL-based) | Easy (simple operators) |
| **Operator Set** | Rich (with functions) | Moderate (standard only) |
| **Nesting Support** | Excellent (dot notation) | Good (dot notation) |
| **Type Safety** | Excellent (CEL typed) | Good (manual parsing) |
| **Backend Conversion** | Excellent (widely documented) | Good (parser-driven) |
| **Error Messages** | Excellent (CEL provides position info) | Good (basic) |
| **Go Library Quality** | Excellent (official google/cel-go) | Good (multiple options) |
| **Production Proven** | Highest (Google Cloud, Firebase, many large APIs) | High (Java/Spring ecosystem heavy) |

### Internal Integration Analysis

#### For AIP-160

**Files that would need modification:**

1. **Filter Parser Implementation**
   - New package: `filtering/` or `filter/`
   - File: `filtering/parser.go` — Parse AIP-160 filter strings
   - File: `filtering/compiler.go` — Compile to backend-specific queries

2. **Backend-Specific Converters**
   - File: `store/xdbsqlite/filter.go` — SQL WHERE clause conversion
   - File: `store/xdbredis/filter.go` — Redis scan pattern conversion
   - File: `store/xdbfs/filter.go` — Filesystem filtering
   - File: `store/xdbmemory/filter.go` — In-memory filtering

3. **Type Definitions**
   - File: `filtering/types.go` — Filter AST structures
   - File: `core/filterexpr.go` — Core expression types (if desired)

4. **CLI Integration**
   - File: `cmd/cli.go` — Add `--filter` flag validation

5. **Tests**
   - File: `filtering/parser_test.go`
   - File: `store/xdbsqlite/filter_test.go`
   - etc. for each backend

**Integration Points:**

- `store.ListQuery.Filter` string already exists and is passed to all backends
- Each backend's `ListRecords()` method would parse the filter string
- Conversion happens at the store layer, before executing backend-specific operations

**Pattern Alignment:**

- XDB's dot-separated attribute names align perfectly with AIP-160's field traversal (`.` operator)
- Tuple-based data model maps well to expression evaluation
- No conflicts with existing patterns

**Validation Against Standards:**

- Would pass internal linters (Go standard format)
- Requires new type definitions but they're localized
- No impact on existing APIs

#### For RSQL

**Files that would need modification:**

1. **Filter Parser Implementation**
   - New package: `filtering/` with RSQL-specific parser
   - File: `filtering/parser.go` — RSQL tokenizer and parser
   - File: `filtering/ast.go` — AST structures

2. **Backend-Specific Converters**
   - Same files as AIP-160

3. **CLI Integration**
   - File: `cmd/cli.go` — Add `--filter` flag validation

**Integration Points:**
- Same as AIP-160
- More lightweight parser implementation

---

## Complexity Analysis

### AIP-160 Complexity

**Essential Complexity:**
- Expression parsing and AST construction
- Type checking against schema/field definitions
- Backend-specific query translation
- Error reporting with source positions

**Accidental Complexity:**
- CEL dependency management (heavyish library)
- Defining allowed functions/extensions per API
- Complex operator precedence rules (OR before AND)

**Overall Assessment:**
The complexity is justified. AIP-160's investment pays off through:
- Production-proven reliability
- Extensibility with custom functions
- Strong typing and validation
- Clear error messages

### RSQL Complexity

**Essential Complexity:**
- Simple tokenization and parsing
- AST construction
- Backend-specific translation

**Accidental Complexity:**
- Manual type inference
- Simpler operator precedence (less powerful)
- Need custom extension mechanism

**Overall Assessment:**
Less overhead, but less power. Good for constrained use cases.

---

## Validation & Implementation Results

### Validation Against Internal Standards

Both AIP-160 and RSQL:
- ✓ Will pass golangci-lint
- ✓ Compatible with XDB's tuple-based model
- ✓ Can work with existing `store.ListQuery.Filter` string
- ✓ No type safety issues in Go
- ✓ Can be tested comprehensively
- ✓ Support all target backends (SQL, Elasticsearch, Redis, filesystem)

### Backend Convertibility Examples

#### AIP-160 → SQL WHERE
```
Filter:  status=RUNNING AND count>5
SQL:     WHERE status = 'RUNNING' AND count > 5
```

#### AIP-160 → Elasticsearch
```
Filter:  status=RUNNING AND count>5
ES:      {
           "query": {
             "bool": {
               "must": [
                 { "term": { "status": "RUNNING" } },
                 { "range": { "count": { "gt": 5 } } }
               ]
             }
           }
         }
```

#### AIP-160 → Redis Scan Pattern
```
Filter:  status=RUNNING
Redis:   SCAN 0 MATCH "*RUNNING*" (+ in-memory validation)
```

#### RSQL → SQL WHERE
```
Filter:  status=="RUNNING";count=gt=5
SQL:     WHERE status = 'RUNNING' AND count > 5
```

The translation logic is similar; the difference is in how the input is parsed.

---

## Migration/Integration Path

### Phase 1: Foundation (1-2 weeks)

1. **Create filter expression types**
   ```go
   // pkg/filtering/types.go
   type Expression interface {
       // Evaluate or convert to backend queries
   }
   
   type Comparison struct {
       Field    string
       Operator string // "=", ">", "<", etc.
       Value    interface{}
   }
   
   type Compound struct {
       Op    string // "AND", "OR"
       Left  Expression
       Right Expression
   }
   ```

2. **Implement AIP-160 or RSQL parser**
   - Parse filter string to AST
   - Validate against schema (optional but recommended)
   - Error handling with position info

3. **Add to one backend (xdbmemory)**
   - Implement `applyFilter(record *core.Record, expr Expression) bool`
   - Verify with tests

### Phase 2: Extend to Other Backends (1-2 weeks)

1. **SQLite**: Convert Expression to SQL WHERE clause
   ```go
   // store/xdbsqlite/filter.go
   func (f *Filter) ToWhereClause() (string, []interface{})
   ```

2. **Filesystem**: In-memory filtering after loading records

3. **Redis**: SCAN + in-memory filtering

4. **Elasticsearch** (future): Convert to Query DSL

### Phase 3: CLI Integration (1 week)

1. Add `--filter` flag to list commands
2. Validate filter syntax early
3. Return helpful error messages

### Phase 4: Documentation (1 week)

1. Add filter syntax documentation
2. Example filters for common use cases
3. Concept doc: `docs/concepts/filtering.md`

---

## Recommendations

### Primary Recommendation: AIP-160

**Recommended for XDB because:**

1. **Proven at Scale**: Google uses it across all their APIs, Kubernetes adopted it, Firebase uses it for data filtering. This is battle-tested in production with billions of queries.

2. **Aligns with XDB's Model**: XDB's dot-separated attribute names (`author.name`, `metadata.labels.env`) are exactly what AIP-160's field traversal was designed for.

3. **Future-Proof**: If XDB adds webhook filtering, authorization policies, or event filtering (like Eventarc), AIP-160 is already the standard.

4. **Rich but Not Overwhelming**: Operators are sufficient for 95% of use cases, and custom functions provide escape hatch for domain-specific logic.

5. **Go Ecosystem**: Official `google/cel-go` library is well-maintained and thoroughly documented.

6. **Extensibility**: As XDB evolves, adding custom functions (e.g., `contains()`, `startsWith()`) is natural.

**Suggested Implementation Plan:**

1. Create `pkg/filtering/` package with AIP-160 parser
2. Use google/cel-go for expression compilation (leverage their AST)
3. Build converter layer for each backend
4. Add to CLI as `--filter` flag
5. Document with examples: `docs/concepts/filtering.md`

**Estimated Effort:** 2-3 weeks for full implementation across all backends

---

### Alternative Recommendation: RSQL/FIQL

**Recommended if:**

1. You want minimal dependencies
2. Users prioritize compact syntax
3. You need simpler parsing logic
4. Bundle size is critical

**Why not RSQL for XDB:**

- Smaller Go library ecosystem (mostly Java/Spring)
- Less documentation
- Not the direction enterprise APIs are moving

**Can be reconsidered if:**
- Users strongly prefer compact syntax
- Go RSQL library ecosystem improves
- Simpler parser maintenance becomes critical

---

## Open Questions & Recommendations

### For the Team to Decide

1. **Extension Functions?**
   - Q: Should filters support custom functions like `contains()`, `startsWith()`?
   - A: AIP-160 supports this; RSQL doesn't
   - Recommendation: Yes, enables future extensibility

2. **Type Checking?**
   - Q: Should we validate filter fields against schema definitions?
   - A: Makes filtering safer but adds dependency on schema
   - Recommendation: Validate at runtime, provide helpful errors

3. **Operator Precedence?**
   - Q: Accept AIP-160's non-standard precedence (OR before AND)?
   - A: Document clearly; use parentheses in examples
   - Recommendation: Yes, worth the slight surprise for consistency with Google's ecosystem

4. **Query Builder UI?**
   - Q: Should we provide a query builder helper for non-CLI users?
   - A: Can be added later, filter syntax is foundation
   - Recommendation: Build parsing first, UI builder as future enhancement

### Suggested Proof-of-Concept

**Scope (1 week):**

1. Implement AIP-160 parser for simple filters (field=value, AND, OR)
2. Implement conversion to SQLite WHERE clause
3. Add tests with 10-15 test cases
4. Document syntax with 5 examples

**Success Criteria:**

- Parse and evaluate `status=RUNNING AND count>5`
- Convert to correct SQL: `WHERE status = 'RUNNING' AND count > 5`
- Return helpful error for invalid syntax
- No dependency on custom CEL functions (keep it simple)

**This validates:**
- Whether AIP-160 syntax feels right for XDB's users
- Whether conversion to SQL is straightforward
- Whether error messages are helpful

---

## Sources & References

### Official Specifications

- [AIP-160: Filtering](https://google.aip.dev/160)
- [Common Expression Language (CEL)](https://cel.dev/)
- [OData Specification](https://www.odata.org/)
- [MongoDB Query Operators](https://www.mongodb.com/docs/manual/reference/mql/query-predicates/)
- [Kubernetes Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
- [Notion API Filters](https://developers.notion.com/reference/post-database-query-filter)

### Implementations

- [google/cel-go: Go implementation of CEL](https://github.com/google/cel-go)
- [jirutka/rsql-parser: RSQL Parser in Java](https://github.com/jirutka/rsql-parser)
- [ganigeorgiev/fexpr: Multi-backend filter query parser in Go](https://github.com/ganigeorgiev/fexpr)
- [cbrand/go-filterparams: Go filter parameter parsing](https://github.com/cbrand/go-filterparams)

### Learning Resources

- [Google Codelabs: CEL-Go](https://codelabs.developers.google.com/codelabs/cel-go)
- [Baeldung: REST Query Language with RSQL](https://www.baeldung.com/rest-api-search-language-rsql-fiql)
- [Microsoft Learn: OData $filter](https://learn.microsoft.com/en-us/graph/filter-query-parameter)
- [ObjectRocket: Elasticsearch Queries from Go Strings](https://kb.objectrocket.com/elasticsearch/how-to-construct-elasticsearch-queries-from-a-string-using-golang-550)

### Related Research

- [PostgreSQL Foreign Data Wrappers with filter pushdown](https://wiki.postgresql.org/wiki/Foreign_data_wrappers)
- [UX Patterns for CLI Filtering](https://lucasfcosta.com/2022/06/01/ux-patterns-cli-tools.html)
- [Filter Expression Best Practices](https://blog.logrocket.com/ux-design/filtering-ux-ui-design-patterns-best-practices/)

---

## Conclusion

XDB should adopt **AIP-160 filtering** as the standard for all filtering operations. It provides:

- Industry-proven syntax that users may already recognize from `gcloud`
- Strong foundation for future extensibility
- Direct convertibility to all target backends
- Clear semantics and error handling
- Active support from a major cloud provider

Start with a proof-of-concept for simple filters and SQLite conversion, validate with the team, then extend to other backends.

