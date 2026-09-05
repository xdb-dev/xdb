package cli

import (
	"context"
	"encoding/json"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/rpc/client"
)

func (a *App) watchCmd() *cli.Command {
	return &cli.Command{
		Name:               "watch",
		Usage:              "Stream change notifications as NDJSON",
		Category:           "operations",
		CustomHelpTemplate: commandHelpTemplate,
		ArgsUsage:          "[URI]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "uri", Usage: "URI scope to watch (namespace, schema, or record)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Ignored; watch always prints NDJSON"},
		},
		Action: a.watchAction,
	}
}

// watchAction streams change events as NDJSON: a {"ready":...} line
// once the subscription is live, then one event object per change.
// The stream ends cleanly when the daemon stops.
func (a *App) watchAction(ctx context.Context, cmd *cli.Command) error {
	uri, err := getURI(cmd)
	if err != nil {
		return invalidArgError("watch", "stream", err)
	}

	w := cmd.Root().Writer

	err = a.client.Stream(ctx, "watch", &api.WatchRequest{URI: uri}, func(f client.StreamFrame) error {
		switch f.Event {
		case "ready":
			line, marshalErr := json.Marshal(map[string]any{"ready": true, "uri": uri})
			if marshalErr != nil {
				return marshalErr
			}
			_, writeErr := w.Write(append(line, '\n'))
			return writeErr

		case "event":
			_, writeErr := w.Write(append([]byte(f.Data), '\n'))
			return writeErr

		default:
			return nil
		}
	})
	if err != nil {
		return wrapRPCError("watch", "stream", uri, err)
	}

	return nil
}
