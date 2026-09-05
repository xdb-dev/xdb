// Package xdbmemory provides an in-memory implementation of [store.Driver].
//
// This is the reference driver. It is used for tests, for embedded
// mode, and to validate the driver contract. All state is held in maps
// that a single [sync.RWMutex] protects. Construct a usable store with
// store.New(xdbmemory.NewDriver()).
package xdbmemory

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// Driver is an in-memory implementation of [store.Driver] with native
// transactions ([store.TxDriver]).
type Driver struct {
	tuples map[string]map[string]*core.Tuple // record path → attr → tuple
	defs   map[string]*schema.Def            // schema path → def
	mu     sync.RWMutex
}

// NewDriver creates a new in-memory driver.
func NewDriver() *Driver {
	return &Driver{
		tuples: make(map[string]map[string]*core.Tuple),
		defs:   make(map[string]*schema.Def),
	}
}

// Close is a no-op for the in-memory driver.
func (d *Driver) Close() error { return nil }

// Health always returns nil — the in-memory driver is always healthy.
func (d *Driver) Health(_ context.Context) error { return nil }

// --- Tuple reads ---

// GetTuples returns the tuples at the given attr-level URIs.
// Absent attrs are omitted.
func (d *Driver) GetTuples(
	_ context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return getTuples(d.tuples, uris)
}

// ScanTuples yields every tuple under scope, ordered by record path
// then attr. Tuples of one record are contiguous.
func (d *Driver) ScanTuples(
	_ context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	d.mu.RLock()
	snapshot := collectTuples(d.tuples, scope)
	d.mu.RUnlock()

	return yieldTuples(snapshot)
}

// --- Tuple writes ---

// Apply executes one mutation atomically under the write lock.
func (d *Driver) Apply(_ context.Context, m store.Mutation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return applyMutation(d.tuples, m)
}

// --- Definition reads ---

// GetSchema retrieves a definition by URI. Returns [core.ErrNotFound] if absent.
func (d *Driver) GetSchema(_ context.Context, uri *core.URI) (*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return getDef(d.defs, uri)
}

// ScanSchemas yields definitions under scope, ordered by schema path.
func (d *Driver) ScanSchemas(
	_ context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	d.mu.RLock()
	snapshot := collectDefs(d.defs, scope)
	d.mu.RUnlock()

	return yieldDefs(snapshot)
}

// --- Definition writes ---

// CreateSchema stores a new definition. Returns [core.ErrAlreadyExists]
// if one exists.
func (d *Driver) CreateSchema(_ context.Context, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := def.URI.Path()
	if _, exists := d.defs[key]; exists {
		return core.ErrAlreadyExists
	}
	d.defs[key] = def
	return nil
}

// PutSchema stores a definition unconditionally.
func (d *Driver) PutSchema(_ context.Context, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.defs[def.URI.Path()] = def
	return nil
}

// DeleteSchema deletes a definition. Returns [core.ErrNotFound] if absent.
func (d *Driver) DeleteSchema(_ context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := uri.Path()
	if _, exists := d.defs[key]; !exists {
		return core.ErrNotFound
	}
	delete(d.defs, key)
	return nil
}

// DropRecords deletes all record tuples belonging to a schema.
func (d *Driver) DropRecords(_ context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	deleteSchemaRecords(d.tuples, uri)
	return nil
}

// --- TxDriver ---

// Tx executes fn against an unlocked view of the driver while holding
// the write lock. Each touched key's prior value is journaled before
// mutation; on error the journal is replayed to roll back — O(touched),
// not a whole-map clone.
func (d *Driver) Tx(_ context.Context, fn func(tx store.Driver) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx := &txDriver{
		d:          d,
		undoTuples: make(map[string]map[string]*core.Tuple),
		undoDefs:   make(map[string]*schema.Def),
	}
	if err := fn(tx); err != nil {
		tx.rollback()
		return err
	}
	return nil
}

// txDriver is an unlocked view of Driver used within Tx. The parent's
// mutex is already held. Writes journal each touched key's prior value
// before mutating; reads delegate to the shared lock-free helpers.
type txDriver struct {
	d          *Driver
	undoTuples map[string]map[string]*core.Tuple // key → prior attrs (nil = absent)
	undoDefs   map[string]*schema.Def            // key → prior def (nil = absent)
}

