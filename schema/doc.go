// Package schema defines schema definition types and validation for XDB.
//
// A schema describes the expected structure of records within a namespace.
// A schema operates in one of three modes. Declared fields type-check in
// every mode. The mode controls only undeclared attributes:
//
//   - [ModeStrict] rejects undeclared attributes. It is the default.
//   - [ModeFlexible] accepts undeclared attributes as-is.
//   - [ModeDynamic] infers a field for each undeclared attribute and adds
//     it to the schema.
package schema
