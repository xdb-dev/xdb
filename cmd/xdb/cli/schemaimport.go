package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/rpc"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/schema/jsonschemaimport"
	"github.com/xdb-dev/xdb/schema/protoimport"
)

// srcFormat identifies an importable schema source format.
type srcFormat string

const (
	formatProto      srcFormat = "proto"
	formatJSONSchema srcFormat = "jsonschema"
)

// schemaImportSubCmd is the `xdb schemas import <file>` command: it imports a
// proto or JSON Schema file, shows the delta against the stored schema, and
// applies it via schemas.create/update.
func (a *App) schemaImportSubCmd() *cli.Command {
	return &cli.Command{
		Name:               "import",
		Usage:              "Import a schema from a .proto or JSON Schema file",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[FILE]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "ns", Usage: "Target namespace"},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to schema file"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Show the delta without writing"},
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation; accept suspected renames as drops"},
			&cli.StringSliceFlag{Name: "rename", Usage: "Acknowledge a rename as old:new (repeatable)"},
			&cli.StringSliceFlag{Name: "allow-json", Usage: "Import a message/pointer as opaque JSON (repeatable)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.schemaImport,
	}
}

// schemaDiffSubCmd is the `xdb schemas diff <file>` command: the same walk as
// import, but it never writes. --check exits non-zero when there is drift.
func (a *App) schemaDiffSubCmd() *cli.Command {
	return &cli.Command{
		Name:               "diff",
		Usage:              "Show drift between a schema file and the stored schema",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[FILE]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "ns", Usage: "Target namespace"},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Path to schema file"},
			&cli.BoolFlag{Name: "check", Usage: "Exit non-zero when there is drift (CI hook)"},
			&cli.StringSliceFlag{Name: "allow-json", Usage: "Import a message/pointer as opaque JSON (repeatable)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Action: a.schemaDiff,
	}
}

// importParams carries the resolved inputs for an import run.
type importParams struct {
	path      string
	ns        string
	renames   map[string]string
	allowJSON []string
	dryRun    bool
	yes       bool
}

func (a *App) schemaImport(ctx context.Context, cmd *cli.Command) error {
	path := schemaFilePath(cmd)
	if path == "" {
		return invalidArgError("schemas", "import", fmt.Errorf("a schema file is required (positional argument or --file)"))
	}

	renames, err := parseRenameFlags(cmd.StringSlice("rename"))
	if err != nil {
		return invalidArgError("schemas", "import", err)
	}

	return a.doImport(ctx, importParams{
		path:      path,
		ns:        cmd.String("ns"),
		renames:   renames,
		allowJSON: cmd.StringSlice("allow-json"),
		dryRun:    cmd.Bool("dry-run"),
		yes:       cmd.Bool("yes"),
	}, os.Stdout)
}

func (a *App) schemaDiff(ctx context.Context, cmd *cli.Command) error {
	path := schemaFilePath(cmd)
	if path == "" {
		return invalidArgError("schemas", "diff", fmt.Errorf("a schema file is required (positional argument or --file)"))
	}

	drift, err := a.doDiff(ctx, importParams{
		path:      path,
		ns:        cmd.String("ns"),
		allowJSON: cmd.StringSlice("allow-json"),
	}, os.Stdout)
	if err != nil {
		return err
	}

	if cmd.Bool("check") && drift {
		return fmt.Errorf("schema drift detected in %s", path)
	}

	return nil
}

// doImport runs the import walk: load, delta, gate on renames, apply.
func (a *App) doImport(ctx context.Context, p importParams, out io.Writer) error {
	defs, source, err := loadSchemas(ctx, p.path, p.ns, p.allowJSON)
	if err != nil {
		return invalidArgError("schemas", "import", err)
	}

	for _, def := range defs {
		stored, err := a.fetchSchema(ctx, def.URI)
		if err != nil {
			return wrapRPCError("schemas", "import", def.URI.String(), err)
		}

		delta := computeDelta(stored, def, source)
		writeDelta(out, delta)

		if p.dryRun {
			continue
		}

		blocking := unresolvedRenames(delta.Renames, p.renames, p.yes)
		if len(blocking) > 0 {
			return invalidArgError("schemas", "import", renameRefusal(blocking))
		}

		if err := a.applySchema(ctx, stored == nil, def); err != nil {
			return wrapRPCError("schemas", "import", def.URI.String(), err)
		}

		_, _ = fmt.Fprintf(out, "applied %s\n", def.URI.String())
	}

	return nil
}

