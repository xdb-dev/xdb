package types

import "fmt"

// Edge is an unidirectional relationship between two keys.
//
// Edge is an immutable data structure containing:
// - Source: The source key.
// - Name: The name of the relationship.
// - Target: The target key.
type Edge struct {
	source *Key
	name   string
	target *Key
}

// NewEdge creates a new Edge.
func NewEdge(source *Key, name string, target *Key) *Edge {
	return &Edge{source: source, name: name, target: target}
}

// Key returns a reference to the edge.
func (e *Edge) Key() *Key {
	return NewKey(e.source.Kind(), e.source.ID(), e.name, e.target.Kind(), e.target.ID())
}

// Source returns the source key of the edge.
func (e *Edge) Source() *Key {
	return e.source
}

// Name returns the name of the edge.
func (e *Edge) Name() string {
	return e.name
}

// Target returns the target key of the edge.
func (e *Edge) Target() *Key {
	return e.target
}

// String returns the string representation of the edge.
func (e *Edge) String() string {
	return fmt.Sprintf("Edge(%s, %s, %s)", e.source, e.name, e.target)
}
