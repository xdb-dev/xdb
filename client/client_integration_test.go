package client_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/client"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func TestIntegration_FullLifecycleWithMemoryStore(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithRecordStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: ts.Listener.Addr().String(),
	}).
		WithSchemaStore().
		WithTupleStore().
		WithRecordStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("health check succeeds", func(t *testing.T) {
		err := c.Health(ctx)
		require.NoError(t, err)
	})

	t.Run("put schema", func(t *testing.T) {
		schemaURI := core.MustParseURI("xdb://testns/users")
		schemaDef := &schema.Def{
			NS:          core.NewNS("testns"),
			Name:        "users",
			Description: "User schema for testing",
			Version:     "1.0.0",
			Mode:        schema.ModeDynamic,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
				{Name: "age", Type: core.TypeFloat},
			},
		}

		err := c.PutSchema(ctx, schemaURI, schemaDef)
		require.NoError(t, err)
	})

	t.Run("get schema", func(t *testing.T) {
		schemaURI := core.MustParseURI("xdb://testns/users")

		def, err := c.GetSchema(ctx, schemaURI)
		require.NoError(t, err)
		require.NotNil(t, def)
		assert.Equal(t, "users", def.Name)
		assert.Equal(t, "testns", def.NS.String())
		assert.Equal(t, schema.ModeDynamic, def.Mode)
		assert.Len(t, def.Fields, 2)
	})

	t.Run("put tuples", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("testns/users/user1", "name", "Alice"),
			core.NewTuple("testns/users/user1", "age", float64(30)),
			core.NewTuple("testns/users/user2", "name", "Bob"),
			core.NewTuple("testns/users/user2", "age", float64(25)),
		}

		err := c.PutTuples(ctx, tuples)
		require.NoError(t, err)
	})

	t.Run("get tuples", func(t *testing.T) {
		uris := []*core.URI{
			core.MustParseURI("xdb://testns/users/user1#name"),
			core.MustParseURI("xdb://testns/users/user1#age"),
			core.MustParseURI("xdb://testns/users/user2#name"),
		}

		tuples, missing, err := c.GetTuples(ctx, uris)
		require.NoError(t, err)
		assert.Len(t, tuples, 3)
		assert.Empty(t, missing)
	})

	t.Run("get tuples with missing", func(t *testing.T) {
		uris := []*core.URI{
			core.MustParseURI("xdb://testns/users/user1#name"),
			core.MustParseURI("xdb://testns/users/nonexistent#name"),
		}

		tuples, missing, err := c.GetTuples(ctx, uris)
		require.NoError(t, err)
		assert.Len(t, tuples, 1)
		assert.Len(t, missing, 1)
	})

	t.Run("delete tuples", func(t *testing.T) {
		uris := []*core.URI{
			core.MustParseURI("xdb://testns/users/user2#name"),
			core.MustParseURI("xdb://testns/users/user2#age"),
		}

		err := c.DeleteTuples(ctx, uris)
		require.NoError(t, err)

		tuples, missing, err := c.GetTuples(ctx, uris)
		require.NoError(t, err)
		assert.Empty(t, tuples)
		assert.Len(t, missing, 2)
	})

	t.Run("delete schema", func(t *testing.T) {
		schemaURI := core.MustParseURI("xdb://testns/users")

		err := c.DeleteSchema(ctx, schemaURI)
		require.NoError(t, err)

		_, err = c.GetSchema(ctx, schemaURI)
		require.Error(t, err)
	})
}

