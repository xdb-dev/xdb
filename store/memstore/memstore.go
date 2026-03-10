// Package memstore provides an in-memory implementation of [store.Store].
//
// This is the reference implementation used for testing, embedded mode,
// and validating the store interface design. All state is held in maps
// protected by a single [sync.RWMutex].
package memstore

import (
	"context"
	"maps"
	"sort"
	"sync"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// Store is an in-memory implementation of [store.Store].
type Store struct {
	mu      sync.RWMutex
	records map[string]*core.Record
	schemas map[string]*schema.Def
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{
		records: make(map[string]*core.Record),
		schemas: make(map[string]*schema.Def),
	}
}

// Health always returns nil — the in-memory store is always healthy.
func (s *Store) Health(_ context.Context) error {
	return nil
}

// --- Records ---

// GetRecord retrieves a record by URI.
func (s *Store) GetRecord(_ context.Context, uri *core.URI) (*core.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getRecord(s.records, uri)
}

// ListRecords lists records scoped by the given URI.
func (s *Store) ListRecords(
	_ context.Context,
	uri *core.URI,
	q *store.ListQuery,
) (*store.Page[*core.Record], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listRecords(s.records, uri, q), nil
}

// CreateRecord creates a new record. Returns [store.ErrAlreadyExists] if it exists.
func (s *Store) CreateRecord(_ context.Context, record *core.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return createRecord(s.records, record)
}

// UpdateRecord updates an existing record. Returns [store.ErrNotFound] if missing.
func (s *Store) UpdateRecord(_ context.Context, record *core.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return updateRecord(s.records, record)
}

// UpsertRecord creates or updates a record unconditionally.
func (s *Store) UpsertRecord(_ context.Context, record *core.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.URI().Path()] = record
	return nil
}

// DeleteRecord deletes a record by URI. Returns [store.ErrNotFound] if missing.
func (s *Store) DeleteRecord(_ context.Context, uri *core.URI) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return deleteFromMap(s.records, uri.Path())
}

// --- Schemas ---

// GetSchema retrieves a schema definition by URI.
func (s *Store) GetSchema(_ context.Context, uri *core.URI) (*schema.Def, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getSchema(s.schemas, uri)
}

// ListSchemas lists schemas, optionally scoped by namespace URI.
func (s *Store) ListSchemas(
	_ context.Context,
	uri *core.URI,
	q *store.ListQuery,
) (*store.Page[*schema.Def], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listSchemas(s.schemas, uri, q), nil
}

// CreateSchema creates a new schema definition. Returns [store.ErrAlreadyExists] if it exists.
func (s *Store) CreateSchema(
	_ context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return createInMap(s.schemas, uri.Path(), def)
}

// UpdateSchema updates an existing schema definition. Returns [store.ErrNotFound] if missing.
func (s *Store) UpdateSchema(
	_ context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return updateInMap(s.schemas, uri.Path(), def)
}

// DeleteSchema deletes a schema by URI. Returns [store.ErrNotFound] if missing.
func (s *Store) DeleteSchema(_ context.Context, uri *core.URI) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return deleteFromMap(s.schemas, uri.Path())
}

// --- Namespaces ---

// GetNamespace checks if any schema exists in the given namespace.
func (s *Store) GetNamespace(_ context.Context, uri *core.URI) (*core.NS, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getNamespace(s.schemas, uri)
}

// ListNamespaces lists unique namespaces derived from schemas.
func (s *Store) ListNamespaces(
	_ context.Context,
	q *store.ListQuery,
) (*store.Page[*core.NS], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listNamespaces(s.schemas, q), nil
}

// --- Batch ---

// ExecuteBatch runs fn within a transaction. On error, all changes are rolled back.
func (s *Store) ExecuteBatch(
	ctx context.Context,
	fn func(tx store.Store) error,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Snapshot current state.
	recordSnap := make(map[string]*core.Record, len(s.records))
	maps.Copy(recordSnap, s.records)
	schemaSnap := make(map[string]*schema.Def, len(s.schemas))
	maps.Copy(schemaSnap, s.schemas)

	// Create an unlocked view that writes directly to our maps.
	tx := &txStore{store: s}

	if err := fn(tx); err != nil {
		// Rollback.
		s.records = recordSnap
		s.schemas = schemaSnap
		return err
	}

	return nil
}

// txStore is an unlocked view of Store used within ExecuteBatch.
// The parent Store's mutex is already held. All methods delegate
// to the shared lock-free helpers.
type txStore struct {
	store *Store
}

func (tx *txStore) GetRecord(_ context.Context, uri *core.URI) (*core.Record, error) {
	return getRecord(tx.store.records, uri)
}

func (tx *txStore) ListRecords(
	_ context.Context,
	uri *core.URI,
	q *store.ListQuery,
) (*store.Page[*core.Record], error) {
	return listRecords(tx.store.records, uri, q), nil
}

func (tx *txStore) CreateRecord(_ context.Context, record *core.Record) error {
	return createRecord(tx.store.records, record)
}

func (tx *txStore) UpdateRecord(_ context.Context, record *core.Record) error {
	return updateRecord(tx.store.records, record)
}

func (tx *txStore) UpsertRecord(_ context.Context, record *core.Record) error {
	tx.store.records[record.URI().Path()] = record
	return nil
}

func (tx *txStore) DeleteRecord(_ context.Context, uri *core.URI) error {
	return deleteFromMap(tx.store.records, uri.Path())
}

func (tx *txStore) GetSchema(_ context.Context, uri *core.URI) (*schema.Def, error) {
	return getSchema(tx.store.schemas, uri)
}