// doDiff runs the import walk without writing and reports whether any schema
// drifted from its stored form.
func (a *App) doDiff(ctx context.Context, p importParams, out io.Writer) (bool, error) {
	defs, source, err := loadSchemas(ctx, p.path, p.ns, p.allowJSON)
	if err != nil {
		return false, invalidArgError("schemas", "diff", err)
	}

	drift := false

	for _, def := range defs {
		stored, err := a.fetchSchema(ctx, def.URI)
		if err != nil {
			return false, wrapRPCError("schemas", "diff", def.URI.String(), err)
		}

		delta := computeDelta(stored, def, source)
		writeDelta(out, delta)

		if delta.hasDrift() {
			drift = true
		}
	}

	return drift, nil
}

// fetchSchema returns the stored schema for uri, or nil when it does not exist.
func (a *App) fetchSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	var resp api.GetSchemaResponse

	err := a.client.Call(ctx, "schemas.get", &api.GetSchemaRequest{
		URI: uri.String(),
	}, &resp)
	if err == nil {
		return resp.Data, nil
	}

	var rpcErr *rpc.Error
	if errors.As(err, &rpcErr) && rpcErr.Code == rpc.CodeNotFound {
		return nil, nil
	}

	return nil, err
}

// applySchema creates a new schema or updates an existing one.
func (a *App) applySchema(ctx context.Context, create bool, def *schema.Def) error {
	data, err := def.MarshalJSON()
	if err != nil {
		return err
	}

	if create {
		return a.client.Call(ctx, "schemas.create", &api.CreateSchemaRequest{
			URI:  def.URI.String(),
			Data: data,
		}, &api.CreateSchemaResponse{})
	}

	return a.client.Call(ctx, "schemas.update", &api.UpdateSchemaRequest{
		URI:  def.URI.String(),
		Data: data,
	}, &api.UpdateSchemaResponse{})
}

// --- Loading ---

// schemaFilePath resolves the input file path from --file or the first
// positional argument.
func schemaFilePath(cmd *cli.Command) string {
	if f := cmd.String("file"); f != "" {
		return f
	}

	return cmd.Args().First()
}

// detectFormat classifies a schema source file by extension, falling back to a
// content sniff when the extension is unknown.
func detectFormat(path string, data []byte) (srcFormat, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".proto":
		return formatProto, nil
	case ".json":
		return formatJSONSchema, nil
	}

	trimmed := strings.TrimSpace(string(data))
	switch {
	case strings.HasPrefix(trimmed, "{"):
		return formatJSONSchema, nil
	case strings.Contains(trimmed, "syntax") && strings.Contains(trimmed, "proto"):
		return formatProto, nil
	}

	return "", fmt.Errorf("cannot detect schema format from %q: expected a .proto or .json file", path)
}

// loadSchemas reads path, detects the format, and imports it into one or more
// schema definitions.
func loadSchemas(ctx context.Context, path, ns string, allowJSON []string) ([]*schema.Def, srcFormat, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read schema file: %w", err)
	}

	format, err := detectFormat(path, data)
	if err != nil {
		return nil, "", err
	}

	switch format {
	case formatJSONSchema:
		opts := []jsonschemaimport.Option{}
		if ns != "" {
			opts = append(opts, jsonschemaimport.WithNamespace(ns))
		}
		if len(allowJSON) > 0 {
			opts = append(opts, jsonschemaimport.WithJSON(allowJSON...))
		}

		def, err := jsonschemaimport.Import(data, opts...)
		if err != nil {
			return nil, format, err
		}

		return []*schema.Def{def}, format, nil

	case formatProto:
		defs, err := loadProto(ctx, path, ns, allowJSON)
		if err != nil {
			return nil, format, err
		}

		return defs, format, nil
	}

	return nil, "", fmt.Errorf("unsupported format %q", format)
}

// loadProto compiles a .proto file (resolving well-known imports) and imports
// every top-level message into a schema definition.
func loadProto(ctx context.Context, path, ns string, allowJSON []string) ([]*schema.Def, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{dir},
		}),
	}

	compiled, err := compiler.Compile(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("compile proto: %w", err)
	}

	files := make([]protoreflect.FileDescriptor, 0, len(compiled))
	for _, f := range compiled {
		files = append(files, f)
	}

	opts := []protoimport.Option{}
	if ns != "" {
		opts = append(opts, protoimport.WithNamespace(ns))
	}
	if len(allowJSON) > 0 {
		opts = append(opts, protoimport.WithAllowJSON(allowJSON...))
	}

	return protoimport.ImportFiles(files, opts...)
}

// --- Delta ---

// fieldSummary is the display form of a single field in a delta.
type fieldSummary struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// fieldChange records a field whose type or required-ness changed.
type fieldChange struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

