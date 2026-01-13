package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

func TestClient_PutSchema_Success(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.example").Schema("users").MustURI()
	schemaDef := &schema.Def{
		NS:          core.NewNS("com.example"),
		Name:        "users",
		Description: "User schema",
		Version:     "1.0.0",
		Mode:        schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
			{Name: "age", Type: core.TypeInt},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/schemas", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req api.PutSchemaRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.NotEmpty(t, req.URI)
		assert.NotNil(t, req.Schema)

		resp := api.PutSchemaResponse{URI: uri}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	err = client.PutSchema(context.Background(), uri, schemaDef)
	assert.NoError(t, err)
}

func TestClient_PutSchema_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithHealthStore().Build()
	require.NoError(t, err)

	uri := core.New().NS("com.example").Schema("users").MustURI()
	schemaDef := &schema.Def{Name: "users"}

	err = client.PutSchema(context.Background(), uri, schemaDef)
	assert.ErrorIs(t, err, api.ErrSchemaStoreNotConfigured)
}

func TestClient_PutSchema_URISerialization(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.example").Schema("users").MustURI()
	expectedURIString := uri.String()

	var capturedURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req api.PutSchemaRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		capturedURI = req.URI

		resp := api.PutSchemaResponse{URI: uri}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	schemaDef := &schema.Def{Name: "users"}
	err = client.PutSchema(context.Background(), uri, schemaDef)
	require.NoError(t, err)

	assert.Equal(t, expectedURIString, capturedURI)
}

func TestClient_PutSchema_SchemaSerialization(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.example").Schema("users").MustURI()
	schemaDef := &schema.Def{
		NS:          core.NewNS("com.example"),
		Name:        "users",
		Description: "User schema for testing",
		Version:     "2.0.0",
		Mode:        schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "email", Description: "User email", Type: core.TypeString},
			{Name: "active", Description: "Active status", Type: core.TypeBool},
		},
	}

	var capturedSchema *schema.Def
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req api.PutSchemaRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		capturedSchema = req.Schema

		resp := api.PutSchemaResponse{URI: uri}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	err = client.PutSchema(context.Background(), uri, schemaDef)
	require.NoError(t, err)

	require.NotNil(t, capturedSchema)
	assert.Equal(t, schemaDef.Name, capturedSchema.Name)
	assert.Equal(t, schemaDef.Description, capturedSchema.Description)
	assert.Equal(t, schemaDef.Version, capturedSchema.Version)
	assert.Equal(t, schemaDef.Mode, capturedSchema.Mode)
	require.Len(t, capturedSchema.Fields, 2)
	assert.Equal(t, "email", capturedSchema.Fields[0].Name)
	assert.Equal(t, "active", capturedSchema.Fields[1].Name)
}

func TestClient_PutSchema_ErrorHandling(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(api.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "invalid schema definition",
		})
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	uri := core.New().NS("com.example").Schema("users").MustURI()
	schemaDef := &schema.Def{Name: "users"}

	err = client.PutSchema(context.Background(), uri, schemaDef)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schema definition")
}

func TestClient_GetSchema_Success(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.example").Schema("posts").MustURI()
	expectedSchema := &schema.Def{
		NS:          core.NewNS("com.example"),
		Name:        "posts",
		Description: "Blog posts",
		Version:     "1.0.0",
		Mode:        schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "title", Type: core.TypeString},
			{Name: "content", Type: core.TypeString},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/schemas/")

		resp := api.GetSchemaResponse{Schema: expectedSchema}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	result, err := client.GetSchema(context.Background(), uri)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, expectedSchema.Name, result.Name)
	assert.Equal(t, expectedSchema.Description, result.Description)
	assert.Equal(t, expectedSchema.Version, result.Version)
	assert.Equal(t, expectedSchema.Mode, result.Mode)
}

func TestClient_GetSchema_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(api.ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "schema not found",
		})
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	uri := core.New().NS("com.example").Schema("nonexistent").MustURI()

	result, err := client.GetSchema(context.Background(), uri)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestClient_GetSchema_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithHealthStore().Build()
	require.NoError(t, err)

	uri := core.New().NS("com.example").Schema("users").MustURI()

	result, err := client.GetSchema(context.Background(), uri)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, api.ErrSchemaStoreNotConfigured)
}

