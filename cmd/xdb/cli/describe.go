package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
)

func (a *App) describeCmd() *cli.Command {
	return &cli.Command{
		Name:               "describe",
		Usage:              "Introspect actions, types, filters, errors, config, daemon, and data schemas",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[resource.action | TypeName | config | daemon]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "methods", Usage: "List all actions (dotted RPC form)"},
			&cli.BoolFlag{Name: "actions", Usage: "Show action \u00d7 resource matrix"},
			&cli.BoolFlag{Name: "types", Usage: "List all types"},
			&cli.BoolFlag{Name: "value-types", Usage: "List supported value types"},
			&cli.BoolFlag{Name: "filter", Usage: "Show CEL filter grammar"},
			&cli.BoolFlag{Name: "errors", Usage: "List error codes"},
			&cli.BoolFlag{Name: "config", Usage: "Explain the config file schema and defaults"},
			&cli.BoolFlag{Name: "daemon", Usage: "Explain daemon lifecycle and commands"},
			&cli.StringFlag{Name: "uri", Usage: "Data schema URI"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.schemaInspect,
	}
}

func (a *App) schemaInspect(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool("methods") {
		return a.listMethods(ctx, cmd)
	}

	if cmd.Bool("actions") {
		return a.listActions(ctx, cmd)
	}

	if cmd.Bool("types") {
		return a.listTypes(ctx, cmd)
	}

	if cmd.Bool("value-types") {
		return listValueTypes(cmd)
	}

	if cmd.Bool("filter") {
		return listFilterGrammar(cmd)
	}

	if cmd.Bool("errors") {
		return listErrorCodes(cmd)
	}

	if cmd.Bool("config") {
		return describeConfig(cmd)
	}

	if cmd.Bool("daemon") {
		return describeDaemon(cmd)
	}

	if uri := cmd.String("uri"); uri != "" {
		return a.describeDataSchema(ctx, cmd, uri)
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("specify a resource.action name, type name, config, daemon, or use --actions/--types/--value-types/--filter/--errors/--config/--daemon")
	}

	name := args.First()

	switch name {
	case "config":
		return describeConfig(cmd)
	case "daemon":
		return describeDaemon(cmd)
	}

	if strings.Contains(name, ".") {
		return a.describeMethod(ctx, cmd, name)
	}

	return a.describeType(ctx, cmd, name)
}

func (a *App) listMethods(ctx context.Context, cmd *cli.Command) error {
	var resp api.ListMethodsResponse
	if err := a.client.Call(ctx, "introspect.methods", &api.ListMethodsRequest{}, &resp); err != nil {
		return err
	}

	items := make([]any, len(resp.Methods))
	for i, m := range resp.Methods {
		items[i] = map[string]any{
			"method":      m.Method,
			"description": m.Description,
		}
	}

	return formatList(cmd, items)
}

func (a *App) listTypes(ctx context.Context, cmd *cli.Command) error {
	var resp api.ListTypesResponse
	if err := a.client.Call(ctx, "introspect.types", &api.ListTypesRequest{}, &resp); err != nil {
		return err
	}

	items := make([]any, len(resp.Types))
	for i, t := range resp.Types {
		items[i] = map[string]string{
			"type":        t.Type,
			"description": t.Description,
		}
	}

	return formatList(cmd, items)
}

// typeDescriptions maps each [core.TID] to a user-facing description.
// Derived from [core.ValueTypes] so the list never drifts.
var typeDescriptions = map[core.TID]string{
	core.TIDString:   "UTF-8 string",
	core.TIDInteger:  "64-bit signed integer",
	core.TIDUnsigned: "64-bit unsigned integer",
	core.TIDFloat:    "64-bit floating point",
	core.TIDBoolean:  "true or false",
	core.TIDTime:     "RFC 3339 timestamp",
	core.TIDBytes:    "Binary data",
	core.TIDJSON:     "Arbitrary JSON",
	core.TIDArray:    "Array of T",
}

func listValueTypes(cmd *cli.Command) error {
	items := make([]any, len(core.ValueTypes))
	for i, tid := range core.ValueTypes {
		items[i] = map[string]string{
			"type":        tid.Lower(),
			"description": typeDescriptions[tid],
		}
	}

	return formatList(cmd, items)
}

func (a *App) describeMethod(ctx context.Context, cmd *cli.Command, name string) error {
	var resp api.DescribeMethodResponse
	if err := a.client.Call(ctx, "introspect.method", &api.DescribeMethodRequest{
		Method: name,
	}, &resp); err != nil {
		return err
	}

	result := map[string]any{
		"method":      resp.Method,
		"description": resp.Description,
		"mutating":    resp.Mutating,
	}

	if len(resp.Parameters) > 0 {
		result["parameters"] = resp.Parameters
	}

	if len(resp.Response) > 0 {
		result["response"] = resp.Response
	}

	return formatOne(cmd, result)
}

