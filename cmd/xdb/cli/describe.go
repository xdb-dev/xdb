package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/api/catalog"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func (a *App) describeCmd() *cli.Command {
	return &cli.Command{
		Name:               "describe",
		Usage:              "Introspect actions, types, filters, errors, config, daemon, and data schemas",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[resource.action | TypeName | config | daemon | schema-format]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "methods", Usage: "List all actions (dotted RPC form)"},
			&cli.BoolFlag{Name: "actions", Usage: "Show action \u00d7 resource matrix"},
			&cli.BoolFlag{Name: "types", Usage: "List all types"},
			&cli.BoolFlag{Name: "value-types", Usage: "List supported value types"},
			&cli.BoolFlag{Name: "filter", Usage: "Show CEL filter grammar"},
			&cli.BoolFlag{Name: "errors", Usage: "List error codes"},
			&cli.BoolFlag{Name: "config", Usage: "Explain the config file schema and defaults"},
			&cli.BoolFlag{Name: "daemon", Usage: "Explain daemon lifecycle and commands"},
			&cli.BoolFlag{Name: "schema-format", Usage: "Explain the schema-definition JSON format"},
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

	if cmd.Bool("schema-format") {
		return describeSchemaFormat(cmd)
	}

	if uri := cmd.String("uri"); uri != "" {
		return a.describeDataSchema(ctx, cmd, uri)
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return describeOverview(cmd)
	}

	name := args.First()

	switch name {
	case "config":
		return describeConfig(cmd)
	case "daemon":
		return describeDaemon(cmd)
	case "schema-format":
		return describeSchemaFormat(cmd)
	}

	if strings.Contains(name, ".") {
		return a.describeMethod(ctx, cmd, name)
	}

	return a.describeType(ctx, cmd, name)
}

func (a *App) listMethods(ctx context.Context, cmd *cli.Command) error {
	var resp api.ListMethodsResponse
	if err := a.client.Call(ctx, "introspect.methods", &api.ListMethodsRequest{}, &resp); err != nil {
		if isConnectionError(err) {
			return listMethodsOffline(cmd)
		}
		return wrapRPCError("introspect", "methods", "", err)
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

// listMethodsOffline serves the method list from the embedded catalog
// when no daemon is reachable.
func listMethodsOffline(cmd *cli.Command) error {
	methods := catalog.Methods()

	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]any, len(names))
	for i, name := range names {
		items[i] = map[string]any{
			"method":      name,
			"description": methods[name].Description,
			"source":      "embedded",
		}
	}

	return formatList(cmd, items)
}

func (a *App) listTypes(ctx context.Context, cmd *cli.Command) error {
	var resp api.ListTypesResponse
	if err := a.client.Call(ctx, "introspect.types", &api.ListTypesRequest{}, &resp); err != nil {
		if isConnectionError(err) {
			return listTypesOffline(cmd)
		}
		return wrapRPCError("introspect", "types", "", err)
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
// The map is written by hand. Keep it in step with [core.ValueTypes].
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
		if isConnectionError(err) {
			return describeMethodOffline(cmd, name)
		}
		return wrapRPCError("introspect", "method", "", err)
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

	addCLISection(cmd, name, result)

	return formatOne(cmd, result)
}

// describeMethodOffline serves method metadata from the embedded
// catalog when no daemon is reachable.
func describeMethodOffline(cmd *cli.Command, name string) error {
	meta, ok := catalog.Method(name)
	if !ok {
		return &output.ErrorEnvelope{
			Code:     CodeNotFound,
			Message:  fmt.Sprintf("unknown method %q", name),
			Resource: "introspect",
			Action:   "method",
			Hint:     "run 'xdb describe --methods' to list available methods",
		}
	}

	result := map[string]any{
		"method":      name,
		"description": meta.Description,
		"mutating":    meta.Mutating,
		"source":      "embedded",
	}

	if len(meta.Parameters) > 0 {
		result["parameters"] = meta.Parameters
	}

	if len(meta.Response) > 0 {
		result["response"] = meta.Response
	}

	addCLISection(cmd, name, result)

	return formatOne(cmd, result)
}

// listTypesOffline serves the type catalog from the embedded catalog.
func listTypesOffline(cmd *cli.Command) error {
	types := catalog.Types()

	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]any, len(names))
	for i, name := range names {
		items[i] = map[string]string{
			"type":        name,
			"description": types[name],
			"source":      "embedded",
		}
	}

	return formatList(cmd, items)
}