// renamePair is a suspected rename: a removed field and an added field the
// importer believes are the same attribute. Proto renames are detected by field
// number (reliable); others by a type+position heuristic (a guess).
type renamePair struct {
	From  string `json:"from"`
	To    string `json:"to"`
	By    string `json:"by"`
	Proto bool   `json:"proto"`
}

// schemaDelta is the difference between a stored schema and an imported one.
type schemaDelta struct {
	URI       string
	Source    srcFormat
	Added     []fieldSummary
	Removed   []fieldSummary
	Changed   []fieldChange
	Renames   []renamePair
	NewSchema bool
}

// hasDrift reports whether the stored schema differs from the imported one.
func (d schemaDelta) hasDrift() bool {
	if d.NewSchema {
		return true
	}

	return len(d.Added) > 0 ||
		len(d.Removed) > 0 ||
		len(d.Changed) > 0 ||
		len(d.Renames) > 0
}

// computeDelta compares a stored schema (nil when absent) against an imported
// one. Suspected renames are pulled out of the plain added/removed lists.
func computeDelta(stored, imported *schema.Def, source srcFormat) schemaDelta {
	d := schemaDelta{
		URI:    imported.URI.String(),
		Source: source,
	}

	if stored == nil {
		d.NewSchema = true
		for _, name := range sortedFieldNames(imported.Fields) {
			d.Added = append(d.Added, summarize(name, imported.Fields[name]))
		}

		return d
	}

	var addedNames, removedNames []string

	for _, name := range sortedFieldNames(imported.Fields) {
		if _, ok := stored.Fields[name]; !ok {
			addedNames = append(addedNames, name)
		}
	}

	for _, name := range sortedFieldNames(stored.Fields) {
		if _, ok := imported.Fields[name]; !ok {
			removedNames = append(removedNames, name)
		}
	}

	for _, name := range sortedFieldNames(imported.Fields) {
		old, ok := stored.Fields[name]
		if !ok {
			continue
		}
		if fieldChanged(old, imported.Fields[name]) {
			d.Changed = append(d.Changed, fieldChange{
				Name: name,
				From: fieldDisplay(old),
				To:   fieldDisplay(imported.Fields[name]),
			})
		}
	}

	renames := detectRenames(stored, imported, addedNames, removedNames, source)
	renamedFrom, renamedTo := renameSets(renames)
	d.Renames = renames

	for _, name := range addedNames {
		if _, ok := renamedTo[name]; ok {
			continue
		}
		d.Added = append(d.Added, summarize(name, imported.Fields[name]))
	}

	for _, name := range removedNames {
		if _, ok := renamedFrom[name]; ok {
			continue
		}
		d.Removed = append(d.Removed, summarize(name, stored.Fields[name]))
	}

	return d
}

// detectRenames pairs removed and added fields into suspected renames. Proto
// sources pair by matching proto.number; other sources pair by matching
// core.Type in field order (a heuristic).
func detectRenames(stored, imported *schema.Def, addedNames, removedNames []string, source srcFormat) []renamePair {
	if len(addedNames) == 0 || len(removedNames) == 0 {
		return nil
	}

	if source == formatProto {
		return protoRenames(stored, imported, addedNames, removedNames)
	}

	return heuristicRenames(stored, imported, addedNames, removedNames)
}

// protoRenames pairs fields whose proto.number matches under a different name.
// protoimport.CheckRename is the authoritative signal; it is consulted first so
// that a source without a proto rename never reports one.
func protoRenames(stored, imported *schema.Def, addedNames, removedNames []string) []renamePair {
	if !errors.Is(protoimport.CheckRename(stored, imported), protoimport.ErrRename) {
		return nil
	}

	byNumber := make(map[string]string, len(removedNames))
	for _, name := range removedNames {
		if num, ok := stored.Fields[name].Annotations["proto.number"]; ok {
			byNumber[num] = name
		}
	}

	pairs := make([]renamePair, 0, len(addedNames))
	for _, name := range addedNames {
		num, ok := imported.Fields[name].Annotations["proto.number"]
		if !ok {
			continue
		}
		from, ok := byNumber[num]
		if !ok {
			continue
		}
		pairs = append(pairs, renamePair{
			From:  from,
			To:    name,
			By:    "proto field #" + num,
			Proto: true,
		})
	}

	return pairs
}