// snapTuple journals a record key's prior attr map before it is
// mutated, once per key. The map is cloned because patch and delete
// mutate it in place.
func (tx *txDriver) snapTuple(key string) {
	if _, seen := tx.undoTuples[key]; seen {
		return
	}
	tx.undoTuples[key] = maps.Clone(tx.d.tuples[key])
}

// snapDef journals a schema key's prior def before it is written, once
// per key. Defs are immutable values, so no clone is needed.
func (tx *txDriver) snapDef(key string) {
	if _, seen := tx.undoDefs[key]; seen {
		return
	}
	tx.undoDefs[key] = tx.d.defs[key]
}

// rollback restores every journaled key to its pre-tx value.
func (tx *txDriver) rollback() {
	for key, prior := range tx.undoTuples {
		if prior == nil {
			delete(tx.d.tuples, key)
		} else {
			tx.d.tuples[key] = prior
		}
	}
	for key, prior := range tx.undoDefs {
		if prior == nil {
			delete(tx.d.defs, key)
		} else {
			tx.d.defs[key] = prior
		}
	}
}

func (tx *txDriver) GetTuples(
	_ context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	return getTuples(tx.d.tuples, uris)
}

func (tx *txDriver) ScanTuples(
	_ context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	return yieldTuples(collectTuples(tx.d.tuples, scope))
}

func (tx *txDriver) Apply(_ context.Context, m store.Mutation) error {
	tx.snapTuple(recordKey(m.Path))
	return applyMutation(tx.d.tuples, m)
}

func (tx *txDriver) GetSchema(_ context.Context, uri *core.URI) (*schema.Def, error) {
	return getDef(tx.d.defs, uri)
}

func (tx *txDriver) ScanSchemas(
	_ context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	return yieldDefs(collectDefs(tx.d.defs, scope))
}

func (tx *txDriver) CreateSchema(_ context.Context, def *schema.Def) error {
	key := def.URI.Path()
	tx.snapDef(key)
	if _, exists := tx.d.defs[key]; exists {
		return core.ErrAlreadyExists
	}
	tx.d.defs[key] = def
	return nil
}

func (tx *txDriver) PutSchema(_ context.Context, def *schema.Def) error {
	key := def.URI.Path()
	tx.snapDef(key)
	tx.d.defs[key] = def
	return nil
}

func (tx *txDriver) DeleteSchema(_ context.Context, uri *core.URI) error {
	key := uri.Path()
	tx.snapDef(key)
	if _, exists := tx.d.defs[key]; !exists {
		return core.ErrNotFound
	}
	delete(tx.d.defs, key)
	return nil
}

func (tx *txDriver) DropRecords(_ context.Context, uri *core.URI) error {
	prefix := uri.NS() + "/" + uri.Schema() + "/"
	for key := range tx.d.tuples {
		if strings.HasPrefix(key, prefix) {
			tx.snapTuple(key)
		}
	}
	deleteSchemaRecords(tx.d.tuples, uri)
	return nil
}

// --- Lock-free helpers ---
// These contain the actual logic. Both Driver (under lock) and
// txDriver (parent lock already held) delegate to these.

// recordKey returns the record-path map key for a URI, ignoring any
// attr component.
func recordKey(uri *core.URI) string {
	return uri.RecordPath()
}

func getTuples(
	tuples map[string]map[string]*core.Tuple,
	uris []*core.URI,
) ([]*core.Tuple, error) {
	var got []*core.Tuple
	for _, uri := range uris {
		attrs, ok := tuples[recordKey(uri)]
		if !ok {
			continue
		}
		if tuple, ok := attrs[uri.Attr()]; ok {
			got = append(got, tuple)
		}
	}
	return got, nil
}

