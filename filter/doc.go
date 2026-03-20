// Package filter provides CEL-based parsing and evaluation of record filter expressions.
//
// Filter expressions use CEL (Common Expression Language) syntax:
//
//	name == "John"                          // equality
//	age > 30                                // comparison
//	status == "active" && age >= 18         // compound AND
//	name == "John" || name == "Jane"        // compound OR
//	!(age < 18)                             // negation
//	name.contains("oh")                     // substring containment
//	name.startsWith("J")                    // prefix match
//	name.endsWith("hn")                     // suffix match
//	size(name) > 3                          // string length
//	status in ["active", "pending"]         // list membership
//
// Use [Compile] to parse a filter string into a [Filter], and [Match] to evaluate
// a [Filter] against a [core.Record].
//
// When a [schema.Def] is provided to [Compile], fields are type-checked against
// the schema. When nil is passed (flexible mode), all variables are dynamically typed.
package filter
