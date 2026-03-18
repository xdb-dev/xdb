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

// methodInfo holds rich metadata for a method.
type methodInfo struct {
	Summary  string   `json:"summary"`
	Example  string   `json:"example"`
	Params   []string `json:"params"`
	Flags    []string `json:"flags"`
	Mutating bool     `json:"mutating"`
}

var methodDescriptions = map[string]methodInfo{
	"records.create": {
		Summary:  "Create a new record. Idempotent: returns existing if already exists.",
		Params:   []string{"--uri (required)"},
		Flags:    []string{"--json", "--file", "--dry-run", "--output", "--quiet"},
		Mutating: true,
		Example:  `xdb records create --uri xdb://com.example/posts/post-1 --json '{"title":"Hello"}'`,
	},
	"records.get": {
		Summary: "Retrieve a record by URI.",
		Params:  []string{"--uri (required)"},
		Flags:   []string{"--fields", "--output"},
		Example: `xdb records get --uri xdb://com.example/posts/post-1 --fields title,author`,
	},
	"records.list": {
		Summary: "List records matching a query.",
		Params:  []string{"--uri (required)"},
		Flags:   []string{"--filter", "--query", "--fields", "--limit", "--offset", "--page-all", "--output"},
		Example: `xdb records list --uri xdb://com.example/posts --filter "age>30" --fields id,name --limit 10`,
	},
	"records.update": {
		Summary:  "Update an existing record (patch semantics). Only supplied fields change.",
		Params:   []string{"--uri (required)"},
		Flags:    []string{"--json", "--file", "--dry-run", "--output", "--quiet"},
		Mutating: true,
		Example:  `xdb records update --uri xdb://com.example/posts/post-1 --json '{"title":"New Title"}'`,
	},
	"records.upsert": {
		Summary:  "Create or replace a record (full replace). Sets complete state.",
		Params:   []string{"--uri (required)"},
		Flags:    []string{"--json", "--file", "--dry-run", "--output", "--quiet"},
		Mutating: true,
		Example:  `xdb records upsert --uri xdb://com.example/posts/post-1 --json '{"title":"Full State"}'`,
	},
	"records.delete": {
		Summary:  "Delete a record. Idempotent: succeeds even if not found.",
		Params:   []string{"--uri (required)", "--force (required)"},
		Flags:    []string{"--dry-run", "--output", "--quiet"},
		Mutating: true,
		Example:  `xdb records delete --uri xdb://com.example/posts/post-1 --force`,
	},
	"schemas.create": {
		Summary:  "Create a new schema definition. Idempotent.",
		Params:   []string{"--uri (required)"},
		Flags:    []string{"--json", "--file", "--dry-run", "--output"},
		Mutating: true,
		Example:  `xdb schemas create --uri xdb://com.example/posts --json '{"Fields":{"title":{"Type":"string"}}}'`,
	},
	"schemas.get": {
		Summary: "Retrieve a schema definition by URI.",
		Params:  []string{"--uri (required)"},
		Flags:   []string{"--output"},
		Example: `xdb schemas get --uri xdb://com.example/posts`,
	},
	"schemas.list": {
		Summary: "List schemas in a namespace.",
		Params:  []string{"--uri"},
		Flags:   []string{"--limit", "--offset", "--output"},
		Example: `xdb schemas list --uri xdb://com.example`,
	},
	"schemas.update": {
		Summary:  "Update a schema definition (patch semantics).",
		Params:   []string{"--uri (required)"},
		Flags:    []string{"--json", "--file", "--dry-run", "--output"},
		Mutating: true,
		Example:  `xdb schemas update --uri xdb://com.example/posts --json '{"Fields":{"author":{"Type":"string"}}}'`,
	},
	"schemas.delete": {
		Summary:  "Delete a schema. Idempotent.",
		Params:   []string{"--uri (required)", "--force (required)"},
		Flags:    []string{"--cascade", "--dry-run", "--output"},
		Mutating: true,
		Example:  `xdb schemas delete --uri xdb://com.example/posts --force`,
	},
	"namespaces.get": {
		Summary: "Retrieve namespace metadata by URI.",
		Params:  []string{"--uri (required)"},
		Flags:   []string{"--output"},
		Example: `xdb namespaces get --uri xdb://com.example`,
	},
	"namespaces.list": {
		Summary: "List all known namespaces.",
		Flags:   []string{"--limit", "--offset", "--output"},
		Example: `xdb namespaces list`,
	},
	"batch.execute": {
		Summary:  "Run multiple operations in a single atomic transaction.",
		Flags:    []string{"--json", "--file", "--dry-run", "--output"},
		Mutating: true,
		Example:  `xdb batch --json '[{"method":"records.create","uri":"xdb://ns/schema/id","body":{}}]'`,
	},
	"system.health": {
		Summary: "Report system health status.",
		Example: `xdb describe system.health`,
	},
	"system.version": {
		Summary: "Report system version.",
		Example: `xdb describe system.version`,
	},
}

