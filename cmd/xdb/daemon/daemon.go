// Package daemon manages the XDB daemon lifecycle.
package daemon

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/rpc"
	"github.com/xdb-dev/xdb/store"
)

// Config holds daemon configuration.
type Config struct {
	SocketPath string
	LogFile    string
	Version    string
}

// Daemon manages the XDB daemon process.
type Daemon struct {
	listener net.Listener
	server   *http.Server
	config   Config
}

// New creates a new [Daemon] with the given configuration.
func New(cfg Config) *Daemon {
	return &Daemon{config: cfg}
}

// PIDPath derives the PID file path from a socket path by replacing the
// extension with .pid.
func PIDPath(socketPath string) string {
	return strings.TrimSuffix(socketPath, filepath.Ext(socketPath)) + ".pid"
}

// IsProcessAlive checks whether a process with the given PID is running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return p.Signal(syscall.Signal(0)) == nil
}

// NewRouter creates a [rpc.Router] with all services registered.
func NewRouter(s store.Store, version string) *rpc.Router {
	r := rpc.NewRouter()

	registerRecords(r, api.NewRecordService(s))
	registerSchemas(r, api.NewSchemaService(s))
	registerNamespaces(r, api.NewNamespaceService(s))
	registerBatch(r, api.NewBatchService(s))
	registerWatch(r, api.NewWatchService(s))
	registerSystem(r, api.NewSystemService(version))

	// Introspection (registered last so it can see all other methods).
	introspect := api.NewIntrospectService(r)
	rpc.RegisterHandler(r, "introspect.method", introspect.DescribeMethod)
	rpc.RegisterHandler(r, "introspect.type", introspect.DescribeType)
	rpc.RegisterHandler(r, "introspect.methods", introspect.ListMethods)
	rpc.RegisterHandler(r, "introspect.types", introspect.ListTypes)

	return r
}

