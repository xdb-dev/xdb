# XDB HTTP Client

The XDB HTTP client provides a Go interface for communicating with an XDB server over HTTP. It supports both TCP and Unix socket transports, with a builder pattern for configuring store capabilities.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client                                    │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐     │
│  │ SchemaStore │ TupleStore  │ RecordStore │HealthStore  │     │
│  └──────┬──────┴──────┬──────┴──────┬──────┴──────┬──────┘     │
│         │             │             │             │              │
│         └─────────────┴──────┬──────┴─────────────┘              │
│                              │                                   │
│                    ┌─────────▼─────────┐                        │
│                    │   http.Client     │                        │
│                    └─────────┬─────────┘                        │
│                              │                                   │
│         ┌────────────────────┼────────────────────┐              │
│         │                    │                    │              │
│  ┌──────▼──────┐      ┌──────▼──────┐            │              │
│  │ TCP Transport│      │Unix Socket  │            │              │
│  │ (net.Dialer) │      │ Transport   │            │              │
│  └─────────────┘      └─────────────┘            │              │
└─────────────────────────────────────────────────────────────────┘
```

The client wraps an `http.Client` and provides typed methods for each store operation. Transport selection is automatic based on configuration.

## Configuration

### ClientConfig

| Field        | Type            | Default | Description             |
| ------------ | --------------- | ------- | ----------------------- |
| `Addr`       | `string`        | -       | TCP address (host:port) |
| `SocketPath` | `string`        | -       | Unix socket path        |
| `Timeout`    | `time.Duration` | 30s     | Request timeout         |

Either `Addr` or `SocketPath` must be set. If both are set, Unix socket takes precedence.

### TCP Connection

```go
cfg := &api.ClientConfig{
    Addr:    "localhost:8080",
    Timeout: 30 * time.Second,
}
```

### Unix Socket Connection

```go
cfg := &api.ClientConfig{
    SocketPath: "/var/run/xdb.sock",
    Timeout:    30 * time.Second,
}
```

## Builder Pattern

The `ClientBuilder` constructs clients with specific store capabilities. This pattern ensures type safety and prevents calling methods on disabled stores.

### Basic Usage

```go
client, err := api.NewClientBuilder(cfg).
    WithSchemaStore().
    WithTupleStore().
    WithRecordStore().
    WithHealthStore().
    Build()
```

### Builder Methods

| Method              | Description                     |
| ------------------- | ------------------------------- |
| `WithSchemaStore()` | Enables schema CRUD operations  |
| `WithTupleStore()`  | Enables tuple CRUD operations   |
| `WithRecordStore()` | Enables record CRUD operations  |
| `WithHealthStore()` | Enables health check operations |
| `WithHTTPClient(c)` | Uses custom http.Client         |

### Build Errors

`Build()` returns an error if:

- No configuration is provided
- No stores are enabled
- Neither `Addr` nor `SocketPath` is set

## Store Interface Implementations

### Schema Operations

```go
// Create or update a schema
err := client.PutSchema(ctx, uri, def)

// Retrieve a schema
def, err := client.GetSchema(ctx, uri)

// List schemas in namespace
schemas, err := client.ListSchemas(ctx, nsURI)

// List all namespaces
namespaces, err := client.ListNamespaces(ctx)

// Delete a schema
err := client.DeleteSchema(ctx, uri)
```

### Tuple Operations

```go
// Create or update tuples
err := client.PutTuples(ctx, tuples)

// Retrieve tuples (returns found tuples and missing URIs)
tuples, missing, err := client.GetTuples(ctx, uris)

// Delete tuples
err := client.DeleteTuples(ctx, uris)
```

### Record Operations

```go
// Create or update records
err := client.PutRecords(ctx, records)

// Retrieve records (returns found records and missing URIs)
records, missing, err := client.GetRecords(ctx, uris)

// Delete records
err := client.DeleteRecords(ctx, uris)
```

### Health Operations

```go
// Full health check (requires WithHealthStore)
err := client.Health(ctx)

