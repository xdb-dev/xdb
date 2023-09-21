package xdb

// Type defines the type of a value.
type Type byte

// Types of values supported by xdb.
const (
	TypeUnknown Type = iota
	TypeEmpty
	TypeString
	TypeInt
	TypeUint
	TypeFloat
	TypeBool
	TypeTime
	TypeBytes
	TypeBinary
	TypeJSON
	TypeList

	// xdb specific types
	TypeKey
	TypeAttr
	TypeEdge
)
