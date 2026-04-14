package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
)

func (a *App) namespacesCmd() *cli.Command {
	return &cli.Command{
		Name:               "namespaces",
		Usage:              "List and inspect namespaces",
		Category:           "resources",
		CustomHelpTemplate: subcommandHelpTemplate,
		Commands: []*cli.Command{
			{
				Name:               "list",
				Usage:              "List all namespaces",
				CustomHelpTemplate: commandHelpTemplate,
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "limit", Usage: "Max items per page"},
					&cli.IntFlag{Name: "offset", Usage: "Page offset"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
				},
				Action: a.namespaceList,
			},
			{
				Name:               "get",
				Usage:              "Get namespace details",
				CustomHelpTemplate: commandHelpTemplate,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "uri", Usage: "Namespace URI"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
				},
				Action: a.namespaceGet,
			},
		},
	}
}

func (a *App) namespaceList(ctx context.Context, cmd *cli.Command) error {
	var resp api.ListNamespacesResponse
	if err := a.client.Call(ctx, "namespaces.list", &api.ListNamespacesRequest{
		Limit:  int(cmd.Int("limit")),
		Offset: int(cmd.Int("offset")),
	}, &resp); err != nil {
		return wrapRPCError("namespaces", "list", "", err)
	}

	items := make([]any, len(resp.Items))
	for i, ns := range resp.Items {
		items[i] = map[string]string{"namespace": ns.String()}
	}

	return formatList(cmd, items)
}

func (a *App) namespaceGet(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("namespaces", "get", err)
	}

	var resp api.GetNamespaceResponse
	if err := a.client.Call(ctx, "namespaces.get", &api.GetNamespaceRequest{
		URI: uri,
	}, &resp); err != nil {
		return wrapRPCError("namespaces", "get", uri, err)
	}

	return formatOne(cmd, map[string]string{
		"namespace": resp.Data.String(),
	})
}