// addCLISection attaches the command and flags for a method by reading
// the live CLI command tree.
func addCLISection(cmd *cli.Command, method string, result map[string]any) {
	target := cliCommandFor(cmd.Root(), method)
	if target == nil {
		return
	}

	flags := make([]map[string]any, 0, len(target.Flags))
	for _, f := range target.Flags {
		names := f.Names()
		if len(names) == 0 {
			continue
		}

		entry := map[string]any{"name": names[0]}
		if len(names) > 1 {
			entry["aliases"] = names[1:]
		}
		if df, ok := f.(cli.DocGenerationFlag); ok {
			entry["usage"] = df.GetUsage()
		}
		flags = append(flags, entry)
	}

	section := map[string]any{"command": "xdb " + cliCommandPath(method)}
	if len(flags) > 0 {
		section["flags"] = flags
	}

	result["cli"] = section
}

// cliCommandPath maps a dotted RPC method to its CLI invocation.
func cliCommandPath(method string) string {
	switch method {
	case "batch.execute":
		return "batch"
	case "watch":
		return "watch"
	default:
		return strings.ReplaceAll(method, ".", " ")
	}
}

// cliCommandFor resolves the CLI command implementing an RPC method.
func cliCommandFor(root *cli.Command, method string) *cli.Command {
	parts := strings.Split(cliCommandPath(method), " ")

	current := root
	for _, part := range parts {
		var next *cli.Command
		for _, c := range current.Commands {
			if c.Name == part {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		current = next
	}

	return current
}

func (a *App) describeType(ctx context.Context, cmd *cli.Command, name string) error {
	var resp api.DescribeTypeResponse
	if err := a.client.Call(ctx, "introspect.type", &api.DescribeTypeRequest{
		Type: name,
	}, &resp); err != nil {
		if isConnectionError(err) {
			return describeTypeOffline(cmd, name)
		}
		return wrapRPCError("introspect", "type", "", err)
	}

	return formatOne(cmd, map[string]any{
		"type":        resp.Type,
		"description": resp.Description,
	})
}

// describeTypeOffline serves type metadata from the embedded catalog.
func describeTypeOffline(cmd *cli.Command, name string) error {
	desc, ok := catalog.Types()[name]
	if !ok {
		return &output.ErrorEnvelope{
			Code:     CodeNotFound,
			Message:  fmt.Sprintf("unknown type %q", name),
			Resource: "introspect",
			Action:   "type",
			Hint:     "run 'xdb describe --types' to list available types",
		}
	}

	return formatOne(cmd, map[string]string{
		"type":        name,
		"description": desc,
		"source":      "embedded",
	})
}

// listActions returns the action x resource matrix derived from the live
// introspect.methods response. Each row is one resource with a list of
// supported actions. The Mutating column lists the subset of actions that
// write, so an agent can tell a safe retry from a repeated write.
func (a *App) listActions(ctx context.Context, cmd *cli.Command) error {
	var resp api.ListMethodsResponse
	if err := a.client.Call(ctx, "introspect.methods", &api.ListMethodsRequest{}, &resp); err != nil {
		if isConnectionError(err) {
			return listActionsOffline(cmd)
		}
		return wrapRPCError("introspect", "actions", "", err)
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
		if m.Mutating {
			r.Mutating = append(r.Mutating, action)
		}
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
		entry{Code: CodeConflict, Description: "Resource exists with different data (use update or upsert)", ExitCode: ExitAppError},
		entry{Code: CodeNotImplemented, Description: "Operation not implemented in this daemon build", ExitCode: ExitAppError},
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
			"pid":    "<dir>/<socket-name>.pid",
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
			"pid":    "<dir>/<socket-name>.pid — PID of the running daemon",
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
		return invalidArgError("schemas", "describe", err)
	}

	var resp api.GetSchemaResponse
	if err := a.client.Call(ctx, "schemas.get", &api.GetSchemaRequest{
		URI: uri.String(),
	}, &resp); err != nil {
		return wrapRPCError("schemas", "describe", uri.String(), err)
	}

	return formatOne(cmd, dataSchemaDescription(uri.String(), resp.Data))
}

// dataSchemaDescription surfaces the schema-level fields agents read — the
// description, mode, revision, and source annotations — alongside a per-field
// breakdown (type, required, description, annotations, and element schema).
func dataSchemaDescription(uri string, def *schema.Def) map[string]any {
	doc := map[string]any{
		"kind":     "DataSchemaDescription",
		"uri":      uri,
		"mode":     string(def.Mode),
		"revision": def.Revision,
	}

	if def.Description != "" {
		doc["description"] = def.Description
	}
	if len(def.Annotations) > 0 {
		doc["annotations"] = def.Annotations
	}

	doc["fields"] = describeFields(def.Fields)

	return doc
}

// describeFields renders a schema's fields into a stable, sorted list.
func describeFields(fields map[string]schema.Field) []map[string]any {
	names := sortedFieldNames(fields)

	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, describeField(name, fields[name]))
	}

	return out
}

