package types

import (
	"time"
)

// TypeID represents the type of a value.
type TypeID int

const (
	UnknownType TypeID = iota
	StringType
	IntegerType
	FloatType
	BooleanType
	BytesType
	TimeType
	PointType
)

var typeNames = map[TypeID]string{
	UnknownType: "UNKNOWN",
	StringType:  "STRING",
	IntegerType: "INTEGER",
	FloatType:   "FLOAT",
	BooleanType: "BOOLEAN",
	BytesType:   "BYTES",
	TimeType:    "TIME",
	PointType:   "POINT",
}

// String returns the name of the type.
func (t TypeID) String() string {
	return typeNames[t]
}

type Type interface {
	Integer | Float | String | Boolean | Bytes | Hybrid
}

// Integer is an union of all possible integer types.
type Integer interface {
	int | int8 | int16 | int32 | int64 |
		[]int | []int8 | []int16 | []int32 | []int64
}

// Float is an union of all possible floating point types.
type Float interface {
	float32 | float64 |
		[]float32 | []float64
}

// String is an union of all possible string types.
type String interface {
	string | []string
}

// Boolean is an union of all possible boolean types.
type Boolean interface {
	bool | []bool
}

// Bytes is an union of all possible byte types.
type Bytes interface {
	[]byte | [][]byte
}

// Point is a point on the Earth's surface.
type Point struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

// Hybrid is all supported composite types.
type Hybrid interface {
	time.Time | []time.Time |
		Point | []Point
}
