package main

import (
	"context"
	"fmt"
	"os"

	xdbcli "github.com/xdb-dev/xdb/cmd/xdb/cli"
)

func main() {
	app := xdbcli.NewApp()

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
