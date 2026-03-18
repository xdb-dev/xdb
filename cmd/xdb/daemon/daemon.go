// Package daemon manages the XDB daemon lifecycle.
package daemon

import (
	"context"
	"fmt"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/rpc"
	"github.com/xdb-dev/xdb/store"
)

// Config holds daemon configuration.
type Config struct {
	Addr       string
	SocketPath string
	LogFile    string
	Version    string
}

// Daemon manages the XDB daemon process.
type Daemon struct {
	config Config
}

// New creates a new [Daemon] with the given configuration.
func New(cfg Config) *Daemon {
	return &Daemon{config: cfg}
}

// NewRouter creates a [rpc.Router] with all services registered.
func NewRouter(s store.Store, version string) *rpc.Router {
	r := rpc.NewRouter()

	records := api.NewRecordService(s)
	schemas := api.NewSchemaService(s)
	namespaces := api.NewNamespaceService(s)
	batch := api.NewBatchService(s)
	watch := api.NewWatchService(s)
	system := api.NewSystemService(version)

	// Record operations.
	rpc.RegisterHandler(r, "records.create", records.Create)
	rpc.RegisterHandler(r, "records.get", records.Get)
	rpc.RegisterHandler(r, "records.list", records.List)
	rpc.RegisterHandler(r, "records.update", records.Update)
	rpc.RegisterHandler(r, "records.upsert", records.Upsert)
	rpc.RegisterHandler(r, "records.delete", records.Delete)

	// Schema operations.
	rpc.RegisterHandler(r, "schemas.create", schemas.Create)
	rpc.RegisterHandler(r, "schemas.get", schemas.Get)
	rpc.RegisterHandler(r, "schemas.list", schemas.List)
	rpc.RegisterHandler(r, "schemas.update", schemas.Update)
	rpc.RegisterHandler(r, "schemas.delete", schemas.Delete)

	// Namespace operations.
	rpc.RegisterHandler(r, "namespaces.get", namespaces.Get)
	rpc.RegisterHandler(r, "namespaces.list", namespaces.List)

	// Batch operations.
	rpc.RegisterHandler(r, "batch.execute", batch.Execute)

	// Streaming.
	rpc.RegisterStream(r, "watch", watch.Watch)

	// System.
	rpc.RegisterHandler(r, "system.health", system.Health)
	rpc.RegisterHandler(r, "system.version", system.Version)

	// Introspection (registered last so it can see all other methods).
	introspect := api.NewIntrospectService(r)
	rpc.RegisterHandler(r, "introspect.method", introspect.DescribeMethod)
	rpc.RegisterHandler(r, "introspect.type", introspect.DescribeType)
	rpc.RegisterHandler(r, "introspect.methods", introspect.ListMethods)
	rpc.RegisterHandler(r, "introspect.types", introspect.ListTypes)

	return r
}

// Start starts the daemon.
func (d *Daemon) Start(_ context.Context) error {
	return fmt.Errorf("daemon: Start not implemented")
}

// Stop stops the daemon.
func (d *Daemon) Stop() error {
	return fmt.Errorf("daemon: Stop not implemented")
}

// Status returns the daemon's current status.
func (d *Daemon) Status() (string, error) {
	return "", fmt.Errorf("daemon: Status not implemented")
}
