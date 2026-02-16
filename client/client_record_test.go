package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/client"
	"github.com/xdb-dev/xdb/core"
)

func TestClient_PutRecords_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when store is not configured")
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	records := []*core.Record{
		core.NewRecord("com.example", "posts", "post-1").
			Set("title", "First Post").
			Set("content", "Hello World"),
	}

	err = c.PutRecords(context.Background(), records)

	assert.ErrorIs(t, err, client.ErrRecordStoreNotConfigured)
}

func TestClient_GetRecords_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when store is not configured")
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/posts/post-1"),
		core.MustParseURI("xdb://com.example/posts/post-2"),
	}

	records, notFound, err := c.GetRecords(context.Background(), uris)

	assert.ErrorIs(t, err, client.ErrRecordStoreNotConfigured)
	assert.Nil(t, records)
	assert.Nil(t, notFound)
}

func TestClient_DeleteRecords_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when store is not configured")
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/posts/post-1"),
		core.MustParseURI("xdb://com.example/posts/post-2"),
	}

	err = c.DeleteRecords(context.Background(), uris)

	assert.ErrorIs(t, err, client.ErrRecordStoreNotConfigured)
}

func TestClient_PutRecords_StubSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/records", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		resp := &api.PutRecordsResponse{
			Created: []string{},
			Updated: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithRecordStore().Build()
	require.NoError(t, err)

	records := []*core.Record{
		core.NewRecord("com.example", "posts", "post-1").
			Set("title", "First Post").
			Set("content", "Hello World"),
		core.NewRecord("com.example", "posts", "post-2").
			Set("title", "Second Post").
			Set("content", "Another post"),
	}

	err = c.PutRecords(context.Background(), records)

	assert.NoError(t, err)
}

func TestClient_GetRecords_StubSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/records", r.URL.Path)

		resp := &api.GetRecordsResponse{
			Records:  []*api.RecordJSON{},
			NotFound: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithRecordStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/posts/post-1"),
		core.MustParseURI("xdb://com.example/posts/post-2"),
	}

	records, notFound, err := c.GetRecords(context.Background(), uris)

	assert.NoError(t, err)
	assert.Empty(t, records)
	assert.Empty(t, notFound)
}

func TestClient_DeleteRecords_StubSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/v1/records", r.URL.Path)

		resp := &api.DeleteRecordsResponse{
			Deleted: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c, err := client.NewBuilder(&client.Config{
		Addr: server.Listener.Addr().String(),
	}).WithRecordStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/posts/post-1"),
		core.MustParseURI("xdb://com.example/posts/post-2"),
	}

	err = c.DeleteRecords(context.Background(), uris)

	assert.NoError(t, err)
}