func TestIntegration_ClientServerCompatibility(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithRecordStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: ts.Listener.Addr().String(),
	}).
		WithSchemaStore().
		WithTupleStore().
		WithRecordStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	schemaURI := core.MustParseURI("xdb://compat/products")
	schemaDef := &schema.Def{
		NS:      core.NewNS("compat"),
		Name:    "products",
		Version: "1.0.0",
		Mode:    schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "title", Type: core.TypeString},
			{Name: "price", Type: core.TypeFloat},
		},
	}

	err = c.PutSchema(ctx, schemaURI, schemaDef)
	require.NoError(t, err)

	t.Run("schema operations", func(t *testing.T) {
		def, err := c.GetSchema(ctx, schemaURI)
		require.NoError(t, err)
		assert.Equal(t, "products", def.Name)

		listURI := core.MustParseURI("xdb://compat")
		schemas, err := c.ListSchemas(ctx, listURI)
		require.NoError(t, err)
		assert.Len(t, schemas, 1)
	})

	t.Run("tuple operations", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("compat/products/prod1", "title", "Widget"),
			core.NewTuple("compat/products/prod1", "price", 9.99),
		}

		err := c.PutTuples(ctx, tuples)
		require.NoError(t, err)

		uris := []*core.URI{
			core.MustParseURI("xdb://compat/products/prod1#title"),
			core.MustParseURI("xdb://compat/products/prod1#price"),
		}

		retrievedTuples, missing, err := c.GetTuples(ctx, uris)
		require.NoError(t, err)
		assert.Len(t, retrievedTuples, 2)
		assert.Empty(t, missing)

		err = c.DeleteTuples(ctx, uris[:1])
		require.NoError(t, err)
	})

	t.Run("record operations", func(t *testing.T) {
		record := core.NewRecord("compat", "products", "prod2").
			Set("title", "Gadget").
			Set("price", 19.99)

		err := c.PutRecords(ctx, []*core.Record{record})
		require.NoError(t, err)

		recordURI := core.MustParseURI("xdb://compat/products/prod2")
		records, missing, err := c.GetRecords(ctx, []*core.URI{recordURI})
		require.NoError(t, err)
		assert.Len(t, records, 1)
		assert.Empty(t, missing)

		err = c.DeleteRecords(ctx, []*core.URI{recordURI})
		require.NoError(t, err)

		records, missing, err = c.GetRecords(ctx, []*core.URI{recordURI})
		require.NoError(t, err)
		assert.Empty(t, records)
		assert.Len(t, missing, 1)
	})

	t.Run("health operations", func(t *testing.T) {
		err := c.Health(ctx)
		require.NoError(t, err)

		err = c.Ping(ctx)
		require.NoError(t, err)
	})
}

func TestIntegration_UnixSocketTransport(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("xdb-integration-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)

	memStore := xdbmemory.New()

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	httpServer := &http.Server{
		Handler: server.Handler(),
	}

	go func() {
		_ = httpServer.Serve(listener)
	}()
	defer httpServer.Close()

	time.Sleep(50 * time.Millisecond)

	c, err := client.NewBuilder(&client.Config{
		SocketPath: socketPath,
	}).
		WithSchemaStore().
		WithTupleStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	err = c.Ping(ctx)
	require.NoError(t, err)

	err = c.Health(ctx)
	require.NoError(t, err)

	schemaURI := core.MustParseURI("xdb://sockettest/items")
	schemaDef := &schema.Def{
		NS:      core.NewNS("sockettest"),
		Name:    "items",
		Version: "1.0.0",
		Mode:    schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "value", Type: core.TypeString},
		},
	}

	err = c.PutSchema(ctx, schemaURI, schemaDef)
	require.NoError(t, err)

	def, err := c.GetSchema(ctx, schemaURI)
	require.NoError(t, err)
	assert.Equal(t, "items", def.Name)
}

func TestIntegration_TCPTransport(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: ts.Listener.Addr().String(),
	}).
		WithSchemaStore().
		WithTupleStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	err = c.Ping(ctx)
	require.NoError(t, err)

	err = c.Health(ctx)
	require.NoError(t, err)

	schemaURI := core.MustParseURI("xdb://tcptest/data")
	schemaDef := &schema.Def{
		NS:      core.NewNS("tcptest"),
		Name:    "data",
		Version: "1.0.0",
		Mode:    schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "content", Type: core.TypeString},
		},
	}

	err = c.PutSchema(ctx, schemaURI, schemaDef)
	require.NoError(t, err)

	def, err := c.GetSchema(ctx, schemaURI)
	require.NoError(t, err)
	assert.Equal(t, "data", def.Name)

	tuples := []*core.Tuple{
		core.NewTuple("tcptest/data/item1", "content", "test value"),
	}

	err = c.PutTuples(ctx, tuples)
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://tcptest/data/item1#content"),
	}

	retrievedTuples, missing, err := c.GetTuples(ctx, uris)
	require.NoError(t, err)
	assert.Len(t, retrievedTuples, 1)
	assert.Empty(t, missing)
}

