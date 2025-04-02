package types

// Tuple is the core data structure of XDB.
//
// Tuple is an immutable data structure containing:
// - Key: Unique identifier for the record.
// - Name: Name of the attribute.
// - Value: Value of the attribute.
// - Options: Options are key-value pairs
type Tuple struct {
	key   *Key
	value *Value
}

// NewTuple creates a new tuple with the given key, name, value, and options.
func NewTuple[T Type](key *Key, value T) *Tuple {
	return &Tuple{key: key, value: NewValue(value)}
}

// Key returns the key that uniquely identifies the tuple.
func (t *Tuple) Key() *Key {
	return t.key
}

// Kind returns the kind of the tuple.
func (t *Tuple) Kind() string {
	return t.key.Kind()
}

// ID returns the ID of the tuple.
func (t *Tuple) ID() string {
	return t.key.ID()
}

// Name returns the attribute name.
func (t *Tuple) Name() string {
	return t.key.Name()
}

// Value returns the value of the attribute.
func (t *Tuple) Value() *Value {
	return t.value
}
