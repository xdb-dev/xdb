package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/x"
)

const defaultClientTimeout = 30 * time.Second

// ClientConfig holds configuration for the HTTP client.
type ClientConfig struct {
	Addr       string
	SocketPath string
	Timeout    time.Duration
}

func (c *ClientConfig) Validate() error {
	if c.Addr == "" && c.SocketPath == "" {
		return ErrNoAddressConfigured
	}
	return nil
}

func (c *ClientConfig) EffectiveAddress() string {
	if c.SocketPath != "" {
		return c.SocketPath
	}
	return c.Addr
}

func (c *ClientConfig) EffectiveTimeout() time.Duration {
	if c.Timeout == 0 {
		return defaultClientTimeout
	}
	return c.Timeout
}

func (c *ClientConfig) UsesUnixSocket() bool {
	return c.SocketPath != ""
}

// ClientBuilder constructs a Client with optional store capabilities.
type ClientBuilder struct {
	cfg        *ClientConfig
	httpClient *http.Client

	schemaEnabled bool
	tupleEnabled  bool
	recordEnabled bool
	healthEnabled bool
}

// NewClientBuilder creates a new ClientBuilder with the given configuration.
func NewClientBuilder(cfg *ClientConfig) *ClientBuilder {
	return &ClientBuilder{
		cfg: cfg,
	}
}

// WithSchemaStore enables schema store operations on the client.
func (b *ClientBuilder) WithSchemaStore() *ClientBuilder {
	b.schemaEnabled = true
	return b
}

// WithTupleStore enables tuple store operations on the client.
func (b *ClientBuilder) WithTupleStore() *ClientBuilder {
	b.tupleEnabled = true
	return b
}

// WithRecordStore enables record store operations on the client.
func (b *ClientBuilder) WithRecordStore() *ClientBuilder {
	b.recordEnabled = true
	return b
}

// WithHealthStore enables health check operations on the client.
func (b *ClientBuilder) WithHealthStore() *ClientBuilder {
	b.healthEnabled = true
	return b
}

// WithHTTPClient sets a custom HTTP client.
func (b *ClientBuilder) WithHTTPClient(client *http.Client) *ClientBuilder {
	b.httpClient = client
	return b
}

// Build creates the Client with all configured options.
func (b *ClientBuilder) Build() (*Client, error) {
	if b.cfg == nil {
		return nil, ErrNoAddressConfigured
	}

	if !b.schemaEnabled && !b.tupleEnabled && !b.recordEnabled && !b.healthEnabled {
		return nil, ErrNoStoresConfigured
	}

	if err := b.cfg.Validate(); err != nil {
		return nil, err
	}

	httpClient := b.httpClient
	if httpClient == nil {
		httpClient = b.createHTTPClient()
	}

	baseURL := b.buildBaseURL()

	return &Client{
		cfg:           b.cfg,
		httpClient:    httpClient,
		baseURL:       baseURL,
		schemaEnabled: b.schemaEnabled,
		tupleEnabled:  b.tupleEnabled,
		recordEnabled: b.recordEnabled,
		healthEnabled: b.healthEnabled,
	}, nil
}

func (b *ClientBuilder) createHTTPClient() *http.Client {
	var transport http.RoundTripper

	if b.cfg.UsesUnixSocket() {
		socketPath := b.cfg.SocketPath
		transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		}
	} else {
		transport = http.DefaultTransport
	}

	return &http.Client{
		Transport: transport,
		Timeout:   b.cfg.EffectiveTimeout(),
	}
}

func (b *ClientBuilder) buildBaseURL() string {
	if b.cfg.UsesUnixSocket() {
		return "http://unix"
	}
	return "http://" + b.cfg.Addr
}

// Client is the XDB HTTP client.
type Client struct {
	cfg        *ClientConfig
	httpClient *http.Client
	baseURL    string

	schemaEnabled bool
	tupleEnabled  bool
	recordEnabled bool
	healthEnabled bool
}