func TestClient_GetSchema_ResponseDeserialization(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.example").Schema("products").MustURI()
	expectedSchema := &schema.Def{
		NS:          core.NewNS("com.example"),
		Name:        "products",
		Description: "Product catalog schema",
		Version:     "3.1.0",
		Mode:        schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "sku", Description: "Stock keeping unit", Type: core.TypeString},
			{Name: "price", Description: "Product price", Type: core.TypeFloat},
			{Name: "quantity", Description: "Available quantity", Type: core.TypeInt},
			{Name: "available", Description: "Availability flag", Type: core.TypeBool},
			{Name: "tags", Description: "Product tags", Type: core.NewArrayType(core.TIDString)},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := api.GetSchemaResponse{Schema: expectedSchema}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	result, err := client.GetSchema(context.Background(), uri)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "products", result.Name)
	assert.Equal(t, "Product catalog schema", result.Description)
	assert.Equal(t, "3.1.0", result.Version)
	assert.Equal(t, schema.ModeDynamic, result.Mode)

	require.Len(t, result.Fields, 5)
	assert.Equal(t, "sku", result.Fields[0].Name)
	assert.Equal(t, "Stock keeping unit", result.Fields[0].Description)
	assert.Equal(t, "price", result.Fields[1].Name)
	assert.Equal(t, "quantity", result.Fields[2].Name)
	assert.Equal(t, "available", result.Fields[3].Name)
	assert.Equal(t, "tags", result.Fields[4].Name)
}

func TestClient_ListSchemas_Success(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.example").MustURI()
	expectedSchemas := []*schema.Def{
		{
			NS:      core.NewNS("com.example"),
			Name:    "users",
			Version: "1.0.0",
			Mode:    schema.ModeStrict,
		},
		{
			NS:      core.NewNS("com.example"),
			Name:    "posts",
			Version: "2.0.0",
			Mode:    schema.ModeFlexible,
		},
		{
			NS:      core.NewNS("com.example"),
			Name:    "comments",
			Version: "1.5.0",
			Mode:    schema.ModeDynamic,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/schemas", r.URL.Path)

		resp := api.ListSchemasResponse{Schemas: expectedSchemas}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	results, err := client.ListSchemas(context.Background(), uri)
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.Equal(t, "users", results[0].Name)
	assert.Equal(t, "posts", results[1].Name)
	assert.Equal(t, "comments", results[2].Name)
}

func TestClient_ListSchemas_EmptyResult(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.empty").MustURI()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := api.ListSchemasResponse{Schemas: []*schema.Def{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	results, err := client.ListSchemas(context.Background(), uri)
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.NotNil(t, results)
}

func TestClient_ListSchemas_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithHealthStore().Build()
	require.NoError(t, err)

	uri := core.New().NS("com.example").MustURI()

	results, err := client.ListSchemas(context.Background(), uri)
	assert.Nil(t, results)
	assert.ErrorIs(t, err, api.ErrSchemaStoreNotConfigured)
}

func TestClient_DeleteSchema_Success(t *testing.T) {
	t.Parallel()

	uri := core.New().NS("com.example").Schema("obsolete").MustURI()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/schemas/")

		resp := api.DeleteSchemaResponse{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	err = client.DeleteSchema(context.Background(), uri)
	assert.NoError(t, err)
}

func TestClient_DeleteSchema_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(api.ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "schema not found",
		})
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	uri := core.New().NS("com.example").Schema("nonexistent").MustURI()

	err = client.DeleteSchema(context.Background(), uri)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestClient_DeleteSchema_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithHealthStore().Build()
	require.NoError(t, err)

	uri := core.New().NS("com.example").Schema("users").MustURI()

	err = client.DeleteSchema(context.Background(), uri)
	assert.ErrorIs(t, err, api.ErrSchemaStoreNotConfigured)
}

func TestClient_ListNamespaces_Success(t *testing.T) {
	t.Parallel()

	expectedNamespaces := []*core.NS{
		core.NewNS("com.example"),
		core.NewNS("org.testing"),
		core.NewNS("io.production"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/namespaces", r.URL.Path)

		resp := api.ListNamespacesResponse{Namespaces: expectedNamespaces}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithSchemaStore().Build()
	require.NoError(t, err)

	results, err := client.ListNamespaces(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.Equal(t, "com.example", results[0].String())
	assert.Equal(t, "org.testing", results[1].String())
	assert.Equal(t, "io.production", results[2].String())
}

func TestClient_ListNamespaces_StoreNotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	defer server.Close()

	client, err := api.NewClientBuilder(&api.ClientConfig{
		Addr: server.Listener.Addr().String(),
	}).WithHealthStore().Build()
	require.NoError(t, err)

	results, err := client.ListNamespaces(context.Background())
	assert.Nil(t, results)
	assert.ErrorIs(t, err, api.ErrSchemaStoreNotConfigured)
}