func TestIntegration_MixedStoreConfiguration(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	t.Run("client with matching configuration", func(t *testing.T) {
		c, err := client.NewBuilder(&client.Config{
			Addr: ts.Listener.Addr().String(),
		}).
			WithSchemaStore().
			WithTupleStore().
			Build()
		require.NoError(t, err)

		ctx := context.Background()

		schemaURI := core.MustParseURI("xdb://mixed/config")
		schemaDef := &schema.Def{
			NS:      core.NewNS("mixed"),
			Name:    "config",
			Version: "1.0.0",
			Mode:    schema.ModeDynamic,
			Fields: []*schema.FieldDef{
				{Name: "key", Type: core.TypeString},
			},
		}

		err = c.PutSchema(ctx, schemaURI, schemaDef)
		require.NoError(t, err)

		def, err := c.GetSchema(ctx, schemaURI)
		require.NoError(t, err)
		assert.Equal(t, "config", def.Name)

		tuples := []*core.Tuple{
			core.NewTuple("mixed/config/entry1", "key", "value1"),
		}

		err = c.PutTuples(ctx, tuples)
		require.NoError(t, err)
	})

	t.Run("client with disabled stores returns proper errors", func(t *testing.T) {
		c, err := client.NewBuilder(&client.Config{
			Addr: ts.Listener.Addr().String(),
		}).
			WithSchemaStore().
			WithTupleStore().
			WithRecordStore().
			WithHealthStore().
			Build()
		require.NoError(t, err)

		ctx := context.Background()

		record := core.NewRecord("mixed", "config", "rec1").
			Set("key", "value")

		err = c.PutRecords(ctx, []*core.Record{record})
		require.Error(t, err, "Server does not have record store enabled, should fail")

		err = c.Health(ctx)
		require.Error(t, err, "Server does not have health store enabled, should fail")
	})

	t.Run("client without store returns configuration error", func(t *testing.T) {
		c, err := client.NewBuilder(&client.Config{
			Addr: ts.Listener.Addr().String(),
		}).
			WithSchemaStore().
			Build()
		require.NoError(t, err)

		ctx := context.Background()

		tuples := []*core.Tuple{
			core.NewTuple("mixed/config/entry1", "key", "value1"),
		}

		err = c.PutTuples(ctx, tuples)
		require.Error(t, err)
		assert.ErrorIs(t, err, client.ErrTupleStoreNotConfigured)

		record := core.NewRecord("mixed", "config", "rec1").
			Set("key", "value")
		err = c.PutRecords(ctx, []*core.Record{record})
		require.Error(t, err)
		assert.ErrorIs(t, err, client.ErrRecordStoreNotConfigured)

		err = c.Health(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, client.ErrHealthStoreNotConfigured)
	})
}

func TestIntegration_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithRecordStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: ts.Listener.Addr().String(),
	}).
		WithSchemaStore().
		WithTupleStore().
		WithRecordStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	schemaURI := core.MustParseURI("xdb://concurrent/data")
	schemaDef := &schema.Def{
		NS:      core.NewNS("concurrent"),
		Name:    "data",
		Version: "1.0.0",
		Mode:    schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "value", Type: core.TypeFloat},
		},
	}

	err = c.PutSchema(ctx, schemaURI, schemaDef)
	require.NoError(t, err)

	const numGoroutines = 10
	const operationsPerGoroutine = 5

	errChan := make(chan error, numGoroutines*operationsPerGoroutine)
	doneChan := make(chan struct{}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer func() { doneChan <- struct{}{} }()

			for j := 0; j < operationsPerGoroutine; j++ {
				tupleID := fmt.Sprintf("concurrent/data/worker%d-item%d", workerID, j)
				tuples := []*core.Tuple{
					core.NewTuple(tupleID, "value", float64(workerID*100+j)),
				}

				if putErr := c.PutTuples(ctx, tuples); putErr != nil {
					errChan <- putErr
					continue
				}

				uri := core.MustParseURI(fmt.Sprintf("xdb://%s#value", tupleID))
				_, _, getErr := c.GetTuples(ctx, []*core.URI{uri})
				if getErr != nil {
					errChan <- getErr
				}
			}
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}

	close(errChan)
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	assert.Empty(t, errors, "concurrent operations should not produce errors")
}