func (a *App) describeType(ctx context.Context, cmd *cli.Command, name string) error {
	var resp api.DescribeTypeResponse
	if err := a.client.Call(ctx, "introspect.type", &api.DescribeTypeRequest{
		Type: name,
	}, &resp); err != nil {
		return err
	}

	return formatOne(cmd, map[string]any{
		"type":        resp.Type,
		"description": resp.Description,
	})
}

// listActions returns the action x resource matrix derived from the live
// introspect.methods response. Each row is one resource with a list of
// supported actions and whether each is mutating.
func (a *App) listActions(ctx context.Context, cmd *cli.Command) error {
	var resp api.ListMethodsResponse
	if err := a.client.Call(ctx, "introspect.methods", &api.ListMethodsRequest{}, &resp); err != nil {
		return wrapRPCError("introspect", "methods", "", err)
	}

	type row struct {
		Resource string   `json:"resource"`
		Actions  []string `json:"actions"`
		Mutating []string `json:"mutating"`
	}

	byResource := make(map[string]*row)

	for _, m := range resp.Methods {
		resource, action, ok := strings.Cut(m.Method, ".")
		if !ok {
			continue
		}

		r := byResource[resource]
		if r == nil {
			r = &row{Resource: resource}
			byResource[resource] = r
		}

		r.Actions = append(r.Actions, action)
	}

	names := make([]string, 0, len(byResource))
	for name := range byResource {
		names = append(names, name)
	}

	sort.Strings(names)

	items := make([]any, 0, len(names))
	for _, name := range names {
		r := byResource[name]
		sort.Strings(r.Actions)
		items = append(items, map[string]any{
			"resource": r.Resource,
			"actions":  r.Actions,
		})
	}

	return formatList(cmd, items)
}

// listFilterGrammar returns the CEL operators and functions supported by
// --filter. Static reference — no RPC call.
func listFilterGrammar(cmd *cli.Command) error {
	doc := map[string]any{
		"kind":      "FilterGrammar",
		"dialect":   "CEL (AIP-160)",
		"operators": []string{"==", "!=", "<", "<=", ">", ">=", "&&", "||", "!", "in"},
		"functions": []string{".contains(s)", ".startsWith(s)", ".endsWith(s)", "size(x)"},
		"examples": []string{
			`status == "published"`,
			`age >= 18 && status == "active"`,
			`title.contains("hello") || title.startsWith("Hi")`,
			`status in ["active", "pending"]`,
			`size(tags) > 0`,
			`!(archived == true)`,
		},
	}

	return formatOne(cmd, doc)
}

// listErrorCodes returns the error code catalog rendered by the CLI.
// Static reference — matches the codes in [output.ErrorEnvelope].
func listErrorCodes(cmd *cli.Command) error {
	type entry struct {
		Code        string `json:"code"`
		Description string `json:"description"`
		ExitCode    int    `json:"exit_code"`
	}

	items := []any{
		entry{Code: CodeNotFound, Description: "Resource does not exist", ExitCode: ExitAppError},
		entry{Code: CodeAlreadyExists, Description: "Resource already exists (use update or upsert)", ExitCode: ExitAppError},
		entry{Code: CodeSchemaViolation, Description: "Payload violates schema constraints", ExitCode: ExitAppError},
		entry{Code: CodeInvalidArgument, Description: "Invalid command-line arguments or RPC parameters", ExitCode: ExitInvalidArgs},
		entry{Code: CodeConnectionRefused, Description: "Daemon is not reachable — run xdb daemon start", ExitCode: ExitConnection},
		entry{Code: CodeInternal, Description: "Unexpected internal error", ExitCode: ExitInternal},
	}

	return formatList(cmd, items)
}

