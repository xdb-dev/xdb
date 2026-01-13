package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
)

func TestClient_PutTuples_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/tuples", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/user-1", "name", "Alice"),
		core.NewTuple("com.example/users/user-1", "age", int64(30)),
	}

	err = client.PutTuples(context.Background(), tuples)
	assert.NoError(t, err)
}

func TestClient_PutTuples_StoreNotConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when store is not configured")
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/user-1", "name", "Alice"),
	}

	err = client.PutTuples(context.Background(), tuples)
	assert.ErrorIs(t, err, api.ErrTupleStoreNotConfigured)
}

func TestClient_PutTuples_TupleSerialization(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/user-1", "name", "Alice"),
		core.NewTuple("com.example/users/user-1", "active", true),
	}

	err = client.PutTuples(context.Background(), tuples)
	require.NoError(t, err)

	var request []map[string]any
	err = json.Unmarshal(receivedBody, &request)
	require.NoError(t, err)

	assert.Len(t, request, 2)

	assert.Contains(t, request[0], "id")
	assert.Contains(t, request[0], "attr")
	assert.Contains(t, request[0], "value")

	assert.Equal(t, "name", request[0]["attr"])
	assert.Equal(t, "Alice", request[0]["value"])

	assert.Equal(t, "active", request[1]["attr"])
	assert.Equal(t, true, request[1]["value"])
}

func TestClient_PutTuples_ValueTypePreservation(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	tuples := []*core.Tuple{
		core.NewTuple("com.example/test/t1", "string_val", "hello"),
		core.NewTuple("com.example/test/t1", "int_val", int64(42)),
		core.NewTuple("com.example/test/t1", "bool_val", true),
		core.NewTuple("com.example/test/t1", "array_val", []string{"a", "b", "c"}),
		core.NewTuple("com.example/test/t1", "map_val", map[string]string{"key": "value"}),
	}

	err = client.PutTuples(context.Background(), tuples)
	require.NoError(t, err)

	var request []map[string]any
	err = json.Unmarshal(receivedBody, &request)
	require.NoError(t, err)

	assert.Len(t, request, 5)

	assert.Equal(t, "hello", request[0]["value"])

	assert.Equal(t, float64(42), request[1]["value"])

	assert.Equal(t, true, request[2]["value"])

	arrayVal, ok := request[3]["value"].([]any)
	assert.True(t, ok, "array value should be []any")
	assert.Len(t, arrayVal, 3)
	assert.Equal(t, "a", arrayVal[0])
	assert.Equal(t, "b", arrayVal[1])
	assert.Equal(t, "c", arrayVal[2])

	mapVal, ok := request[4]["value"].(map[string]any)
	assert.True(t, ok, "map value should be map[string]any")
	assert.Equal(t, "value", mapVal["key"])
}

func TestClient_GetTuples_SuccessAllFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/tuples", r.URL.Path)

		response := api.GetTuplesResponse{
			Tuples: []*api.Tuple{
				{ID: "com.example/users/user-1", Attr: "name", Value: "Alice"},
				{ID: "com.example/users/user-1", Attr: "age", Value: float64(30)},
			},
			Missing: []string{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/users/user-1#name"),
		core.MustParseURI("xdb://com.example/users/user-1#age"),
	}

	tuples, missing, err := client.GetTuples(context.Background(), uris)
	assert.NoError(t, err)
	assert.Len(t, tuples, 2)
	assert.Empty(t, missing)
}

func TestClient_GetTuples_SomeMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.GetTuplesResponse{
			Tuples: []*api.Tuple{
				{ID: "com.example/users/user-1", Attr: "name", Value: "Alice"},
			},
			Missing: []string{"xdb://com.example/users/user-1#email"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/users/user-1#name"),
		core.MustParseURI("xdb://com.example/users/user-1#email"),
	}

	tuples, missing, err := client.GetTuples(context.Background(), uris)
	assert.NoError(t, err)
	assert.Len(t, tuples, 1)
	assert.Len(t, missing, 1)
	assert.Equal(t, "xdb://com.example/users/user-1#email", missing[0].String())
}

func TestClient_GetTuples_AllMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.GetTuplesResponse{
			Tuples: []*api.Tuple{},
			Missing: []string{
				"xdb://com.example/users/user-1#name",
				"xdb://com.example/users/user-1#age",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/users/user-1#name"),
		core.MustParseURI("xdb://com.example/users/user-1#age"),
	}

	tuples, missing, err := client.GetTuples(context.Background(), uris)
	assert.NoError(t, err)
	assert.Empty(t, tuples)
	assert.Len(t, missing, 2)
}

func TestClient_GetTuples_TupleDeserialization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.GetTuplesResponse{
			Tuples: []*api.Tuple{
				{ID: "com.example/test/t1", Attr: "string_val", Value: "hello"},
				{ID: "com.example/test/t1", Attr: "int_val", Value: float64(42)},
				{ID: "com.example/test/t1", Attr: "bool_val", Value: true},
				{ID: "com.example/test/t1", Attr: "array_val", Value: []any{"a", "b"}},
				{ID: "com.example/test/t1", Attr: "map_val", Value: map[string]any{"key": "value"}},
			},
			Missing: []string{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/test/t1#string_val"),
		core.MustParseURI("xdb://com.example/test/t1#int_val"),
		core.MustParseURI("xdb://com.example/test/t1#bool_val"),
		core.MustParseURI("xdb://com.example/test/t1#array_val"),
		core.MustParseURI("xdb://com.example/test/t1#map_val"),
	}

	tuples, _, err := client.GetTuples(context.Background(), uris)
	require.NoError(t, err)
	require.Len(t, tuples, 5)

	assert.Equal(t, "string_val", tuples[0].Attr().String())
	assert.Equal(t, "hello", tuples[0].Value().Unwrap())

	assert.Equal(t, "int_val", tuples[1].Attr().String())

	assert.Equal(t, "bool_val", tuples[2].Attr().String())
	assert.Equal(t, true, tuples[2].Value().Unwrap())

	assert.Equal(t, "array_val", tuples[3].Attr().String())

	assert.Equal(t, "map_val", tuples[4].Attr().String())
}

func TestClient_DeleteTuples_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/v1/tuples", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/users/user-1#name"),
		core.MustParseURI("xdb://com.example/users/user-1#age"),
	}

	err = client.DeleteTuples(context.Background(), uris)
	assert.NoError(t, err)
}

func TestClient_DeleteTuples_URISerialization(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithTupleStore().Build()
	require.NoError(t, err)

	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/users/user-1#name"),
		core.MustParseURI("xdb://com.example/users/user-2#email"),
	}

	err = client.DeleteTuples(context.Background(), uris)
	require.NoError(t, err)

	var request []string
	err = json.Unmarshal(receivedBody, &request)
	require.NoError(t, err)

	assert.Len(t, request, 2)
	assert.Equal(t, "xdb://com.example/users/user-1#name", request[0])
	assert.Equal(t, "xdb://com.example/users/user-2#email", request[1])
}