// Simple connectivity test (always available)
err := client.Ping(ctx)
```

## Error Handling

### Error Mapping

HTTP errors are mapped to store errors where applicable:

| HTTP Status | Error Code            | Store Error                  |
| ----------- | --------------------- | ---------------------------- |
| 404         | `NOT_FOUND`           | `store.ErrNotFound`          |
| 400         | `SCHEMA_MODE_CHANGED` | `store.ErrSchemaModeChanged` |
| 400         | `FIELD_CHANGE_TYPE`   | `store.ErrFieldChangeType`   |
| Other       | -                     | Generic error with message   |

### Error Handling Example

```go
def, err := client.GetSchema(ctx, uri)
if err != nil {
    if errors.Is(err, store.ErrNotFound) {
        // Handle missing schema
        return createDefaultSchema(ctx, uri)
    }
    return err
}
```

### Store Not Configured Errors

Calling methods on disabled stores returns specific errors:

| Error                         | Cause                                            |
| ----------------------------- | ------------------------------------------------ |
| `ErrSchemaStoreNotConfigured` | Schema method called without `WithSchemaStore()` |
| `ErrTupleStoreNotConfigured`  | Tuple method called without `WithTupleStore()`   |
| `ErrRecordStoreNotConfigured` | Record method called without `WithRecordStore()` |
| `ErrHealthStoreNotConfigured` | Health method called without `WithHealthStore()` |

## Transport Selection

### When to Use TCP

- Remote server connections
- Containerized deployments
- Load-balanced environments
- Cross-network communication

### When to Use Unix Socket

- Local server on same machine
- Lower latency requirements
- Higher throughput needs
- Security-sensitive deployments (socket file permissions)

### Performance Comparison

Unix sockets provide:

- No TCP handshake overhead
- No network stack processing
- Direct kernel-to-kernel communication
- Typically 10-30% lower latency for local connections

## Timeout and Context Management

### Request Timeouts

The client applies the configured timeout to each request:

```go
cfg := &api.ClientConfig{
    Addr:    "localhost:8080",
    Timeout: 30 * time.Second, // Applied to each request
}
```

### Context Deadlines

Context deadlines are respected and combined with the client timeout:

```go
// Request uses the shorter of context deadline and client timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

def, err := client.GetSchema(ctx, uri)
```

### Cancellation

Contexts support cancellation for long-running operations:

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(100 * time.Millisecond)
    cancel() // Cancel the request
}()

_, err := client.GetSchema(ctx, uri)
// err will be context.Canceled
```

## Thread Safety

The `Client` is safe for concurrent use from multiple goroutines. The underlying `http.Client` manages connection pooling automatically.

### Concurrent Usage Example

```go
client, _ := api.NewClientBuilder(cfg).
    WithSchemaStore().
    Build()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        uri := core.MustParseURI(fmt.Sprintf("xdb://ns/schema%d", id))
        _, _ = client.GetSchema(ctx, uri)
    }(i)
}
wg.Wait()
```

### Connection Pooling

The default HTTP transport maintains connection pools. For high-throughput applications, consider a custom transport:

```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 100,
    IdleConnTimeout:     90 * time.Second,
}

httpClient := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,
}

client, _ := api.NewClientBuilder(cfg).
    WithHTTPClient(httpClient).
    WithSchemaStore().
    Build()
```

## API Endpoints

The client communicates with these server endpoints:

| Operation      | Method | Endpoint            |
| -------------- | ------ | ------------------- |
| PutSchema      | PUT    | `/v1/schemas`       |
| GetSchema      | GET    | `/v1/schemas/{uri}` |
| ListSchemas    | GET    | `/v1/schemas`       |
| DeleteSchema   | DELETE | `/v1/schemas/{uri}` |
| ListNamespaces | GET    | `/v1/namespaces`    |
| PutTuples      | PUT    | `/v1/tuples`        |
| GetTuples      | GET    | `/v1/tuples`        |
| DeleteTuples   | DELETE | `/v1/tuples`        |
| PutRecords     | PUT    | `/v1/records`       |
| GetRecords     | GET    | `/v1/records`       |
| DeleteRecords  | DELETE | `/v1/records`       |
| Health         | GET    | `/v1/health`        |

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/xdb-dev/xdb/api"
    "github.com/xdb-dev/xdb/core"
    "github.com/xdb-dev/xdb/schema"
)

func main() {
    // Create client with TCP connection
    cfg := &api.ClientConfig{
        Addr:    "localhost:8080",
        Timeout: 30 * time.Second,
    }

    client, err := api.NewClientBuilder(cfg).
        WithSchemaStore().
        WithTupleStore().
        WithHealthStore().
        Build()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Check server health
    if err := client.Health(ctx); err != nil {
        log.Fatal("Server unhealthy:", err)
    }

    // Create a schema
    userSchema := &schema.Def{
        Name:    "users",
        Version: "1.0.0",
        Mode:    schema.ModeDynamic,
        Fields: []*schema.FieldDef{
            {Name: "name", Type: core.TypeString},
            {Name: "email", Type: core.TypeString},
        },
    }

    uri := core.MustParseURI("xdb://com.example/users")
    if err := client.PutSchema(ctx, uri, userSchema); err != nil {
        log.Fatal("Failed to create schema:", err)
    }

    // Store tuples
    tuples := []*core.Tuple{
        core.NewTuple("user-1", "name", "Alice"),
        core.NewTuple("user-1", "email", "alice@example.com"),
    }

    if err := client.PutTuples(ctx, tuples); err != nil {
        log.Fatal("Failed to store tuples:", err)
    }

    log.Println("Successfully created schema and stored tuples")
}
```
