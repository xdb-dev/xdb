# JSON-RPC Server & Client Plan

**Date:** 2026-03-10
**Status:** Draft
**Depends on:** [Agent-First CLI Plan](./2026-03-09-agent-first-cli.md)

## Overview

The XDB daemon exposes a JSON-RPC 2.0 server over HTTP and Unix socket. The CLI is a thin client that translates commands into JSON-RPC calls. This plan covers the protocol design, server architecture, client library, and how the CLI connects to the daemon.

## Why JSON-RPC 2.0

The CLI plan already uses dot-notation RPC methods (`records.create`, `schemas.list`). JSON-RPC 2.0 is a natural fit:

- **Method naming** — `records.create` maps directly to JSON-RPC method strings
- **Batch** — JSON-RPC 2.0 has native batch support (array of request objects). The CLI `batch` command maps 1:1
- **Simplicity** — Small spec, easy to implement. No code generation, no protobuf, no gRPC dependencies
- **Error codes** — Structured error objects with codes and messages
- **ID tracking** — Request/response correlation via `id` field
- **Notifications** — Fire-and-forget calls (no `id`) for future use (e.g., daemon events)

### What We Don't Need

- **gRPC** — Adds protobuf dependency, code generation, HTTP/2 requirement. Overkill for a local daemon
- **REST** — Multiple endpoints, HTTP verbs, URL routing. More surface area, no batch support
- **Custom protocol** — No reason to invent one when JSON-RPC 2.0 fits exactly

---

## Protocol

### Request Format

All requests follow [JSON-RPC 2.0](https://www.jsonrpc.org/specification):

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "records.create",
  "params": {
    "uri": "xdb://com.example/posts/post-123",
    "body": { "title": "Hello", "content": "World" }
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "kind": "Record",
    "uri": "xdb://com.example/posts/post-123",
    "data": { "title": "Hello", "content": "World" }
  }
}
```

### Error Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32001,
    "message": "Record already exists",
    "data": {
      "xdb_code": "ALREADY_EXISTS",
      "uri": "xdb://com.example/posts/post-123"
    }
  }
}
```

### Batch Request

JSON-RPC 2.0 batch = array of requests. The CLI `batch` command sends this directly:

```json
[
  {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "records.create",
    "params": {
      "uri": "xdb://com.example/posts/post-1",
      "body": { "title": "First" }
    }
  },
  {
    "jsonrpc": "2.0",
    "id": 2,
    "method": "records.create",
    "params": {
      "uri": "xdb://com.example/posts/post-2",
      "body": { "title": "Second" }
    }
  },
  {
    "jsonrpc": "2.0",
    "id": 3,
    "method": "records.get",
    "params": {
      "uri": "xdb://com.example/posts/post-1"
    }
  }
]
```

Batch response = array of responses in the same order:

```json
[
  { "jsonrpc": "2.0", "id": 1, "result": { "kind": "Record", "uri": "..." } },
  { "jsonrpc": "2.0", "id": 2, "result": { "kind": "Record", "uri": "..." } },
  { "jsonrpc": "2.0", "id": 3, "result": { "kind": "Record", "uri": "...", "data": { "title": "First" } } }
]
```

### XDB Batch vs JSON-RPC Batch

JSON-RPC 2.0 batch has no transaction semantics — each request is independent. XDB needs atomic batch with rollback. Two options:

1. **`batch.execute` method** — A single JSON-RPC request wrapping an array of operations. Transaction semantics are XDB's responsibility, not the transport's.
2. **JSON-RPC array with `X-XDB-Transaction: true` header** — Uses native batch syntax but opts into atomicity via header.

**Decision: Option 1 — `batch.execute` method.**

Rationale: Transaction semantics are application-level, not transport-level. Putting them in a header conflates concerns. A dedicated method is explicit and works over any transport (including Unix socket where headers aren't natural).

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "batch.execute",
  "params": {
    "operations": [
      { "method": "records.create", "uri": "xdb://com.example/posts/post-1", "body": { "title": "First" } },
      { "method": "records.create", "uri": "xdb://com.example/posts/post-2", "body": { "title": "Second" } },
      { "method": "records.get", "uri": "xdb://com.example/posts/post-1" }
    ],
    "dry_run": false
  }
}
```

Response uses the `BatchResult` envelope from the CLI plan:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "kind": "BatchResult",
    "total": 3,
    "succeeded": 3,
    "failed": 0,
    "results": [
      { "index": 0, "method": "records.create", "uri": "...", "status": "ok" },
      { "index": 1, "method": "records.create", "uri": "...", "status": "ok" },
      { "index": 2, "method": "records.get", "uri": "...", "status": "ok", "data": { "title": "First" } }
    ]
  }
}
```

