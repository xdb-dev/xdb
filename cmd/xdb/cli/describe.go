package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
)

func (a *App) describeCmd() *cli.Command {
	return &cli.Command{
		Name:               "describe",
		Usage:              "Introspect methods, types, and data schemas",
		Category:           "agent",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[method.name | TypeName]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "methods", Usage: "List all methods"},
			&cli.BoolFlag{Name: "types", Usage: "List all types"},
			&cli.BoolFlag{Name: "value-types", Usage: "List supported value types"},
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

	if cmd.Bool("types") {
		return a.listTypes(ctx, cmd)
	}

	if cmd.Bool("value-types") {
		return listValueTypes(cmd)
	}

	if uri := cmd.String("uri"); uri != "" {
		return a.describeDataSchema(ctx, cmd, uri)
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("specify a method name, type name, or use --methods/--types/--value-types")
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
