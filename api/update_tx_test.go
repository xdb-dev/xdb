package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// cloningStore wraps a store and returns deep copies from GetRecord and
// GetSchema, simulating backends (fs, redis, sqlite) that deserialize fresh
// objects on every read instead of sharing pointers. Without copies, the
// in-memory store aliases records and hides read-patch-write races.
type cloningStore struct {
	store.Store
	tx store.TX // nil when the inner store does not support TX
}

func newCloningStore(inner store.Store) *cloningStore {
	c := &cloningStore{Store: inner}
	if tx, ok := inner.(store.TX); ok {
		c.tx = tx
	}
	return c
}

func (c *cloningStore) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	rec, err := c.Store.GetRecord(ctx, uri)
	if err != nil {
		return nil, err
	}
	return cloneRecord(rec), nil
}

func (c *cloningStore) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	def, err := c.Store.GetSchema(ctx, uri)
	if err != nil {
		return nil, err
	}
	return cloneDef(def), nil
}

// Run implements [store.TX], wrapping the tx-scoped store in a cloning view.
func (c *cloningStore) Run(ctx context.Context, fn func(tx store.Store) error) error {
	if c.tx == nil {
		return fmt.Errorf("cloningStore: inner store does not support TX")
	}
	return c.tx.Run(ctx, func(tx store.Store) error {
		return fn(&cloningStore{Store: tx})
	})
}

func cloneRecord(rec *core.Record) *core.Record {
	uri := rec.URI()
	clone := core.NewRecord(uri.NS(), uri.Schema(), uri.ID())
	for _, t := range rec.Tuples() {
		clone.Set(t.Attr(), t.Value())
	}
	return clone
}

func cloneDef(def *schema.Def) *schema.Def {
	clone := *def
	clone.Fields = maps.Clone(def.Fields)
	clone.Annotations = maps.Clone(def.Annotations)
	return &clone
}

// TestRecordService_Update_ConcurrentPatches verifies that two concurrent
// patches to different attributes both survive. Requires Update to run its
// read-patch-write inside a transaction on TX-capable stores.
func TestRecordService_Update_ConcurrentPatches(t *testing.T) {
	ctx := context.Background()

	for i := range 100 {
		s := newCloningStore(store.New(xdbmemory.NewDriver()))
		svc := api.NewRecordService(s)

		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Original","author":"Alice"}`),
		})
		require.NoError(t, err)

		patches := []json.RawMessage{
			json.RawMessage(`{"title":"Updated"}`),
			json.RawMessage(`{"author":"Bob"}`),
		}

		var wg sync.WaitGroup
		errs := make([]error, len(patches))
		for j, patch := range patches {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[j] = svc.Update(ctx, &api.UpdateRecordRequest{
					URI:  "xdb://com.example/posts/post-1",
					Data: patch,
				})
			}()
		}
		wg.Wait()

		for j, err := range errs {
			require.NoError(t, err, "iteration %d patch %d", i, j)
		}

		resp, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Updated", m["title"], "iteration %d lost title patch", i)
		assert.Equal(t, "Bob", m["author"], "iteration %d lost author patch", i)

		if t.Failed() {
			return
		}
	}
}

// TestSchemaService_Update_ConcurrentPatches verifies that two concurrent
// field patches to a schema both survive without conflict errors.
func TestSchemaService_Update_ConcurrentPatches(t *testing.T) {
	ctx := context.Background()

	for i := range 100 {
		s := newCloningStore(store.New(xdbmemory.NewDriver()))
		svc := api.NewSchemaService(s)

		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://com.example/posts",
			Data: json.RawMessage(`{"fields":{"title":{"type":"string"}}}`),
		})
		require.NoError(t, err)

		patches := []json.RawMessage{
			json.RawMessage(`{"fields":{"author":{"type":"string"}}}`),
			json.RawMessage(`{"fields":{"rating":{"type":"float"}}}`),
		}

		var wg sync.WaitGroup
		errs := make([]error, len(patches))
		for j, patch := range patches {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[j] = svc.Update(ctx, &api.UpdateSchemaRequest{
					URI:  "xdb://com.example/posts",
					Data: patch,
				})
			}()
		}
		wg.Wait()

		for j, err := range errs {
			require.NoError(t, err, "iteration %d patch %d", i, j)
		}

		resp, err := svc.Get(ctx, &api.GetSchemaRequest{
			URI: "xdb://com.example/posts",
		})
		require.NoError(t, err)

		assert.Contains(t, resp.Data.Fields, "title", "iteration %d", i)
		assert.Contains(t, resp.Data.Fields, "author", "iteration %d lost author patch", i)
		assert.Contains(t, resp.Data.Fields, "rating", "iteration %d lost rating patch", i)

		if t.Failed() {
			return
		}
	}
}