Non-transactional batch is not supported as a transport-level feature. With `/rpc/{method}` routing, each request targets one method. Use `batch.execute` for all multi-operation needs — set `"atomic": false` in params to opt out of transaction semantics if independent execution is desired.

---

## Methods

Every method takes a `params` object. Method names match the CLI's `resource.method` convention.

### records

| Method | Params | Result | Mutating | Idempotent |
|--------|--------|--------|----------|------------|
| `records.create` | `uri`, `body` | `Record` | Yes | Yes — returns existing if exists |
| `records.get` | `uri`, `fields?` | `Record` | No | Yes |
| `records.list` | `uri`, `filters?`, `query?`, `fields?`, `limit?`, `offset?` | `RecordList` | No | Yes |
| `records.update` | `uri`, `body` | `Record` | Yes | Yes — patch merge, same input = same result |
| `records.upsert` | `uri`, `body` | `Record` | Yes | Yes — full replace |
| `records.delete` | `uri` | `{}` | Yes | Yes — succeeds if not found |

- `create` — if record exists, returns the existing record without modification (no error)
- `update` — **patch semantics**: merges only supplied fields over the existing record. Unspecified fields are preserved. Returns `NOT_FOUND` if record doesn't exist
- `upsert` — **full replace**: sets the complete record state. Creates if not exists
- `delete` — succeeds even if record doesn't exist (no `NOT_FOUND` error)

### schemas

| Method | Params | Result | Mutating | Idempotent |
|--------|--------|--------|----------|------------|
| `schemas.create` | `uri`, `body?` | `Schema` | Yes | Yes — returns existing if exists |
| `schemas.get` | `uri` | `Schema` | No | Yes |
| `schemas.list` | `uri`, `limit?`, `offset?` | `SchemaList` | No | Yes |
| `schemas.update` | `uri`, `body` | `Schema` | Yes | Yes — patch merge |
| `schemas.delete` | `uri` | `{}` | Yes | Yes — succeeds if not found |

- `create` — if schema exists, returns the existing schema without modification
- `update` — **patch semantics**: merges supplied attrs over existing definition
- `delete` — succeeds even if schema doesn't exist. Returns `VALIDATION_ERROR` if records exist (use batch with `--cascade`)

### namespaces

| Method | Params | Result | Mutating |
|--------|--------|--------|----------|
| `namespaces.list` | `limit?`, `offset?` | `NamespaceList` | No |
| `namespaces.get` | `uri` | `Namespace` | No |

### batch

| Method | Params | Result | Mutating |
|--------|--------|--------|----------|
| `batch.execute` | `operations`, `dry_run?` | `BatchResult` | Yes |

### introspection

| Method | Params | Result | Mutating |
|--------|--------|--------|----------|
| `schema.describe_method` | `method` | `MethodDescription` | No |
| `schema.describe_type` | `type` | `TypeDescription` | No |
| `schema.describe_data` | `uri` | `DataSchemaDescription` | No |
| `schema.list_methods` | — | `MethodList` | No |
| `schema.list_types` | — | `TypeList` | No |

### watch

| Method | Params | Result | Mutating |
|--------|--------|--------|----------|
| `watch.subscribe` | `uri` | Streaming `WatchEvent` (NDJSON) | No |

The watch method uses a long-lived HTTP connection. Events are streamed as NDJSON lines (`Content-Type: application/x-ndjson`). The connection closes when the client disconnects or the watched schema is deleted.

### system

| Method | Params | Result | Mutating |
|--------|--------|--------|----------|
| `system.status` | — | `ServerStatus` | No |
| `system.context` | — | `AgentContext` (markdown string) | No |
| `system.skills` | `name?` | `Skill` or `SkillList` | No |