// describeConfig returns a static description of the XDB config file — its
// default path, JSON schema, field defaults, derived paths, and validation
// rules — so agents can discover how to configure the daemon without reading
// the concept docs.
func describeConfig(cmd *cli.Command) error {
	doc := map[string]any{
		"kind":         "ConfigDescription",
		"default_path": DefaultConfigPath(),
		"root_flag":    "--config / -c (overrides default path)",
		"created_by":   []string{"xdb init", "xdb daemon start (on first run)"},
		"fields": []map[string]any{
			{"name": "dir", "default": defaultConfigDir, "description": "Root directory for all XDB data (absolute or starts with ~)"},
			{"name": "daemon.socket", "default": defaultSocket, "description": "Unix socket filename (no path separators)"},
			{"name": "store.backend", "default": defaultBackend, "description": "Store backend: sqlite, memory, redis, or fs"},
			{"name": "store.sqlite.path", "default": "<datadir>/xdb.db", "description": "SQLite database file path"},
			{"name": "store.sqlite.journal", "default": defaultJournal, "description": "Journal mode: wal, delete, truncate, persist, memory, off"},
			{"name": "store.sqlite.sync", "default": defaultSync, "description": "Synchronous mode: off, normal, full, extra"},
			{"name": "store.sqlite.cache_size", "default": defaultCacheSize, "description": "Page cache size in KiB (negative) or pages (positive)"},
			{"name": "store.sqlite.busy_timeout", "default": defaultBusyTimeout, "description": "Busy timeout in milliseconds"},
			{"name": "store.redis.addr", "default": "(required for redis)", "description": "Redis server address (host:port)"},
			{"name": "store.redis.password", "default": "", "description": "Redis auth password"},
			{"name": "store.redis.db", "default": 0, "description": "Redis database number"},
			{"name": "store.fs.dir", "default": "<datadir>", "description": "Filesystem store root directory"},
			{"name": "log_level", "default": defaultLogLevel, "description": "Log level: debug, info, warn, error"},
		},
		"derived_paths": map[string]string{
			"socket": "<dir>/" + defaultSocket,
			"log":    "<dir>/xdb.log",
			"pid":    "<dir>/xdb.pid",
			"data":   "<dir>/data",
		},
		"validation": []string{
			"dir must be non-empty and absolute (or start with ~)",
			"daemon.socket must be a filename (no / or \\)",
			"log_level must be debug, info, warn, or error",
			"store.backend must be memory, sqlite, fs, or redis",
			"store.redis.addr is required when backend is redis",
		},
		"example": map[string]any{
			"dir":       "~/.xdb",
			"daemon":    map[string]any{"socket": "xdb.sock"},
			"store":     map[string]any{"backend": "sqlite"},
			"log_level": "info",
		},
	}

	return formatOne(cmd, doc)
}

// describeDaemon returns a static description of the daemon lifecycle — the
// subcommands, socket/log/pid file locations, and the parent-child spawn
// pattern — so agents can manage the daemon without reading the concept docs.
func describeDaemon(cmd *cli.Command) error {
	doc := map[string]any{
		"kind":      "DaemonDescription",
		"transport": "JSON-RPC 2.0 over Unix domain socket",
		"commands": []map[string]any{
			{"name": "xdb daemon start", "flags": []string{"--foreground"}, "description": "Spawn the daemon in the background (idempotent). --foreground blocks in the current process."},
			{"name": "xdb daemon stop", "description": "Send SIGTERM and wait up to 5s for exit (idempotent)."},
			{"name": "xdb daemon status", "description": "Report running|stopped, socket path, and PID."},
			{"name": "xdb daemon restart", "description": "Stop if running, then start."},
			{"name": "xdb init", "description": "Create config and data dir, then start the daemon."},
		},
		"spawn_pattern": []string{
			"Parent CLI loads config and checks PID file",
			"Parent re-execs the binary with XDB_DAEMON_CHILD=1 and setsid",
			"Child redirects stdout/stderr to the log file and writes the PID file",
			"Child serves JSON-RPC on the Unix socket until SIGTERM/SIGINT",
			"Parent waits up to 3s for the socket to accept connections, then exits",
		},
		"files": map[string]string{
			"socket": "<dir>/" + defaultSocket + " — Unix socket for JSON-RPC",
			"pid":    "<dir>/xdb.pid — PID of the running daemon",
			"log":    "<dir>/xdb.log — daemon stdout/stderr",
		},
		"idempotency": "start is a no-op when already running; stop is a no-op when already stopped",
		"related":     []string{"describe --config", "describe --methods"},
	}

	return formatOne(cmd, doc)
}

func (a *App) describeDataSchema(ctx context.Context, cmd *cli.Command, raw string) error {
	uri, err := core.ParseURI(raw)
	if err != nil {
		return err
	}

	var resp api.GetSchemaResponse
	if err := a.client.Call(ctx, "schemas.get", &api.GetSchemaRequest{
		URI: uri.String(),
	}, &resp); err != nil {
		return err
	}

	return formatOne(cmd, map[string]any{
		"kind": "DataSchemaDescription",
		"uri":  uri.String(),
		"data": resp.Data,
	})
}