// Schemas returns the schema store interface if enabled.
func (c *Client) Schemas() *Client {
	if !c.schemaEnabled {
		return nil
	}
	return c
}

// Tuples returns the tuple store interface if enabled.
func (c *Client) Tuples() *Client {
	if !c.tupleEnabled {
		return nil
	}
	return c
}

// Records returns the record store interface if enabled.
func (c *Client) Records() *Client {
	if !c.recordEnabled {
		return nil
	}
	return c
}

// Ping tests connectivity to the server.
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.EffectiveTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/health", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return c.parseErrorResponse(resp)
	}

	return nil
}

// Health checks the health of the server.
func (c *Client) Health(ctx context.Context) error {
	if !c.healthEnabled {
		return ErrHealthStoreNotConfigured
	}

	var result HealthResponse
	err := c.doRequest(ctx, http.MethodGet, "/v1/health", nil, &result)
	if err != nil {
		return err
	}

	if result.Status != "healthy" {
		if result.StoreHealth.Error != nil {
			return fmt.Errorf("server unhealthy: %s", *result.StoreHealth.Error)
		}
		return fmt.Errorf("server unhealthy")
	}

	return nil
}

// PutSchema creates or updates a schema.
func (c *Client) PutSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	if !c.schemaEnabled {
		return ErrSchemaStoreNotConfigured
	}

	req := &PutSchemaRequest{
		URI:    uri.String(),
		Schema: def,
	}

	var result PutSchemaResponse
	return c.doRequest(ctx, http.MethodPut, "/v1/schemas", req, &result)
}

// GetSchema retrieves a schema by URI.
func (c *Client) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	if !c.schemaEnabled {
		return nil, ErrSchemaStoreNotConfigured
	}

	path := "/v1/schemas/" + uri.Path()
	var result GetSchemaResponse
	err := c.doRequest(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}

	return result.Schema, nil
}

// ListSchemas lists schemas matching the URI pattern.
func (c *Client) ListSchemas(ctx context.Context, uri *core.URI) ([]*schema.Def, error) {
	if !c.schemaEnabled {
		return nil, ErrSchemaStoreNotConfigured
	}

	var result ListSchemasResponse
	err := c.doRequest(ctx, http.MethodGet, "/v1/schemas", &ListSchemasRequest{URI: uri.String()}, &result)
	if err != nil {
		return nil, err
	}

	return result.Schemas, nil
}

// ListNamespaces lists all namespaces.
func (c *Client) ListNamespaces(ctx context.Context) ([]*core.NS, error) {
	if !c.schemaEnabled {
		return nil, ErrSchemaStoreNotConfigured
	}

	var result ListNamespacesResponse
	err := c.doRequest(ctx, http.MethodGet, "/v1/namespaces", nil, &result)
	if err != nil {
		return nil, err
	}

	return result.Namespaces, nil
}

// DeleteSchema deletes a schema by URI.
func (c *Client) DeleteSchema(ctx context.Context, uri *core.URI) error {
	if !c.schemaEnabled {
		return ErrSchemaStoreNotConfigured
	}

	path := "/v1/schemas/" + uri.Path()
	var result DeleteSchemaResponse
	return c.doRequest(ctx, http.MethodDelete, path, nil, &result)
}

// PutTuples creates or updates tuples.
func (c *Client) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
	if !c.tupleEnabled {
		return ErrTupleStoreNotConfigured
	}

	req := x.Map(tuples, func(t *core.Tuple) *Tuple {
		uri := t.URI()
		path := uri.NS().String() + "/" + uri.Schema().String() + "/" + uri.ID().String()
		return &Tuple{
			ID:    path,
			Attr:  t.Attr().String(),
			Value: valueToJSON(t.Value()),
		}
	})

	var result PutTuplesResponse
	return c.doRequest(ctx, http.MethodPut, "/v1/tuples", req, &result)
}