---

## Error Codes

XDB uses the JSON-RPC 2.0 error code ranges:

### Standard JSON-RPC errors (-32700 to -32600)

| Code | Constant | Meaning |
|------|----------|---------|
| -32700 | `PARSE_ERROR` | Invalid JSON |
| -32600 | `INVALID_REQUEST` | Not a valid JSON-RPC request |
| -32601 | `METHOD_NOT_FOUND` | Method does not exist |
| -32602 | `INVALID_PARAMS` | Invalid method parameters |
| -32603 | `INTERNAL_ERROR` | Server internal error |

### XDB application errors (-32001 to -32099)

| Code | Constant | Meaning |
|------|----------|---------|
| -32001 | `NOT_FOUND` | Record/schema not found (only from `update` — `create`/`delete` are idempotent) |
| -32002 | `ALREADY_EXISTS` | Reserved (not returned by idempotent `create`, kept for future use) |
| -32003 | `INVALID_URI` | URI failed validation |
| -32004 | `SCHEMA_VIOLATION` | Payload violates strict schema |
| -32005 | `PAYLOAD_TOO_LARGE` | Payload exceeds size limit |
| -32006 | `BATCH_LIMIT_EXCEEDED` | Too many operations in batch |
| -32007 | `BATCH_ROLLED_BACK` | Batch transaction rolled back |
| -32008 | `VALIDATION_ERROR` | Generic input validation failure |

All error responses include `data` with `xdb_code` (string constant) and contextual fields:

```json
{
  "code": -32003,
  "message": "URI path contains path traversal sequence '..'",
  "data": {
    "xdb_code": "INVALID_URI",
    "field": "uri",
    "value": "xdb://../../.ssh/keys"
  }
}
```

---

## Transport

### HTTP

The server uses method-scoped endpoints. The method name from the JSON-RPC request body is also encoded in the URL path:

```
POST http://localhost:8147/rpc/records.create
POST http://localhost:8147/rpc/schemas.list
POST http://localhost:8147/rpc/batch.execute
Content-Type: application/json
```

The server extracts the method from the URL path, not the request body. The `method` field in the JSON-RPC body is still present (spec compliance) but the path is authoritative.

**Why `/rpc/{method}` instead of `/rpc`:**

- **Observability** — HTTP access logs, metrics dashboards, and tracing tools show the method in the URL. A single `/rpc` endpoint makes all calls look identical in logs and metrics. With `/rpc/{method}`, you get per-method latency percentiles, error rates, and throughput for free from any standard HTTP observability tool (nginx logs, Prometheus `http_request_duration_seconds`, OpenTelemetry spans).
- **Rate limiting** — Middleware can rate-limit by path pattern (e.g., throttle `records.list` differently from `records.create`) without inspecting request bodies.
- **Access control** — If auth is added later, policies can be path-based (e.g., allow reads, deny mutations) using standard HTTP middleware.

Requests to `/rpc` without a method suffix return `405 Method Not Allowed` with a helpful error.

#### Health check

```
GET http://localhost:8147/health
```

Returns `200 OK` with `{"status": "ok"}`. Used by the CLI `daemon status` command and monitoring.

### Unix Socket

Same JSON-RPC protocol over a Unix domain socket at `~/.xdb/xdb.sock`. The CLI prefers the socket when available (lower latency, no TCP overhead).

The server serves HTTP over the Unix socket (same `net/http` handler). The client connects via `http.Client` with a custom `Dialer` that dials the socket path. This means the same HTTP handler code serves both TCP and Unix socket — no separate protocol implementation.

### Transport Selection (Client)

The client tries transports in order:

1. **Unix socket** — if `~/.xdb/xdb.sock` exists and is connectable
2. **HTTP** — fall back to `localhost:8147`

The `--addr` flag or config `daemon.addr` overrides to force HTTP. The `--socket` flag or config `daemon.socket` overrides the socket path.

---

## Server Architecture

