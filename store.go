package xdb

import "context"

// TupleStore defines the interface for storing, retrieving and deleting
// tuples.
type TupleStore interface {
	PutTuples(ctx context.Context, tuples ...*Tuple) error
	GetTuples(ctx context.Context, refs ...*Ref) ([]*Tuple, error)
	DeleteTuples(ctx context.Context, refs ...*Ref) error
}
