// Package main demonstrates the XDB HTTP client usage.
//
// This example shows how to:
//   - Configure the client for TCP and Unix socket connections
//   - Use the builder pattern to enable multiple stores
//   - Perform basic CRUD operations on schemas and tuples
//   - Handle errors and use context for timeouts
//
// Run this example with an XDB server running on localhost:8080:
//
//	go run main.go
//
// Or with a Unix socket:
//
//	XDB_SOCKET_PATH=/var/run/xdb.sock go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Choose connection type based on environment
	client, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create a context with timeout for all operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Verify server connectivity
	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("server ping failed: %w", err)
	}
	fmt.Println("Connected to XDB server")

	// Check server health (requires health store enabled)
	if err := client.Health(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	fmt.Println("Server is healthy")

	// Demonstrate schema operations
	if err := schemaOperations(ctx, client); err != nil {
		return fmt.Errorf("schema operations failed: %w", err)
	}

	// Demonstrate tuple operations
	if err := tupleOperations(ctx, client); err != nil {
		return fmt.Errorf("tuple operations failed: %w", err)
	}

	// Demonstrate record operations
	if err := recordOperations(ctx, client); err != nil {
		return fmt.Errorf("record operations failed: %w", err)
	}

	// Demonstrate error handling patterns
	if err := errorHandlingPatterns(ctx, client); err != nil {
		return fmt.Errorf("error handling demo failed: %w", err)
	}

	return nil
}

func createClient() (*api.Client, error) {
	// Option 1: TCP connection (default for most deployments)
	tcpConfig := &api.ClientConfig{
		Addr:    "localhost:8080",
		Timeout: 30 * time.Second,
	}

	// Option 2: Unix socket connection (for local deployments)
	socketPath := os.Getenv("XDB_SOCKET_PATH")
	if socketPath != "" {
		return api.NewClientBuilder(&api.ClientConfig{
			SocketPath: socketPath,
			Timeout:    30 * time.Second,
		}).
			WithSchemaStore().
			WithTupleStore().
			WithRecordStore().
			WithHealthStore().
			Build()
	}

	// Use TCP connection with all stores enabled
	return api.NewClientBuilder(tcpConfig).
		WithSchemaStore().
		WithTupleStore().
		WithRecordStore().
		WithHealthStore().
		Build()
}

func schemaOperations(ctx context.Context, client *api.Client) error {
	fmt.Println("\n--- Schema Operations ---")

	// Define a schema for user records
	userSchema := &schema.Def{
		Name:        "users",
		Description: "User account information",
		Version:     "1.0.0",
		Mode:        schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
			{Name: "email", Type: core.TypeString},
			{Name: "age", Type: core.TypeInt},
			{Name: "active", Type: core.TypeBool},
		},
	}

	// Create or update the schema
	schemaURI := core.MustParseURI("xdb://com.example/users")
	if err := client.PutSchema(ctx, schemaURI, userSchema); err != nil {
		return fmt.Errorf("put schema failed: %w", err)
	}
	fmt.Printf("Created schema: %s\n", schemaURI)

	// Retrieve the schema
	retrieved, err := client.GetSchema(ctx, schemaURI)
	if err != nil {
		return fmt.Errorf("get schema failed: %w", err)
	}
	fmt.Printf("Retrieved schema: %s (version %s, %d fields)\n",
		retrieved.Name, retrieved.Version, len(retrieved.Fields))

	// List all schemas in namespace
	nsURI := core.MustParseURI("xdb://com.example")
	schemas, err := client.ListSchemas(ctx, nsURI)
	if err != nil {
		return fmt.Errorf("list schemas failed: %w", err)
	}
	fmt.Printf("Found %d schemas in namespace\n", len(schemas))

	// List all namespaces
	namespaces, err := client.ListNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("list namespaces failed: %w", err)
	}
	fmt.Printf("Found %d namespaces\n", len(namespaces))

	return nil
}

func tupleOperations(ctx context.Context, client *api.Client) error {
	fmt.Println("\n--- Tuple Operations ---")

	// Create tuples for a user record
	userID := "user-123"
	tuples := []*core.Tuple{
		core.NewTuple(userID, "name", "Alice Smith"),
		core.NewTuple(userID, "email", "alice@example.com"),
		core.NewTuple(userID, "age", 28),
		core.NewTuple(userID, "active", true),
	}

	// Store the tuples
	if err := client.PutTuples(ctx, tuples); err != nil {
		return fmt.Errorf("put tuples failed: %w", err)
	}
	fmt.Printf("Stored %d tuples for user %s\n", len(tuples), userID)

	// Retrieve specific tuples by URI
	uris := []*core.URI{
		core.MustParseURI("xdb://com.example/users/" + userID + "#name"),
		core.MustParseURI("xdb://com.example/users/" + userID + "#email"),
	}

	retrieved, missing, err := client.GetTuples(ctx, uris)
	if err != nil {
		return fmt.Errorf("get tuples failed: %w", err)
	}
	fmt.Printf("Retrieved %d tuples, %d missing\n", len(retrieved), len(missing))

	for _, t := range retrieved {
		fmt.Printf("  %s = %v\n", t.Attr(), t.Value().Unwrap())
	}

	// Update a tuple
	updateTuples := []*core.Tuple{
		core.NewTuple(userID, "age", 29),
	}
	if err := client.PutTuples(ctx, updateTuples); err != nil {
		return fmt.Errorf("update tuple failed: %w", err)
	}
	fmt.Println("Updated user age")

	// Delete tuples
	deleteURIs := []*core.URI{
		core.MustParseURI("xdb://com.example/users/" + userID + "#active"),
	}
	if err := client.DeleteTuples(ctx, deleteURIs); err != nil {
		return fmt.Errorf("delete tuples failed: %w", err)
	}
	fmt.Println("Deleted active tuple")

	return nil
}

// errorHandlingPatterns shows how to handle common client errors.
func errorHandlingPatterns(ctx context.Context, client *api.Client) error {
	fmt.Println("\n--- Error Handling ---")

	// Attempting to get a non-existent schema demonstrates store.ErrNotFound
	nonExistentURI := core.MustParseURI("xdb://com.example/nonexistent")
	_, err := client.GetSchema(ctx, nonExistentURI)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Println("Schema not found (expected)")
		} else {
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	// Context cancellation handling
	shortCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond)
	if err := client.Ping(shortCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Request timed out (expected)")
		} else {
			fmt.Printf("Timeout error: %v\n", err)
		}
	}

	return nil
}

// recordOperations shows record store usage.
func recordOperations(ctx context.Context, client *api.Client) error {
	fmt.Println("\n--- Record Operations ---")

	// Create a record with multiple tuples using the builder pattern
	record := core.NewRecord("com.example", "users", "user-456").
		Set("name", "Bob Jones").
		Set("email", "bob@example.com")

	// Store records
	if err := client.PutRecords(ctx, []*core.Record{record}); err != nil {
		return fmt.Errorf("put records failed: %w", err)
	}
	fmt.Println("Stored record")

	// Retrieve records
	records, missing, err := client.GetRecords(ctx, []*core.URI{record.URI()})
	if err != nil {
		return fmt.Errorf("get records failed: %w", err)
	}
	fmt.Printf("Retrieved %d records, %d missing\n", len(records), len(missing))

	// Delete records
	if err := client.DeleteRecords(ctx, []*core.URI{record.URI()}); err != nil {
		return fmt.Errorf("delete records failed: %w", err)
	}
	fmt.Println("Deleted record")

	return nil
}