```
┌─────────────────────────────────────────────┐
│                  Daemon                      │
│                                              │
│  ┌──────────┐    ┌───────────┐              │
│  │ HTTP TCP │    │ HTTP Unix │              │
│  │ :8147    │    │ xdb.sock  │              │
│  └────┬─────┘    └─────┬─────┘              │
│       └───────┬────────┘                     │
│               ↓                              │
│       ┌───────────────┐                      │
│       │  RPC Handler  │                      │
│       │  (dispatch)   │                      │
│       └───────┬───────┘                      │
│               ↓                              │
│       ┌───────────────┐                      │
│       │   Router      │                      │
│       │ method→handler│                      │
│       └───────┬───────┘                      │
│               ↓                              │
│       ┌───────────────┐                      │
│       │   Service     │                      │
│       │   Layer       │                      │
│       └───────┬───────┘                      │
│               ↓                              │
│       ┌───────────────┐                      │
│       │   Store       │                      │
│       │  Interface    │                      │
│       └───────────────┘                      │
└─────────────────────────────────────────────┘
```

### Components

#### RPC Handler (`internal/rpc/handler.go`)

Single HTTP handler mounted at `/rpc/{method}` that:

1. Extracts method name from URL path
2. Reads and parses request body as JSON-RPC 2.0
3. Dispatches to router using the path method
4. Writes JSON-RPC response

For `batch.execute`, delegates to the batch handler which manages the transaction.

```go
type Handler struct {
    router *Router
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Parse, dispatch, respond
}
```

#### Router (`internal/rpc/router.go`)

Maps method strings to handler functions:

```go
type MethodHandler func(ctx context.Context, params json.RawMessage) (any, *Error)

type Router struct {
    methods map[string]MethodHandler
}

func (r *Router) Register(method string, handler MethodHandler) {
    r.methods[method] = handler
}
```

Registration happens at startup:

```go
router := rpc.NewRouter()

// Records
router.Register("records.create", svc.RecordsCreate)
router.Register("records.get", svc.RecordsGet)
router.Register("records.list", svc.RecordsList)
router.Register("records.update", svc.RecordsUpdate)
router.Register("records.upsert", svc.RecordsUpsert)
router.Register("records.delete", svc.RecordsDelete)

// Schemas
router.Register("schemas.create", svc.SchemasCreate)
// ...

// Batch
router.Register("batch.execute", svc.BatchExecute)
```

#### Service Layer (`internal/service/`)

Business logic between RPC handlers and the store. Handles:

- Parameter validation (URI, payload, limits)
- Schema enforcement (strict schema checks)
- Patch merge logic (`update` reads existing record, merges supplied fields, writes back)
- Idempotent create (check existence, return existing if found)
- Idempotent delete (suppress `ErrNotFound`)
- Dry-run logic
- Field masking (`fields` parameter)
- Batch transaction coordination

```go
type Service struct {
    store store.Store
}

func (s *Service) RecordsCreate(
    ctx context.Context,
    params json.RawMessage,
) (any, *rpc.Error) {
    var p RecordsCreateParams
    if err := json.Unmarshal(params, &p); err != nil {
        return nil, rpc.InvalidParams(err.Error())
    }
    // Validate URI, check schema, create record
}
```

The service layer is the **only** consumer of the store interface. RPC handlers don't touch the store directly.

#### Store Interface (`store/store.go`)

Already exists with 4 backend implementations (memory, SQLite, Redis, filesystem). The server depends on it but doesn't define it. Pre-work adds `Close() error` to the interface for daemon shutdown and `Fields []string` to `ListQuery` for store-level field projection.

---

## Client Library

### `xdb/client` Package

A Go client that speaks JSON-RPC 2.0 to the daemon. Used by the CLI and available as a library for Go programs.

```go
// Connect to daemon (auto-detects socket vs HTTP)
c, err := client.New(client.WithConfig(cfg))

// Or explicit transport
c, err := client.New(client.WithSocket("/path/to/xdb.sock"))
c, err := client.New(client.WithAddr("localhost:8147"))
```

#### Typed Methods

The client provides typed methods that mirror the RPC surface:

