package store

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

// Op specifies how a [Mutation] changes a record and whether the record
// must be absent. Drivers implement the semantics documented for each op.
type Op int

const (
	// OpPatch adds or replaces the supplied tuples, preserving other
	// attributes. It creates the record if absent.
	OpPatch Op = iota

	// OpCreate writes a new record or returns [core.ErrAlreadyExists]
	// if the path has tuples. The existence check and write must be atomic.
	OpCreate

	// OpPut replaces unconditionally (upsert, HTTP PUT): the
	// mutation's tuples become the record's full tuple set; absent
	// attrs are dropped.
	OpPut

	// OpDelete removes the tuples named by Attrs. Empty Attrs removes
	// the whole record. Missing records and attributes are ignored.
	OpDelete
)

// String returns the op name for logging and error messages.
func (op Op) String() string {
	switch op {
	case OpPatch:
		return "patch"
	case OpCreate:
		return "create"
	case OpPut:
		return "put"
	case OpDelete:
		return "delete"
	default:
		return fmt.Sprintf("op(%d)", int(op))
	}
}

// Mutation is one record-scoped write, applied atomically by
// [TupleWriter.Apply].
type Mutation struct {
	Path   *core.URI
	Tuples []*core.Tuple
	Attrs  []string
	Op     Op
}

// MutationError attributes a batch-write failure to the mutation that
// caused it. The facade applies mutations sequentially and stops at
// the first failure; mutations before Index remain applied (the
// facade provides whole-batch atomicity on [TxDriver]s). Drivers and
// middleware return bare errors — attribution happens above them.
type MutationError struct {
	Err   error
	Path  *core.URI
	Index int
}

// Error implements the error interface.
func (e *MutationError) Error() string {
	return fmt.Sprintf("store: mutation %d (%s): %v", e.Index, e.Path, e.Err)
}

// Unwrap returns the underlying error for [errors.Is] / [errors.As].
func (e *MutationError) Unwrap() error {
	return e.Err
}
