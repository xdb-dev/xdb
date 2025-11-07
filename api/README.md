# XDB API

HTTP API layer providing RESTful endpoints for XDB operations.

## Structure

```
api/
├── tuple_api.go    # Tuple CRUD endpoints
├── repo_api.go     # Repository management
└── types.go        # Request/response types
```

## Endpoints

### TupleAPI

Handles tuple-level CRUD operations:

- **GetTuples**: Retrieve tuples by URI
- **PutTuples**: Store tuples
- **DeleteTuples**: Remove tuples

### RepoAPI

Handles repository management:

- **MakeRepo**: Create repositories with optional schemas

## Implementation Guide

### Endpoint Pattern

Use the generic `EndpointFunc[Req, Res]` type for type-safe handlers:

```go
type EndpointFunc[Req any, Res any] func(ctx context.Context, req *Req) (*Res, error)
```

### Request/Response Types

Define API-specific types in `types.go`:

Keep API types separate from core types to allow independent evolution.

### Handler Structure

```go
func (a *TupleAPI) PutTuples() EndpointFunc[PutTuplesRequest, PutTuplesResponse] {
    return func(ctx context.Context, req *PutTuplesRequest) (*PutTuplesResponse, error) {
        // Convert API types to core types
        // Call driver methods
        // Return response
    }
}
```

### Error Handling

- Return errors directly from endpoint functions
- Let the HTTP handler layer convert errors to appropriate status codes
- Include context in error messages