```go
// Records
record, err := c.Records.Create(ctx, uri, body)
record, err := c.Records.Get(ctx, uri, client.WithFields("title", "author.id"))
list, err := c.Records.List(ctx, uri, client.WithLimit(10), client.WithFilter("age>30"))
record, err := c.Records.Update(ctx, uri, body)
record, err := c.Records.Upsert(ctx, uri, body)
err = c.Records.Delete(ctx, uri)

// Schemas
schema, err := c.Schemas.Create(ctx, uri, body)
schema, err := c.Schemas.Get(ctx, uri)
list, err := c.Schemas.List(ctx, uri)

// Namespaces
list, err := c.Namespaces.List(ctx)
ns, err := c.Namespaces.Get(ctx, uri)

// Batch
result, err := c.Batch.Execute(ctx, ops, client.WithDryRun())

// System
status, err := c.System.Status(ctx)
```

#### Raw RPC Call

For advanced use or methods not covered by typed wrappers:

```go
var result json.RawMessage
err := c.Call(ctx, "records.create", params, &result)
```

#### Non-Atomic Batch

For independent operations (no rollback on failure):

```go
result, err := c.Batch.Execute(ctx, ops, client.WithAtomic(false))
```

Each operation executes independently — failures don't roll back other operations. Use `c.Batch.Execute()` without `WithAtomic(false)` for transactional batch (default).

### Internal Types

```go
// client/types.go

type Record struct {
    Kind string         `json:"kind"`
    URI  string         `json:"uri"`
    Data map[string]any `json:"data"`
}

type RecordList struct {
    Kind       string   `json:"kind"`
    URI        string   `json:"uri"`
    Items      []Record `json:"items"`
    NextOffset int      `json:"next_offset,omitempty"`
    Total      int      `json:"total"`
}

type BatchResult struct {
    Kind      string         `json:"kind"`
    Total     int            `json:"total"`
    Succeeded int            `json:"succeeded"`
    Failed    int            `json:"failed"`
    RolledBack bool          `json:"rolled_back,omitempty"`
    Results   []BatchEntry   `json:"results"`
}
```

These are the **wire types** — they match the JSON envelope format from the CLI plan. The `core` package types (Record, Tuple, etc.) are internal domain types. The service layer converts between them.

---

## CLI ↔ Daemon Integration

### How the CLI Uses the Client

Each CLI command constructs a JSON-RPC call via the client. The CLI uses urfave/cli v3:

```go
// internal/cli/records.go

func (a *App) recordsCreateCmd() *cli.Command {
    return &cli.Command{
        Name: "create",
        Action: func(ctx context.Context, cmd *cli.Command) error {
            // Parse flags
            uri := cmd.String("uri")
            body := readInput(cmd) // --json, --file, or stdin

            if cmd.Bool("dry-run") {
                result, err := a.client.Records.Create(ctx, uri, body, client.WithDryRun())
                return a.output.Write(result)
            }

            record, err := a.client.Records.Create(ctx, uri, body)
            return a.output.Write(record)
        },
    }
}
```

The CLI always talks to a running daemon. If the daemon is not running, the client returns `client.ErrNotRunning` and the CLI exits with code 2: `"daemon not running — start it with: xdb daemon start"`.

---

## Daemon Lifecycle

### Process Management

The daemon is a long-running process managed by the CLI:

```bash
xdb daemon start     # Start in background, write PID to ~/.xdb/xdb.pid
xdb daemon stop      # Send SIGTERM, wait for graceful shutdown
xdb daemon status    # Check if running (PID file + health check)
xdb daemon restart   # stop + start
```

### Startup Sequence

1. Read config from `~/.xdb/config.json`
2. Initialize store backend (memory, sqlite, redis, fs)
3. Build service layer with store
4. Build RPC router with service
5. Start HTTP listeners (TCP + Unix socket)
6. Write PID file
7. Log startup message

### Shutdown Sequence

1. Receive SIGTERM or SIGINT
2. Stop accepting new connections
3. Wait for in-flight requests to complete (timeout: 10s)
4. Close store (flush, close connections)
5. Remove PID file and socket file
6. Exit

### PID File

Location: `~/.xdb/xdb.pid`

The CLI checks the PID file before starting. If a PID file exists and the process is still running, `daemon start` fails with a clear error. If the PID file exists but the process is gone (stale PID), the CLI removes the stale file and starts normally.

---

## Package Structure

All `internal/` packages live inside the `cmd/xdb/` module (separate Go module from root). The `client/` package lives in the root module.

