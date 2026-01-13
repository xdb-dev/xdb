// Package api provides the HTTP API layer for XDB, including both server
// and client implementations.
//
// # Server
//
// The server provides HTTP endpoints for schema, tuple, and record operations.
// Use ServerBuilder to construct a server with specific store implementations:
//
//	cfg := &api.ServerConfig{
//	    Addr:       ":8080",
//	    SocketPath: "/var/run/xdb.sock", // Optional Unix socket
//	}
//
//	server, err := api.NewServerBuilder(cfg).
//	    WithSchemaStore(schemaStore).
//	    WithTupleStore(tupleStore).
//	    WithRecordStore(recordStore).
//	    WithHealthStore(healthStore).
//	    Build()
//
// # Client
//
// The Client provides a Go interface for communicating with an XDB server
// over HTTP. It supports both TCP and Unix socket transports.
//
// # Client Configuration
//
// ClientConfig specifies connection parameters:
//
//	// TCP connection
//	cfg := &api.ClientConfig{
//	    Addr:    "localhost:8080",
//	    Timeout: 30 * time.Second,
//	}
//
//	// Unix socket connection (preferred for local deployments)
//	cfg := &api.ClientConfig{
//	    SocketPath: "/var/run/xdb.sock",
//	    Timeout:    30 * time.Second,
//	}
//
// When both Addr and SocketPath are set, Unix socket takes precedence.
// The default timeout is 30 seconds if not specified.
//
// # Builder Pattern
//
// ClientBuilder constructs clients with specific store capabilities enabled.
// Only enabled stores can be used; calling methods on disabled stores returns
// an error:
//
//	client, err := api.NewClientBuilder(cfg).
//	    WithSchemaStore().   // Enable schema operations
//	    WithTupleStore().    // Enable tuple operations
//	    WithRecordStore().   // Enable record operations
//	    WithHealthStore().   // Enable health checks
//	    Build()
//
// At least one store must be enabled, otherwise Build() returns an error.
//
// # Store Operations
//
// The client provides methods that mirror the store interfaces:
//
// Schema operations (requires WithSchemaStore):
//
//	err := client.PutSchema(ctx, uri, def)
//	def, err := client.GetSchema(ctx, uri)
//	defs, err := client.ListSchemas(ctx, uri)
//	namespaces, err := client.ListNamespaces(ctx)
//	err := client.DeleteSchema(ctx, uri)
//
// Tuple operations (requires WithTupleStore):
//
//	err := client.PutTuples(ctx, tuples)
//	tuples, missing, err := client.GetTuples(ctx, uris)
//	err := client.DeleteTuples(ctx, uris)
//
// Record operations (requires WithRecordStore):
//
//	err := client.PutRecords(ctx, records)
//	records, missing, err := client.GetRecords(ctx, uris)
//	err := client.DeleteRecords(ctx, uris)
//
// Health operations (requires WithHealthStore):
//
//	err := client.Health(ctx)
//	err := client.Ping(ctx) // Always available, even without WithHealthStore
//
// # Error Handling
//
// The client maps HTTP error responses back to store errors where applicable:
//
//   - HTTP 404 with code "NOT_FOUND" maps to store.ErrNotFound
//   - HTTP 400 with code "SCHEMA_MODE_CHANGED" maps to store.ErrSchemaModeChanged
//   - HTTP 400 with code "FIELD_CHANGE_TYPE" maps to store.ErrFieldChangeType
//
// Other errors are returned as generic errors with the HTTP status code
// and message.
//
// Example error handling:
//
//	def, err := client.GetSchema(ctx, uri)
//	if errors.Is(err, store.ErrNotFound) {
//	    // Schema does not exist
//	}
//
// # Context and Timeouts
//
// All client methods accept a context for cancellation and deadlines.
// The client applies its configured timeout on top of any context deadline:
//
//	// Request will timeout after 5 seconds, even if client timeout is 30s
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	def, err := client.GetSchema(ctx, uri)
//
// # Transport Selection
//
// TCP transport is suitable for remote servers or containerized deployments.
// Unix socket transport provides better performance for local deployments
// and avoids network overhead:
//
//	// Unix socket - lower latency, no TCP overhead
//	cfg := &api.ClientConfig{SocketPath: "/var/run/xdb.sock"}
//
//	// TCP - required for remote servers
//	cfg := &api.ClientConfig{Addr: "xdb.example.com:8080"}
//
// # Thread Safety
//
// The Client is safe for concurrent use. Multiple goroutines can call
// client methods simultaneously. The underlying http.Client handles
// connection pooling automatically.
//
// # Custom HTTP Client
//
// For advanced use cases, provide a custom http.Client:
//
//	httpClient := &http.Client{
//	    Timeout: 60 * time.Second,
//	    Transport: &http.Transport{
//	        MaxIdleConns:        100,
//	        MaxIdleConnsPerHost: 100,
//	    },
//	}
//
//	client, err := api.NewClientBuilder(cfg).
//	    WithHTTPClient(httpClient).
//	    WithSchemaStore().
//	    Build()
package api