func TestIntegration_ErrorHandling(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithRecordStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: ts.Listener.Addr().String(),
	}).
		WithSchemaStore().
		WithTupleStore().
		WithRecordStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("get nonexistent schema returns not found", func(t *testing.T) {
		schemaURI := core.MustParseURI("xdb://nonexistent/schema")

		_, err := c.GetSchema(ctx, schemaURI)
		require.Error(t, err)
	})

	t.Run("put tuples without schema fails", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("noschema/data/item1", "value", "test"),
		}

		err := c.PutTuples(ctx, tuples)
		require.Error(t, err)
	})

	t.Run("put records without schema fails", func(t *testing.T) {
		record := core.NewRecord("noschema", "data", "item1").
			Set("value", "test")

		err := c.PutRecords(ctx, []*core.Record{record})
		require.Error(t, err)
	})
}

func TestIntegration_LargePayloads(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithRecordStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: ts.Listener.Addr().String(),
	}).
		WithSchemaStore().
		WithTupleStore().
		WithRecordStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	schemaURI := core.MustParseURI("xdb://large/data")
	schemaDef := &schema.Def{
		NS:      core.NewNS("large"),
		Name:    "data",
		Version: "1.0.0",
		Mode:    schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "content", Type: core.TypeString},
		},
	}

	err = c.PutSchema(ctx, schemaURI, schemaDef)
	require.NoError(t, err)

	t.Run("many tuples in single request", func(t *testing.T) {
		const numTuples = 100
		tuples := make([]*core.Tuple, numTuples)

		for i := 0; i < numTuples; i++ {
			tuples[i] = core.NewTuple(fmt.Sprintf("large/data/item%d", i), "content", fmt.Sprintf("value-%d", i))
		}

		err := c.PutTuples(ctx, tuples)
		require.NoError(t, err)

		uris := make([]*core.URI, numTuples)
		for i := 0; i < numTuples; i++ {
			uris[i] = core.MustParseURI(fmt.Sprintf("xdb://large/data/item%d#content", i))
		}

		retrievedTuples, missing, err := c.GetTuples(ctx, uris)
		require.NoError(t, err)
		assert.Len(t, retrievedTuples, numTuples)
		assert.Empty(t, missing)
	})

	t.Run("large string value", func(t *testing.T) {
		largeContent := make([]byte, 10000)
		for i := range largeContent {
			largeContent[i] = byte('a' + (i % 26))
		}

		tuples := []*core.Tuple{
			core.NewTuple("large/data/bigcontent", "content", string(largeContent)),
		}

		err := c.PutTuples(ctx, tuples)
		require.NoError(t, err)

		uris := []*core.URI{
			core.MustParseURI("xdb://large/data/bigcontent#content"),
		}

		retrievedTuples, _, err := c.GetTuples(ctx, uris)
		require.NoError(t, err)
		require.Len(t, retrievedTuples, 1)
	})
}

func TestIntegration_ContextTimeout(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr:    ts.Listener.Addr().String(),
		Timeout: 100 * time.Millisecond,
	}).
		WithSchemaStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	err = c.Ping(ctx)
	require.NoError(t, err)
}

func TestIntegration_MultipleSchemas(t *testing.T) {
	t.Parallel()

	memStore := xdbmemory.New()

	server, err := api.NewServerBuilder(&api.ServerConfig{}).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithHealthStore(memStore).
		Build()
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: ts.Listener.Addr().String(),
	}).
		WithSchemaStore().
		WithTupleStore().
		WithHealthStore().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	schemas := []struct {
		uri  string
		name string
	}{
		{"xdb://multins/users", "users"},
		{"xdb://multins/products", "products"},
		{"xdb://multins/orders", "orders"},
	}

	for _, s := range schemas {
		schemaURI := core.MustParseURI(s.uri)
		schemaDef := &schema.Def{
			NS:      core.NewNS("multins"),
			Name:    s.name,
			Version: "1.0.0",
			Mode:    schema.ModeDynamic,
			Fields: []*schema.FieldDef{
				{Name: "id", Type: core.TypeString},
			},
		}

		err := c.PutSchema(ctx, schemaURI, schemaDef)
		require.NoError(t, err)
	}

	listURI := core.MustParseURI("xdb://multins")
	schemaList, err := c.ListSchemas(ctx, listURI)
	require.NoError(t, err)
	assert.Len(t, schemaList, 3)

	for _, s := range schemas {
		schemaURI := core.MustParseURI(s.uri)
		def, err := c.GetSchema(ctx, schemaURI)
		require.NoError(t, err)
		assert.Equal(t, s.name, def.Name)
	}
}