var typeDescriptions = map[string]string{
	"Record":    "A collection of tuples sharing the same ID within a schema.",
	"Schema":    "A definition of attributes and their types for a schema.",
	"Namespace": "A logical grouping of schemas (e.g., com.example).",
	"Tuple":     "A single attribute-value pair within a record.",
	"Value":     "A typed value (string, integer, float, bool, time, bytes).",
	"URI":       "A reference to XDB data: xdb://NS/SCHEMA/ID#ATTR",
}

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

func (a *App) schemaInspect(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("methods") {
		return listMethods(cmd)
	}

	if cmd.Bool("types") {
		return listTypes(cmd)
	}

	if cmd.Bool("value-types") {
		return listValueTypes(cmd)
	}

	if uri := cmd.String("uri"); uri != "" {
		return a.describeDataSchema(cmd, uri)
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("specify a method name, type name, or use --methods/--types/--value-types")
	}

	name := args.First()

	if strings.Contains(name, ".") {
		return describeMethod(cmd, name)
	}

	return describeType(cmd, name)
}

func listMethods(cmd *cli.Command) error {
	names := make([]string, 0, len(methodDescriptions))
	for name := range methodDescriptions {
		names = append(names, name)
	}

	sort.Strings(names)

	items := make([]any, len(names))
	for i, name := range names {
		items[i] = map[string]string{
			"method":  name,
			"summary": methodDescriptions[name].Summary,
		}
	}

	return formatList(cmd, items)
}

func listTypes(cmd *cli.Command) error {
	names := make([]string, 0, len(typeDescriptions))
	for name := range typeDescriptions {
		names = append(names, name)
	}

	sort.Strings(names)

	items := make([]any, len(names))
	for i, name := range names {
		items[i] = map[string]string{
			"type":        name,
			"description": typeDescriptions[name],
		}
	}

	return formatList(cmd, items)
}

func listValueTypes(cmd *cli.Command) error {
	types := []map[string]string{
		{"type": "string", "go": "string"},
		{"type": "integer", "go": "int64"},
		{"type": "unsigned", "go": "uint64"},
		{"type": "float", "go": "float64"},
		{"type": "bool", "go": "bool"},
		{"type": "time", "go": "time.Time"},
		{"type": "bytes", "go": "[]byte"},
	}

	items := make([]any, len(types))
	for i, t := range types {
		items[i] = t
	}

	return formatList(cmd, items)
}

func describeMethod(cmd *cli.Command, name string) error {
	info, ok := methodDescriptions[name]
	if !ok {
		return fmt.Errorf("unknown method: %s", name)
	}

	result := map[string]any{
		"kind":     "MethodDescription",
		"method":   name,
		"summary":  info.Summary,
		"mutating": info.Mutating,
		"example":  info.Example,
	}

	if len(info.Params) > 0 {
		result["params"] = info.Params
	}

	if len(info.Flags) > 0 {
		result["flags"] = info.Flags
	}

	return formatOne(cmd, result)
}

func describeType(cmd *cli.Command, name string) error {
	desc, ok := typeDescriptions[name]
	if !ok {
		return fmt.Errorf("unknown type: %s", name)
	}

	return formatOne(cmd, map[string]any{
		"kind":        "TypeDescription",
		"type":        name,
		"description": desc,
	})
}

func (a *App) describeDataSchema(cmd *cli.Command, raw string) error {
	uri, err := core.ParseURI(raw)
	if err != nil {
		return err
	}

	ctx := context.Background()

	resp, err := a.schemas.Get(ctx, &api.GetSchemaRequest{
		URI: uri.String(),
	})
	if err != nil {
		return err
	}

	return formatOne(cmd, map[string]any{
		"kind": "DataSchemaDescription",
		"uri":  uri.String(),
		"data": resp.Data,
	})
}
