package xdbstruct

// NSGetter provides the namespace for a record during encoding.
// Structs implementing this interface can override the default namespace
// specified in Options.
type NSGetter interface {
	GetNS() string
}

// NSSetter receives the namespace from a record during decoding.
// The decoder will call SetNS after populating struct fields.
type NSSetter interface {
	SetNS(ns string)
}

// SchemaGetter provides the schema for a record during encoding.
// Structs implementing this interface can override the default schema
// specified in Options.
type SchemaGetter interface {
	GetSchema() string
}

// SchemaSetter receives the schema from a record during decoding.
// The decoder will call SetSchema after populating struct fields.
type SchemaSetter interface {
	SetSchema(schema string)
}

// IDGetter provides the ID for a record during encoding.
// This takes precedence over the primary_key tag.
type IDGetter interface {
	GetID() string
}

// IDSetter receives the ID from a record during decoding.
// The decoder will call SetID after populating struct fields.
type IDSetter interface {
	SetID(id string)
}