func (tx *txStore) ListSchemas(
	_ context.Context,
	uri *core.URI,
	q *store.ListQuery,
) (*store.Page[*schema.Def], error) {
	return listSchemas(tx.store.schemas, uri, q), nil
}

func (tx *txStore) CreateSchema(_ context.Context, uri *core.URI, def *schema.Def) error {
	return createInMap(tx.store.schemas, uri.Path(), def)
}

func (tx *txStore) UpdateSchema(_ context.Context, uri *core.URI, def *schema.Def) error {
	return updateInMap(tx.store.schemas, uri.Path(), def)
}

func (tx *txStore) DeleteSchema(_ context.Context, uri *core.URI) error {
	return deleteFromMap(tx.store.schemas, uri.Path())
}

func (tx *txStore) GetNamespace(_ context.Context, uri *core.URI) (*core.NS, error) {
	return getNamespace(tx.store.schemas, uri)
}

func (tx *txStore) ListNamespaces(
	_ context.Context,
	q *store.ListQuery,
) (*store.Page[*core.NS], error) {
	return listNamespaces(tx.store.schemas, q), nil
}

// --- Lock-free helpers ---
// These contain the actual logic. Both Store (under lock) and txStore
// (parent lock already held) delegate to these.

func getRecord(
	records map[string]*core.Record,
	uri *core.URI,
) (*core.Record, error) {
	r, ok := records[uri.Path()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return r, nil
}

func listRecords(
	records map[string]*core.Record,
	uri *core.URI,
	q *store.ListQuery,
) *store.Page[*core.Record] {
	ns := uri.NS()
	schemaScope := uri.Schema()

	var matched []*core.Record
	for _, r := range records {
		if !r.NS().Equals(ns) {
			continue
		}
		if schemaScope != nil && !r.Schema().Equals(schemaScope) {
			continue
		}
		matched = append(matched, r)
	}

	return sortAndPaginate(matched, func(r *core.Record) string {
		return r.URI().Path()
	}, q)
}

func createRecord(records map[string]*core.Record, record *core.Record) error {
	key := record.URI().Path()
	if _, exists := records[key]; exists {
		return store.ErrAlreadyExists
	}
	records[key] = record
	return nil
}

func updateRecord(records map[string]*core.Record, record *core.Record) error {
	key := record.URI().Path()
	if _, exists := records[key]; !exists {
		return store.ErrNotFound
	}
	records[key] = record
	return nil
}

func getSchema(
	schemas map[string]*schema.Def,
	uri *core.URI,
) (*schema.Def, error) {
	def, ok := schemas[uri.Path()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return def, nil
}

func listSchemas(
	schemas map[string]*schema.Def,
	uri *core.URI,
	q *store.ListQuery,
) *store.Page[*schema.Def] {
	var matched []*schema.Def
	for _, def := range schemas {
		if uri != nil && !def.URI.NS().Equals(uri.NS()) {
			continue
		}
		matched = append(matched, def)
	}

	return sortAndPaginate(matched, func(d *schema.Def) string {
		return d.URI.Path()
	}, q)
}

func getNamespace(
	schemas map[string]*schema.Def,
	uri *core.URI,
) (*core.NS, error) {
	ns := uri.NS()
	for _, def := range schemas {
		if def.URI.NS().Equals(ns) {
			return ns, nil
		}
	}
	return nil, store.ErrNotFound
}

func listNamespaces(
	schemas map[string]*schema.Def,
	q *store.ListQuery,
) *store.Page[*core.NS] {
	seen := make(map[string]*core.NS)
	for _, def := range schemas {
		ns := def.URI.NS()
		seen[ns.String()] = ns
	}

	items := make([]*core.NS, 0, len(seen))
	for _, ns := range seen {
		items = append(items, ns)
	}

	return sortAndPaginate(items, func(ns *core.NS) string {
		return ns.String()
	}, q)
}

// --- Generic map helpers ---

func createInMap[T any](m map[string]T, key string, val T) error {
	if _, exists := m[key]; exists {
		return store.ErrAlreadyExists
	}
	m[key] = val
	return nil
}

func updateInMap[T any](m map[string]T, key string, val T) error {
	if _, exists := m[key]; !exists {
		return store.ErrNotFound
	}
	m[key] = val
	return nil
}

func deleteFromMap[T any](m map[string]T, key string) error {
	if _, exists := m[key]; !exists {
		return store.ErrNotFound
	}
	delete(m, key)
	return nil
}

// --- Pagination ---

// sortAndPaginate extracts sort keys once (O(N)), sorts, then paginates.
func sortAndPaginate[T any](
	items []T,
	keyFn func(T) string,
	q *store.ListQuery,
) *store.Page[T] {
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = keyFn(item)
	}

	sort.Sort(keySorter[T]{items: items, keys: keys})

	return paginate(items, q)
}

// paginate applies offset/limit to a sorted slice and returns a [store.Page].
func paginate[T any](items []T, q *store.ListQuery) *store.Page[T] {
	total := len(items)

	offset := 0
	limit := total
	if q != nil {
		offset = q.Offset
		if q.Limit > 0 {
			limit = q.Limit
		}
	}

	offset = min(offset, total)
	end := min(offset+limit, total)

	var nextOffset int
	if end < total {
		nextOffset = end
	}

	return &store.Page[T]{
		Items:      items[offset:end],
		Total:      total,
		NextOffset: nextOffset,
	}
}

// keySorter sorts items by pre-extracted string keys.
type keySorter[T any] struct {
	items []T
	keys  []string
}

func (s keySorter[T]) Len() int           { return len(s.items) }
func (s keySorter[T]) Less(i, j int) bool { return s.keys[i] < s.keys[j] }
func (s keySorter[T]) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}
