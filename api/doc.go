// Package api provides the HTTP API layer for XDB, including server
// implementation and shared request/response types.
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
// The HTTP client has been extracted to the [github.com/xdb-dev/xdb/client] package.
package api
