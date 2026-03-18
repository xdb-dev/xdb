// Package filter provides parsing and evaluation of record filter expressions.
//
// Filter expressions use the syntax: ATTR OP VALUE
//
// Supported operators: =, !=, >, <, >=, <=, contains.
// Attribute paths may be dotted (e.g., "author.id").
// Values may be quoted to include spaces (e.g., "hello world").
//
// Use [Parse] to parse a filter string into an [Expr], and [Match] to evaluate
// an [Expr] against a [core.Record].
package filter
