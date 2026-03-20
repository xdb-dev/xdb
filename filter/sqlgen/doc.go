// Package sqlgen converts compiled CEL filter expressions into parameterized
// SQL WHERE clauses.
//
// It supports two table strategies:
//
//   - [ColumnStrategy]: standard column-per-field tables where each attribute
//     maps directly to a column name.
//   - [KVStrategy]: entity-attribute-value tables with (_id, _attr, _type, _val)
//     rows, where each filter condition becomes a subquery.
//
// All generated SQL uses ? placeholders to prevent SQL injection.
package sqlgen