// collectTuples snapshots all tuples under scope, ordered by record
// path then attr, so scans are deterministic and per-record contiguous.
func collectTuples(
	tuples map[string]map[string]*core.Tuple,
	scope *core.URI,
) []*core.Tuple {
	// Record-scoped scans address the single key directly, avoiding a
	// full-map walk (mirrors the fs/redis/sqlite drivers).
	if scope != nil && scope.ID() != "" {
		attrs := tuples[scope.RecordPath()]
		snapshot := make([]*core.Tuple, 0, len(attrs))
		for _, attr := range slices.Sorted(maps.Keys(attrs)) {
			snapshot = append(snapshot, attrs[attr])
		}
		return snapshot
	}

	keys := make([]string, 0, len(tuples))
	for key, attrs := range tuples {
		if !matchScope(anyTuplePath(attrs), scope) {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var snapshot []*core.Tuple
	for _, key := range keys {
		attrs := tuples[key]
		for _, attr := range slices.Sorted(maps.Keys(attrs)) {
			snapshot = append(snapshot, attrs[attr])
		}
	}
	return snapshot
}

// anyTuplePath returns the record path URI of any tuple in the attr
// map, or nil when the map is empty.
func anyTuplePath(attrs map[string]*core.Tuple) *core.URI {
	for _, tuple := range attrs {
		return tuple.Path()
	}
	return nil
}

// matchScope reports whether the record path URI falls under scope.
// A nil scope matches everything; empty scope components are wildcards.
func matchScope(path, scope *core.URI) bool {
	if path == nil {
		return false
	}
	if scope == nil {
		return true
	}
	if scope.NS() != "" && path.NS() != scope.NS() {
		return false
	}
	if scope.Schema() != "" && path.Schema() != scope.Schema() {
		return false
	}
	if scope.ID() != "" && path.ID() != scope.ID() {
		return false
	}
	return true
}

func yieldTuples(snapshot []*core.Tuple) iter.Seq2[*core.Tuple, error] {
	return func(yield func(*core.Tuple, error) bool) {
		for _, tuple := range snapshot {
			if !yield(tuple, nil) {
				return
			}
		}
	}
}

func yieldDefs(snapshot []*schema.Def) iter.Seq2[*schema.Def, error] {
	return func(yield func(*schema.Def, error) bool) {
		for _, def := range snapshot {
			if !yield(def, nil) {
				return
			}
		}
	}
}

func applyMutation(
	tuples map[string]map[string]*core.Tuple,
	m store.Mutation,
) error {
	key := recordKey(m.Path)

	switch m.Op {
	case store.OpPatch:
		// A patch that carries no tuples changes nothing. Returning
		// early keeps it from creating an empty attr map.
		if len(m.Tuples) == 0 {
			return nil
		}
		attrs, ok := tuples[key]
		if !ok {
			attrs = make(map[string]*core.Tuple, len(m.Tuples))
			tuples[key] = attrs
		}
		for _, tuple := range m.Tuples {
			attrs[tuple.Attr()] = tuple
		}

	case store.OpCreate:
		if len(tuples[key]) > 0 {
			return core.ErrAlreadyExists
		}
		if len(m.Tuples) == 0 {
			delete(tuples, key)
			return nil
		}
		tuples[key] = tupleSet(m.Tuples)

	case store.OpPut:
		// A record with no tuples does not exist, so an empty put
		// removes the key rather than leaving an empty attr map.
		if len(m.Tuples) == 0 {
			delete(tuples, key)
			return nil
		}
		tuples[key] = tupleSet(m.Tuples)

	case store.OpDelete:
		if len(m.Attrs) == 0 {
			delete(tuples, key)
			return nil
		}
		attrs := tuples[key]
		for _, attr := range m.Attrs {
			delete(attrs, attr)
		}
		if len(attrs) == 0 {
			delete(tuples, key)
		}

	default:
		return fmt.Errorf("xdbmemory: unknown op %s", m.Op)
	}

	return nil
}

func tupleSet(ts []*core.Tuple) map[string]*core.Tuple {
	attrs := make(map[string]*core.Tuple, len(ts))
	for _, tuple := range ts {
		attrs[tuple.Attr()] = tuple
	}
	return attrs
}

func getDef(defs map[string]*schema.Def, uri *core.URI) (*schema.Def, error) {
	def, ok := defs[uri.Path()]
	if !ok {
		return nil, core.ErrNotFound
	}
	return def, nil
}

// collectDefs snapshots all defs under scope, ordered by schema path.
func collectDefs(defs map[string]*schema.Def, scope *core.URI) []*schema.Def {
	keys := make([]string, 0, len(defs))
	for key, def := range defs {
		if scope != nil && def.URI.NS() != scope.NS() {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	snapshot := make([]*schema.Def, 0, len(keys))
	for _, key := range keys {
		snapshot = append(snapshot, defs[key])
	}
	return snapshot
}

func deleteSchemaRecords(
	tuples map[string]map[string]*core.Tuple,
	uri *core.URI,
) {
	prefix := uri.NS() + "/" + uri.Schema() + "/"
	for key := range tuples {
		if strings.HasPrefix(key, prefix) {
			delete(tuples, key)
		}
	}
}
