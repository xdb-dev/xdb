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
		Usage:              "Introspect actions, types, filters, errors, and data schemas",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[resource.action | TypeName]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "methods", Usage: "List all actions (dotted RPC form)"},
			&cli.BoolFlag{Name: "actions", Usage: "Show action \u00d7 resource matrix"},
			&cli.BoolFlag{Name: "types", Usage: "List all types"},
			&cli.BoolFlag{Name: "value-types", Usage: "List supported value types"},
			&cli.BoolFlag{Name: "filter", Usage: "Show CEL filter grammar"},
			&cli.BoolFlag{Name: "errors", Usage: "List error codes"},
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

	if uri := cmd.String("uri"); uri != "" {
		return a.describeDataSchema(ctx, cmd, uri)
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("specify a resource.action name, type name, or use --actions/--types/--value-types/--filter/--errors")
	}

	name := args.First()

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
