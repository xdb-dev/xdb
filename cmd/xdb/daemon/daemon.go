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
	"github.com/xdb-dev/xdb/api/catalog"
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
// The returned bus carries change notifications from mutating services
// to watch streams; the caller owns its lifecycle and must Close it on
// shutdown so watch streams end cleanly.
func NewRouter(s store.Store, version string) (*rpc.Router, *api.Bus) {
	r := rpc.NewRouter()
	bus := api.NewBus()

	registerRecords(r, api.NewRecordService(s, api.WithEvents(bus)))
	registerSchemas(r, api.NewSchemaService(s, api.WithEvents(bus)))
	registerNamespaces(r, api.NewNamespaceService(s))
	registerBatch(r, api.NewBatchService(s, api.WithEvents(bus)))
	registerWatch(r, api.NewWatchService(bus))
	registerSystem(r, api.NewSystemService(version))

	// Introspection (registered last so it can see all other methods).
	introspect := api.NewIntrospectService(r)
	rpc.RegisterHandlerWithMeta(r, "introspect.method", introspect.DescribeMethod, mustMeta("introspect.method"))
	rpc.RegisterHandlerWithMeta(r, "introspect.type", introspect.DescribeType, mustMeta("introspect.type"))
	rpc.RegisterHandlerWithMeta(r, "introspect.methods", introspect.ListMethods, mustMeta("introspect.methods"))
	rpc.RegisterHandlerWithMeta(r, "introspect.types", introspect.ListTypes, mustMeta("introspect.types"))

	return r, bus
}

// mustMeta returns the catalog metadata for name, panicking if none is
// registered. A panic here means a method is being wired up without a
// corresponding [catalog.Methods] entry — registration and the catalog
// have drifted apart.
func mustMeta(name string) rpc.MethodMeta {
	meta, ok := catalog.Method(name)
	if !ok {
		panic("daemon: no catalog entry for method " + name)
	}
	return meta
}

func registerRecords(r *rpc.Router, svc *api.RecordService) {
	rpc.RegisterHandlerWithMeta(r, "records.create", svc.Create, mustMeta("records.create"))
	rpc.RegisterHandlerWithMeta(r, "records.get", svc.Get, mustMeta("records.get"))
	rpc.RegisterHandlerWithMeta(r, "records.list", svc.List, mustMeta("records.list"))
	rpc.RegisterHandlerWithMeta(r, "records.update", svc.Update, mustMeta("records.update"))
	rpc.RegisterHandlerWithMeta(r, "records.upsert", svc.Upsert, mustMeta("records.upsert"))
	rpc.RegisterHandlerWithMeta(r, "records.delete", svc.Delete, mustMeta("records.delete"))
}

func registerSchemas(r *rpc.Router, svc *api.SchemaService) {
	rpc.RegisterHandlerWithMeta(r, "schemas.create", svc.Create, mustMeta("schemas.create"))
	rpc.RegisterHandlerWithMeta(r, "schemas.get", svc.Get, mustMeta("schemas.get"))
	rpc.RegisterHandlerWithMeta(r, "schemas.list", svc.List, mustMeta("schemas.list"))
	rpc.RegisterHandlerWithMeta(r, "schemas.update", svc.Update, mustMeta("schemas.update"))
	rpc.RegisterHandlerWithMeta(r, "schemas.delete", svc.Delete, mustMeta("schemas.delete"))
}

func registerNamespaces(r *rpc.Router, svc *api.NamespaceService) {
	rpc.RegisterHandlerWithMeta(r, "namespaces.get", svc.Get, mustMeta("namespaces.get"))
	rpc.RegisterHandlerWithMeta(r, "namespaces.list", svc.List, mustMeta("namespaces.list"))
}

func registerBatch(r *rpc.Router, svc *api.BatchService) {
	rpc.RegisterHandlerWithMeta(r, "batch.execute", svc.Execute, mustMeta("batch.execute"))
}

func registerWatch(r *rpc.Router, svc *api.WatchService) {
	rpc.RegisterStreamWithMeta(r, "watch", svc.Watch, mustMeta("watch"))
}

func registerSystem(r *rpc.Router, svc *api.SystemService) {
	rpc.RegisterHandlerWithMeta(r, "system.health", svc.Health, mustMeta("system.health"))
	rpc.RegisterHandlerWithMeta(r, "system.version", svc.Version, mustMeta("system.version"))
}

// Start starts the daemon with the given [store.Store].
func (d *Daemon) Start(ctx context.Context, s store.Store) error {
	router, bus := NewRouter(s, d.config.Version)
	defer bus.Close()

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
