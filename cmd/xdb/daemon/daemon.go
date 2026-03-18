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
	"github.com/xdb-dev/xdb/store/xdbmemory"
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
	listener net.Listener
	server   *http.Server
	config   Config
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
func (d *Daemon) Start(ctx context.Context) error {
	s := xdbmemory.New()
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

	pid := pidPath(d.config.SocketPath)

	if pidErr := WritePID(pid); pidErr != nil {
		return pidErr
	}

	err = d.server.Serve(d.listener)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	_ = RemovePID(pid)

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
	pp := pidPath(d.config.SocketPath)

	if !isProcessRunning(pp) {
		return "stopped", nil
	}

	return "running", nil
}

// isProcessRunning checks if the process in the PID file is alive.
func isProcessRunning(pp string) bool {
	pid, err := ReadPID(pp)
	if err != nil {
		return false
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		_ = RemovePID(pp)
		return false
	}

	err = p.Signal(syscall.Signal(0))
	if err != nil {
		_ = RemovePID(pp)
		return false
	}

	return true
}

func pidPath(socketPath string) string {
	return strings.TrimSuffix(socketPath, filepath.Ext(socketPath)) + ".pid"
}