func registerRecords(r *rpc.Router, svc *api.RecordService) {
	rpc.RegisterHandlerWithMeta(r, "records.create", svc.Create, rpc.MethodMeta{
		Description: "Create a new record. Idempotent: returns existing if already exists.",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"uri":  {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
			"data": {Description: "Record data as JSON object", Type: "object"},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The created or existing record data", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "records.get", svc.Get, rpc.MethodMeta{
		Description: "Retrieve a record by URI.",
		Parameters: map[string]rpc.ParamMeta{
			"uri":    {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
			"fields": {Description: "Field projection list", Type: "array"},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The record data", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "records.list", svc.List, rpc.MethodMeta{
		Description: "List records matching a query.",
		Parameters: map[string]rpc.ParamMeta{
			"uri":    {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
			"filter": {Description: "CEL filter expression", Type: "string"},
			"fields": {Description: "Field projection list", Type: "array"},
			"limit":  {Description: "Max items per page", Type: "integer"},
			"offset": {Description: "Page offset", Type: "integer"},
		},
		Response: map[string]rpc.ParamMeta{
			"items":       {Description: "Matching records", Type: "array"},
			"next_offset": {Description: "Offset for next page", Type: "integer"},
			"total":       {Description: "Total matching records", Type: "integer"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "records.update", svc.Update, rpc.MethodMeta{
		Description: "Update an existing record (patch semantics). Only supplied fields change.",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"uri":  {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
			"data": {Description: "Patch data as JSON object", Type: "object", Required: true},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The updated record data", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "records.upsert", svc.Upsert, rpc.MethodMeta{
		Description: "Create or replace a record (full replace). Sets complete state.",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"uri":  {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
			"data": {Description: "Record data as JSON object", Type: "object"},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The upserted record data", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "records.delete", svc.Delete, rpc.MethodMeta{
		Description: "Delete a record. Idempotent: succeeds even if not found.",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"uri": {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
		},
	})
}

func registerSchemas(r *rpc.Router, svc *api.SchemaService) {
	rpc.RegisterHandlerWithMeta(r, "schemas.create", svc.Create, rpc.MethodMeta{
		Description: "Create a new schema definition. Idempotent: returns existing if already exists.",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"uri":  {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
			"data": {Description: "Schema definition as JSON object", Type: "object"},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The created or existing schema definition", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "schemas.get", svc.Get, rpc.MethodMeta{
		Description: "Retrieve a schema definition by URI.",
		Parameters: map[string]rpc.ParamMeta{
			"uri": {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The schema definition", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "schemas.list", svc.List, rpc.MethodMeta{
		Description: "List schemas in a namespace.",
		Parameters: map[string]rpc.ParamMeta{
			"uri":    {Description: "Namespace URI (xdb://ns)", Type: "string"},
			"limit":  {Description: "Max items per page", Type: "integer"},
			"offset": {Description: "Page offset", Type: "integer"},
		},
		Response: map[string]rpc.ParamMeta{
			"items":       {Description: "Schema definitions", Type: "array"},
			"next_offset": {Description: "Offset for next page", Type: "integer"},
			"total":       {Description: "Total schemas", Type: "integer"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "schemas.update", svc.Update, rpc.MethodMeta{
		Description: "Update a schema definition (patch semantics).",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"uri":  {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
			"data": {Description: "Patch data as JSON object", Type: "object", Required: true},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The updated schema definition", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "schemas.delete", svc.Delete, rpc.MethodMeta{
		Description: "Delete a schema. Idempotent: succeeds even if not found.",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"uri":     {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
			"cascade": {Description: "Delete all records in the schema", Type: "boolean"},
		},
	})
}

func registerNamespaces(r *rpc.Router, svc *api.NamespaceService) {
	rpc.RegisterHandlerWithMeta(r, "namespaces.get", svc.Get, rpc.MethodMeta{
		Description: "Retrieve namespace metadata by URI.",
		Parameters: map[string]rpc.ParamMeta{
			"uri": {Description: "Namespace URI (xdb://ns)", Type: "string", Required: true},
		},
		Response: map[string]rpc.ParamMeta{
			"data": {Description: "The namespace metadata", Type: "object"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "namespaces.list", svc.List, rpc.MethodMeta{
		Description: "List all known namespaces.",
		Parameters: map[string]rpc.ParamMeta{
			"limit":  {Description: "Max items per page", Type: "integer"},
			"offset": {Description: "Page offset", Type: "integer"},
		},
		Response: map[string]rpc.ParamMeta{
			"items":       {Description: "Namespaces", Type: "array"},
			"next_offset": {Description: "Offset for next page", Type: "integer"},
			"total":       {Description: "Total namespaces", Type: "integer"},
		},
	})
}

func registerBatch(r *rpc.Router, svc *api.BatchService) {
	rpc.RegisterHandlerWithMeta(r, "batch.execute", svc.Execute, rpc.MethodMeta{
		Description: "Run multiple operations in a single atomic transaction.",
		Mutating:    true,
		Parameters: map[string]rpc.ParamMeta{
			"operations": {Description: "Array of operations to execute", Type: "array", Required: true},
			"dry_run":    {Description: "Validate without writing", Type: "boolean"},
		},
		Response: map[string]rpc.ParamMeta{
			"total":     {Description: "Total operations", Type: "integer"},
			"succeeded": {Description: "Successful operations", Type: "integer"},
			"failed":    {Description: "Failed operations", Type: "integer"},
			"results":   {Description: "Per-operation results", Type: "array"},
		},
	})
}

func registerWatch(r *rpc.Router, svc *api.WatchService) {
	rpc.RegisterStreamWithMeta(r, "watch", svc.Watch, rpc.MethodMeta{
		Description: "Stream changes matching a URI pattern.",
		Parameters: map[string]rpc.ParamMeta{
			"uri": {Description: "URI pattern to watch", Type: "string", Required: true},
		},
	})
}

func registerSystem(r *rpc.Router, svc *api.SystemService) {
	rpc.RegisterHandlerWithMeta(r, "system.health", svc.Health, rpc.MethodMeta{
		Description: "Report system health status.",
		Response: map[string]rpc.ParamMeta{
			"status": {Description: "Health status", Type: "string"},
		},
	})
	rpc.RegisterHandlerWithMeta(r, "system.version", svc.Version, rpc.MethodMeta{
		Description: "Report system version.",
		Response: map[string]rpc.ParamMeta{
			"version": {Description: "Daemon version string", Type: "string"},
		},
	})
}

// Start starts the daemon with the given [store.Store].
func (d *Daemon) Start(ctx context.Context, s store.Store) error {
	router := NewRouter(s, d.config.Version)

	d.server = &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	d.server.SetKeepAlivesEnabled(false)

	// Remove stale socket file if it exists.
	_ = os.Remove(d.config.SocketPath)

	lc := net.ListenConfig{}

	ln, err := lc.Listen(ctx, "unix", d.config.SocketPath)
	if err != nil {
		return err
	}

	d.listener = ln

	if chmodErr := os.Chmod(d.config.SocketPath, 0o600); chmodErr != nil {
		return chmodErr
	}

	pp := PIDPath(d.config.SocketPath)

	if pidErr := WritePID(pp); pidErr != nil {
		return pidErr
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer shutdownCancel()

		_ = d.server.Shutdown(shutdownCtx)
	}()

	err = d.server.Serve(d.listener)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	_ = RemovePID(pp)

	return err
}

// Stop stops the daemon.
func (d *Daemon) Stop() error {
	if d.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return d.server.Shutdown(ctx)
}

// Status returns the daemon's current status.
func (d *Daemon) Status() (string, error) {
	pp := PIDPath(d.config.SocketPath)

	pid, _ := ReadPID(pp)
	if !IsProcessAlive(pid) {
		return "stopped", nil
	}

	return "running", nil
}
