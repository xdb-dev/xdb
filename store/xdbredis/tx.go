package xdbredis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// Run executes fn within a Redis MULTI/EXEC transaction.
//
// Reads inside fn see live data; writes are queued in a pipeline and
// applied atomically on commit. If fn returns an error, queued writes
// are discarded. Existence checks (Create/Update/Delete) read live state,
// so a concurrent writer between the check and EXEC may cause a queued
// command to misbehave at commit — single-writer use is recommended.
func (s *Store) Run(ctx context.Context, fn func(tx store.Store) error) error {
	pipe := s.client.TxPipeline()
	tv := &txStore{store: s, pipe: pipe}

	if err := fn(tv); err != nil {
		pipe.Discard()
		return err
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("xdbredis: commit tx: %w", err)
	}
	return nil
}

// txStore implements [store.Store] with writes deferred to a Redis pipeline.
type txStore struct {
	store *Store
	pipe  redis.Pipeliner
}

// --- Reads delegate to the live store ---

func (t *txStore) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	return t.store.GetRecord(ctx, uri)
}

func (t *txStore) ListRecords(ctx context.Context, q *store.Query) (*store.Page[*core.Record], error) {
	return t.store.ListRecords(ctx, q)
}

func (t *txStore) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	return t.store.GetSchema(ctx, uri)
}

func (t *txStore) ListSchemas(ctx context.Context, q *store.Query) (*store.Page[*schema.Def], error) {
	return t.store.ListSchemas(ctx, q)
}

func (t *txStore) GetNamespace(ctx context.Context, uri *core.URI) (string, error) {
	return t.store.GetNamespace(ctx, uri)
}

func (t *txStore) ListNamespaces(ctx context.Context, q *store.Query) (*store.Page[string], error) {
	return t.store.ListNamespaces(ctx, q)
}

func (t *txStore) Close() error { return nil }

// --- Writes queue into the pipeline ---

func (t *txStore) CreateRecord(ctx context.Context, record *core.Record) error {
	key := t.store.recordKey(record.URI())
	exists, err := t.store.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check record exists: %w", err)
	}
	if exists > 0 {
		return store.ErrAlreadyExists
	}
	if err := t.store.validateAndEvolve(ctx, record); err != nil {
		return err
	}
	return t.queueWriteRecord(ctx, record)
}

func (t *txStore) UpdateRecord(ctx context.Context, record *core.Record) error {
	key := t.store.recordKey(record.URI())
	exists, err := t.store.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check record exists: %w", err)
	}
	if exists == 0 {
		return store.ErrNotFound
	}
	if err := t.store.validateAndEvolve(ctx, record); err != nil {
		return err
	}
	return t.queueWriteRecord(ctx, record)
}

func (t *txStore) UpsertRecord(ctx context.Context, record *core.Record) error {
	if err := t.store.validateAndEvolve(ctx, record); err != nil {
		return err
	}
	return t.queueWriteRecord(ctx, record)
}

func (t *txStore) DeleteRecord(ctx context.Context, uri *core.URI) error {
	key := t.store.recordKey(uri)
	exists, err := t.store.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check record exists: %w", err)
	}
	if exists == 0 {
		return store.ErrNotFound
	}

	t.pipe.Del(ctx, key)
	t.pipe.SRem(ctx, t.store.recordIndexKey(uri), uri.ID())
	return nil
}

func (t *txStore) queueWriteRecord(ctx context.Context, record *core.Record) error {
	uri := record.URI()
	key := t.store.recordKey(uri)

	fields, err := encodeRecord(record)
	if err != nil {
		return err
	}
	fields[sentinelField] = "1"

	t.pipe.Del(ctx, key)
	t.pipe.HSet(ctx, key, fields)
	t.pipe.SAdd(ctx, t.store.recordIndexKey(uri), record.URI().ID())
	t.pipe.SAdd(ctx, t.store.schemaIndexKey(uri), record.URI().Schema())
	return nil
}

func (t *txStore) CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("%w: %w", store.ErrSchemaViolation, err)
	}
	key := t.store.schemaKey(uri)
	exists, err := t.store.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check schema exists: %w", err)
	}
	if exists > 0 {
		return store.ErrAlreadyExists
	}
	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("xdbredis: marshal schema: %w", err)
	}

	t.pipe.Set(ctx, key, data, 0)
	t.pipe.SAdd(ctx, t.store.schemaIndexKey(uri), uri.Schema())
	t.pipe.SAdd(ctx, t.store.nsIndexKey(), uri.NS())
	return nil
}

func (t *txStore) UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("%w: %w", store.ErrSchemaViolation, err)
	}
	existing, err := t.store.GetSchema(ctx, uri)
	if err != nil {
		return err
	}
	if vErr := schema.ValidateUpdate(existing, def); vErr != nil {
		return fmt.Errorf("%w: %w", store.ErrSchemaViolation, vErr)
	}
	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("xdbredis: marshal schema: %w", err)
	}
	t.pipe.Set(ctx, t.store.schemaKey(uri), data, 0)
	return nil
}

func (t *txStore) DeleteSchema(ctx context.Context, uri *core.URI) error {
	key := t.store.schemaKey(uri)
	exists, err := t.store.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check schema exists: %w", err)
	}
	if exists == 0 {
		return store.ErrNotFound
	}

	idxKey := t.store.recordIndexKey(uri)
	recordIDs, err := t.store.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: list record index: %w", err)
	}

	ns := uri.NS()
	schemaName := uri.Schema()
	for _, id := range recordIDs {
		recURI := core.MustNewURI(ns, schemaName, id)
		t.pipe.Del(ctx, t.store.recordKey(recURI))
	}

	t.pipe.Del(ctx, key)
	t.pipe.Del(ctx, idxKey)
	t.pipe.SRem(ctx, t.store.schemaIndexKey(uri), schemaName)
	// Namespace index cleanup (when last schema is removed) is skipped
	// inside a transaction — it requires post-EXEC observation.
	return nil
}

func (t *txStore) DeleteSchemaRecords(_ context.Context, _ *core.URI) error {
	return nil
}
