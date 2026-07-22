package main

import (
	"context"
	"os"

	xdbcli "github.com/xdb-dev/xdb/cmd/xdb/cli"
)

func main() {
	app := xdbcli.NewApp()

	err := app.Run(context.Background(), os.Args)

	// The root command's ExitErrHandler already rendered most errors to
	// stderr using the live --output flag; [xdbcli.FinalizeError] renders
	// the few that escape it (e.g. urfave's own "No help topic for X").
	xdbcli.FinalizeError(app, err)

	os.Exit(xdbcli.ExitCodeFor(err))
}
