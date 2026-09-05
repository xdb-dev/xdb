package store

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

// Op is the closed set of write intents a [Mutation] can carry, modeled
// on HTTP: patch, create, put, delete. Exists-semantics are data the
// driver receives, not code the driver invents — a driver cannot get an
// op's create-or-replace behavior subtly wrong because the distinction
// arrives in the mutation.
type Op int

const (
	// OpPatch puts facts; everything else at the path is untouched
	// (HTTP PATCH). A record springs into existence when its first
	// tuples are patched — no exists-check.
	OpPatch Op = iota

	// OpCreate replaces; fails with [core.ErrAlreadyExists] if the
	// path has any tuples (HTTP POST / PUT-If-None-Match). MUST be
	// atomic (INSERT, O_EXCL, EXISTS-gated write) — this is the one op
	// where check-then-write loses data.
	OpCreate

	// OpPut replaces unconditionally (upsert, HTTP PUT): the
	// mutation's tuples become the record's full tuple set; absent
	// attrs are dropped.
	OpPut

	// OpDelete removes facts; idempotent (HTTP DELETE). Attrs names
	// the tuples to remove; empty Attrs removes the whole record.
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