```
# Root module (github.com/xdb-dev/xdb)
client/
  client.go            # Client struct, transport selection
  records.go           # Records typed methods
  schemas.go           # Schemas typed methods
  namespaces.go        # Namespaces typed methods
  batch.go             # Batch typed methods
  system.go            # System typed methods
  transport.go         # HTTP + Unix socket transport
  types.go             # Wire types (Record, RecordList, etc.)

# CLI/daemon module (github.com/xdb-dev/xdb/cmd/xdb)
cmd/xdb/internal/rpc/
  request.go           # JSON-RPC 2.0 request/response types
  handler.go           # HTTP handler (parse, dispatch, respond)
  router.go            # Method → handler mapping
  errors.go            # Error codes and constructors

cmd/xdb/internal/service/
  service.go           # Service struct, constructor
  records.go           # records.* method handlers
  schemas.go           # schemas.* method handlers
  namespaces.go        # namespaces.* method handlers
  batch.go             # batch.execute handler
  watch.go             # watch pubsub fan-out
  introspect.go        # schema.describe_* handlers
  system.go            # system.* handlers

cmd/xdb/internal/daemon/
  daemon.go            # Daemon lifecycle (start, stop, listeners)
  pid.go               # PID file management
```

---

## Implementation Phases

### Phase 1: RPC Core

1. `internal/rpc/request.go` — JSON-RPC 2.0 request/response types, parsing, serialization
2. `internal/rpc/errors.go` — Error codes, constructors, XDB error mapping
3. `internal/rpc/router.go` — Method registry, dispatch
4. `internal/rpc/handler.go` — HTTP handler (single request + JSON-RPC array batch)

### Phase 2: Service Layer

5. `internal/service/service.go` — Service struct with store dependency
6. `internal/service/records.go` — records CRUD handlers
7. `internal/service/schemas.go` — schemas CRUD handlers
8. `internal/service/namespaces.go` — namespaces handlers
9. `internal/service/batch.go` — `batch.execute` with transaction coordination

### Phase 3: Daemon

10. `internal/daemon/daemon.go` — TCP + Unix socket listeners, graceful shutdown
11. `internal/daemon/pid.go` — PID file management, stale detection

### Phase 4: Client

12. `client/transport.go` — HTTP + Unix socket transport, auto-detection
13. `client/client.go` — Client struct, `Call()`, `BatchCall()`
14. `client/records.go` + `schemas.go` + `namespaces.go` + `batch.go` + `system.go` — Typed method wrappers

### Phase 5: Integration

16. Wire CLI commands to client (replace any direct store calls)
17. `xdb daemon start/stop/status/restart` commands
18. Integration tests: CLI → daemon → store → response

---

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Protocol | JSON-RPC 2.0 | Method naming matches CLI's `resource.method`, native batch, minimal spec |
| Transactional batch | `batch.execute` method | Transaction semantics are application-level, not transport-level |
| Non-transactional batch | `batch.execute` with `atomic: false` | No JSON-RPC array batch — incompatible with `/rpc/{method}` routing |
| HTTP endpoint | `POST /rpc/{method}` | Method in URL path for free o11y (per-method metrics, logs, traces) |
| Unix socket transport | HTTP over UDS | Same handler code for both transports, `net/http` does the heavy lifting |
| Client interface | Typed methods + raw `Call()` | Type safety for common ops, escape hatch for advanced use |
| Wire types vs domain types | Separate packages | `client/types.go` for JSON wire format, `core/` for domain logic |
| PID management | File-based with stale detection | Simple, no external deps, standard Unix pattern |

---

## What This Plan Does NOT Cover

- **Authentication/authorization** — Local daemon, no auth needed. Add later if XDB supports remote servers.
- **TLS** — Localhost only. Unix socket is preferred transport.
- **WebSocket streaming** — Watch uses long-lived HTTP with NDJSON streaming, not WebSocket.
- **MCP integration** — Explicitly excluded from scope. Can be added as a thin adapter over the RPC router later.
- **Store interface design** — Store interfaces and 4 backends already exist. This plan consumes them.
- **Embedded mode** — Removed. The CLI always talks to a running daemon via JSON-RPC. No in-process store fallback.
