// Package client provides an HTTP client for communicating with an XDB server.
//
// The Client supports both TCP and Unix socket transports, with a builder
// pattern for configuring store capabilities.
//
// # Configuration
//
// Config specifies connection parameters:
//
//	// TCP connection
//	cfg := &client.Config{
//	    Addr:    "localhost:8080",
//	    Timeout: 30 * time.Second,
//	}
//
//	// Unix socket connection (preferred for local deployments)
//	cfg := &client.Config{
//	    SocketPath: "/var/run/xdb.sock",
//	    Timeout:    30 * time.Second,
//	}
//
// When both Addr and SocketPath are set, Unix socket takes precedence.
// The default timeout is 30 seconds if not specified.
//
// # Builder Pattern
//
// Builder constructs clients with specific store capabilities enabled.
// Only enabled stores can be used; calling methods on disabled stores returns
// an error:
//
//	c, err := client.NewBuilder(cfg).
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
//	err := c.PutSchema(ctx, uri, def)
//	def, err := c.GetSchema(ctx, uri)
//	defs, err := c.ListSchemas(ctx, uri)
//	namespaces, err := c.ListNamespaces(ctx)
//	err := c.DeleteSchema(ctx, uri)
//
// Tuple operations (requires WithTupleStore):
//
//	err := c.PutTuples(ctx, tuples)
//	tuples, missing, err := c.GetTuples(ctx, uris)
//	err := c.DeleteTuples(ctx, uris)
//
// Record operations (requires WithRecordStore):
//
//	err := c.PutRecords(ctx, records)
//	records, missing, err := c.GetRecords(ctx, uris)
//	err := c.DeleteRecords(ctx, uris)
//
// Health operations (requires WithHealthStore):
//
//	err := c.Health(ctx)
//	err := c.Ping(ctx) // Always available, even without WithHealthStore
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
// # Thread Safety
//
// The Client is safe for concurrent use. Multiple goroutines can call
// client methods simultaneously. The underlying http.Client handles
// connection pooling automatically.
package client