// describeField renders one field, including the object-array element schema
// when present.
func describeField(name string, f schema.Field) map[string]any {
	entry := map[string]any{
		"name":     name,
		"type":     typeDisplay(f.Type),
		"required": f.Required,
	}

	if f.Indexed {
		entry["indexed"] = true
	}
	if f.Unique {
		entry["unique"] = true
	}
	if f.Description != "" {
		entry["description"] = f.Description
	}
	if len(f.Annotations) > 0 {
		entry["annotations"] = f.Annotations
	}
	if len(f.Items) > 0 {
		entry["items"] = describeFields(f.Items)
	}

	return entry
}

// listActionsOffline derives the action matrix from the embedded catalog.
func listActionsOffline(cmd *cli.Command) error {
	byResource := make(map[string][]string)

	for method := range catalog.Methods() {
		resource, action, ok := strings.Cut(method, ".")
		if !ok {
			resource, action = method, method
		}
		byResource[resource] = append(byResource[resource], action)
	}

	names := make([]string, 0, len(byResource))
	for name := range byResource {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]any, 0, len(names))
	for _, name := range names {
		actions := byResource[name]
		sort.Strings(actions)
		items = append(items, map[string]any{
			"resource": name,
			"actions":  actions,
			"source":   "embedded",
		})
	}

	return formatList(cmd, items)
}

// describeSchemaFormat describes the schema-definition JSON format.
// Type names and modes are read from their definitions.
func describeSchemaFormat(cmd *cli.Command) error {
	modes := make([]string, 0, 3)
	for _, m := range schema.ValidModes() {
		modes = append(modes, string(m))
	}

	doc := map[string]any{
		"kind": "SchemaFormat",
		"top_level_keys": []map[string]string{
			{"key": "fields", "description": "Map of field name to field definition (required)"},
			{"key": "mode", "description": "Validation mode; defaults to strict"},
			{"key": "description", "description": "Human-readable schema description"},
			{"key": "annotations", "description": "Free-form string key/value metadata"},
			{"key": "revision", "description": "Optimistic-concurrency base revision for updates (omit or 0 for unconditional)"},
		},
		"field_keys": []map[string]string{
			{"key": "type", "description": "Value type name (required)"},
			{"key": "required", "description": "Reject writes missing this field"},
			{"key": "indexed", "description": "Build a lookup index on this scalar field (SQLite strict and dynamic schemas; a hint elsewhere)"},
			{"key": "unique", "description": "Reject duplicate values on a backend that materializes a unique index (SQLite strict and dynamic schemas); a hint elsewhere. Fixed at create"},
			{"key": "elem_type", "description": "Element type; required when type is array"},
			{"key": "items", "description": "Member field definitions for array fields with json elements"},
			{"key": "description", "description": "Human-readable field description"},
			{"key": "annotations", "description": "Free-form string key/value metadata"},
		},
		"types": core.ValueTypeNames(),
		"modes": modes,
		"mode_semantics": map[string]string{
			"strict":   "only declared fields are accepted",
			"flexible": "undeclared fields pass through unvalidated",
			"dynamic":  "undeclared fields evolve the schema automatically",
		},
		"example": map[string]any{
			"mode": "strict",
			"fields": map[string]any{
				"title": map[string]any{"type": "string", "required": true},
				"tags":  map[string]any{"type": "array", "elem_type": "string"},
			},
		},
		"notes": []string{
			"all keys are lowercase",
			"schemas update adds or replaces fields; removal is not supported",
			"indexed and unique are scalar-only and fixed at creation. Both are backend capabilities: only a backend that materializes an index applies them (SQLite strict and dynamic schemas), where indexed speeds up lookups and a duplicate write on a unique field fails with UNIQUE_VIOLATION. Elsewhere both are stored declarations that change no behavior",
		},
	}

	return formatOne(cmd, doc)
}

// describeOverview lists every describe topic, so a bare `xdb describe`
// is a map instead of an error.
func describeOverview(cmd *cli.Command) error {
	doc := map[string]any{
		"kind": "DescribeOverview",
		"topics": []map[string]string{
			{"invoke": "describe --actions", "description": "Action x resource matrix"},
			{"invoke": "describe --methods", "description": "Every RPC method with descriptions"},
			{"invoke": "describe <resource>.<action>", "description": "One method: parameters, response, CLI flags"},
			{"invoke": "describe --types", "description": "Core type catalog"},
			{"invoke": "describe <TypeName>", "description": "One core type"},
			{"invoke": "describe --value-types", "description": "Value types usable in schema fields"},
			{"invoke": "describe --schema-format", "description": "Schema-definition JSON format"},
			{"invoke": "describe --filter", "description": "CEL filter grammar with examples"},
			{"invoke": "describe --errors", "description": "Error codes and exit codes"},
			{"invoke": "describe --config", "description": "Config file schema and defaults"},
			{"invoke": "describe --daemon", "description": "Daemon lifecycle and commands"},
			{"invoke": "describe --uri <schema-uri>", "description": "A data schema's fields and modes"},
		},
	}

	return formatOne(cmd, doc)
}
