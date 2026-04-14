package main

import (
	"context"
	"os"

	xdbcli "github.com/xdb-dev/xdb/cmd/xdb/cli"
)

func main() {
	app := xdbcli.NewApp()

	if err := app.Run(context.Background(), os.Args); err != nil {
		// The root command's ExitErrHandler already rendered the error to
		// stderr using the live --output flag; here we just translate to a
		// stable exit code.
		os.Exit(xdbcli.ExitCodeFor(err))
	}
}