// heuristicRenames pairs a removed and an added field that share a core.Type,
// matched greedily in sorted order. This is a guess, not a reliable signal.
func heuristicRenames(stored, imported *schema.Def, addedNames, removedNames []string) []renamePair {
	used := make(map[string]bool, len(addedNames))

	var pairs []renamePair
	for _, from := range removedNames {
		oldType := stored.Fields[from].Type
		for _, to := range addedNames {
			if used[to] {
				continue
			}
			if imported.Fields[to].Type != oldType {
				continue
			}
			used[to] = true
			pairs = append(pairs, renamePair{
				From: from,
				To:   to,
				By:   "type+position heuristic",
			})

			break
		}
	}

	return pairs
}

// unresolvedRenames returns the suspected renames that would apply without an
// explicit acknowledgment. Proto renames are reliably detected and applied
// automatically; non-proto renames must be acknowledged with --rename or --yes
// or they orphan the old attribute's stored data.
func unresolvedRenames(renames []renamePair, acks map[string]string, yes bool) []renamePair {
	if yes {
		return nil
	}

	blocking := make([]renamePair, 0, len(renames))
	for _, r := range renames {
		if r.Proto {
			continue
		}
		if acks[r.From] == r.To {
			continue
		}
		blocking = append(blocking, r)
	}

	return blocking
}

// renameRefusal builds the error returned when suspected renames are not
// acknowledged.
func renameRefusal(blocking []renamePair) error {
	var b strings.Builder
	b.WriteString("refusing to import: suspected rename(s) would orphan the old attribute's stored data:\n")

	for _, r := range blocking {
		fmt.Fprintf(&b, "  %s -> %s (%s)\n", r.From, r.To, r.By)
	}

	b.WriteString("pass --rename old:new to acknowledge each rename, or --yes to accept the drops")

	return errors.New(b.String())
}

// parseRenameFlags parses repeated old:new tokens into an acknowledgment map.
func parseRenameFlags(vals []string) (map[string]string, error) {
	if len(vals) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(vals))
	for _, v := range vals {
		old, newName, ok := strings.Cut(v, ":")
		if !ok || old == "" || newName == "" {
			return nil, fmt.Errorf("invalid --rename %q: expected old:new", v)
		}
		out[old] = newName
	}

	return out, nil
}

// --- Delta output ---

// writeDelta renders a delta to w in a compact, human-readable form.
func writeDelta(w io.Writer, d schemaDelta) {
	var b strings.Builder

	switch {
	case d.NewSchema:
		fmt.Fprintf(&b, "%s [%s]: new schema, %d field(s)\n", d.URI, d.Source, len(d.Added))
	case !d.hasDrift():
		fmt.Fprintf(&b, "%s [%s]: in sync\n", d.URI, d.Source)
	default:
		fmt.Fprintf(&b, "%s [%s]: drift\n", d.URI, d.Source)
	}

	for _, f := range d.Added {
		fmt.Fprintf(&b, "  + %s %s%s\n", f.Name, f.Type, requiredMark(f.Required))
	}
	for _, f := range d.Removed {
		fmt.Fprintf(&b, "  - %s %s%s\n", f.Name, f.Type, requiredMark(f.Required))
	}
	for _, c := range d.Changed {
		fmt.Fprintf(&b, "  ~ %s: %s -> %s\n", c.Name, c.From, c.To)
	}
	for _, r := range d.Renames {
		fmt.Fprintf(&b, "  rename? %s -> %s (%s)\n", r.From, r.To, r.By)
	}

	_, _ = io.WriteString(w, b.String())
}

func requiredMark(required bool) string {
	if required {
		return " (required)"
	}

	return ""
}

// --- Helpers ---

func summarize(name string, f schema.Field) fieldSummary {
	return fieldSummary{
		Name:     name,
		Type:     typeDisplay(f.Type),
		Required: f.Required,
	}
}

func fieldChanged(a, b schema.Field) bool {
	return a.Type != b.Type || a.Required != b.Required
}

func fieldDisplay(f schema.Field) string {
	return typeDisplay(f.Type) + requiredMark(f.Required)
}

// typeDisplay renders a core.Type as a lowercase name, expanding arrays to
// array<elem>.
func typeDisplay(t core.Type) string {
	if t.ID() == core.TIDArray && t.ElemTypeID() != "" {
		return "array<" + t.ElemTypeID().Lower() + ">"
	}

	return t.ID().Lower()
}

func sortedFieldNames(fields map[string]schema.Field) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func renameSets(renames []renamePair) (from, to map[string]struct{}) {
	from = make(map[string]struct{}, len(renames))
	to = make(map[string]struct{}, len(renames))

	for _, r := range renames {
		from[r.From] = struct{}{}
		to[r.To] = struct{}{}
	}

	return from, to
}
