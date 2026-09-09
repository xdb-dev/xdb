// Package x holds the generic slice helpers the standard library does
// not: Map, and Index for building a lookup by key.
//
// Anything the standard library already covers belongs there instead.
// Prefer slices.Concat over a Join, and slices.DeleteFunc over a Filter.
// Keep this package small enough for a reader to hold all of it at once.
package x