// GetTuples retrieves tuples by URIs.
func (c *Client) GetTuples(ctx context.Context, uris []*core.URI) ([]*core.Tuple, []*core.URI, error) {
	if !c.tupleEnabled {
		return nil, nil, ErrTupleStoreNotConfigured
	}

	req := x.Map(uris, func(uri *core.URI) string {
		return uri.String()
	})

	var result GetTuplesResponse
	err := c.doRequest(ctx, http.MethodGet, "/v1/tuples", req, &result)
	if err != nil {
		return nil, nil, err
	}

	tuples := x.Map(result.Tuples, func(t *Tuple) *core.Tuple {
		return core.NewTuple(t.ID, t.Attr, t.Value)
	})

	missing := x.Map(result.Missing, func(s string) *core.URI {
		return core.MustParseURI(s)
	})

	return tuples, missing, nil
}

// DeleteTuples deletes tuples by URIs.
func (c *Client) DeleteTuples(ctx context.Context, uris []*core.URI) error {
	if !c.tupleEnabled {
		return ErrTupleStoreNotConfigured
	}

	req := x.Map(uris, func(uri *core.URI) string {
		return uri.String()
	})

	var result DeleteTuplesResponse
	return c.doRequest(ctx, http.MethodDelete, "/v1/tuples", req, &result)
}

// PutRecords creates or updates records.
func (c *Client) PutRecords(ctx context.Context, records []*core.Record) error {
	if !c.recordEnabled {
		return ErrRecordStoreNotConfigured
	}

	req := &PutRecordsRequest{
		Records: records,
	}

	var result PutRecordsResponse
	return c.doRequest(ctx, http.MethodPut, "/v1/records", req, &result)
}

// GetRecords retrieves records by URIs.
func (c *Client) GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error) {
	if !c.recordEnabled {
		return nil, nil, ErrRecordStoreNotConfigured
	}

	req := &GetRecordsRequest{
		URIs: x.Map(uris, func(uri *core.URI) string {
			return uri.String()
		}),
	}

	var result GetRecordsResponse
	err := c.doRequest(ctx, http.MethodGet, "/v1/records", req, &result)
	if err != nil {
		return nil, nil, err
	}

	missing := x.Map(result.NotFound, func(s string) *core.URI {
		return core.MustParseURI(s)
	})

	return result.Records, missing, nil
}

// DeleteRecords deletes records by URIs.
func (c *Client) DeleteRecords(ctx context.Context, uris []*core.URI) error {
	if !c.recordEnabled {
		return ErrRecordStoreNotConfigured
	}

	req := &DeleteRecordsRequest{
		URIs: x.Map(uris, func(uri *core.URI) string {
			return uri.String()
		}),
	}

	var result DeleteRecordsResponse
	return c.doRequest(ctx, http.MethodDelete, "/v1/records", req, &result)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body, result any) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.EffectiveTimeout())
	defer cancel()

	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return c.parseErrorResponse(resp)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) parseErrorResponse(resp *http.Response) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read response body", resp.StatusCode)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	switch errResp.Code {
	case "NOT_FOUND":
		return errors.Wrap(store.ErrNotFound, "message", errResp.Message)
	case "SCHEMA_MODE_CHANGED":
		return errors.Wrap(store.ErrSchemaModeChanged, "message", errResp.Message)
	case "FIELD_CHANGE_TYPE":
		return errors.Wrap(store.ErrFieldChangeType, "message", errResp.Message)
	default:
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Message)
	}
}

func valueToJSON(v *core.Value) any {
	if v == nil || v.IsNil() {
		return nil
	}

	switch v.Type().ID() {
	case core.TIDArray:
		arr := v.Unwrap().([]*core.Value)
		result := make([]any, len(arr))
		for i, elem := range arr {
			result[i] = valueToJSON(elem)
		}
		return result
	case core.TIDMap:
		mp := v.Unwrap().(map[*core.Value]*core.Value)
		result := make(map[string]any)
		for k, val := range mp {
			result[k.ToString()] = valueToJSON(val)
		}
		return result
	default:
		return v.Unwrap()
	}
}
