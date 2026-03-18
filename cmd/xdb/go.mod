module github.com/xdb-dev/xdb/cmd/xdb

go 1.26

require (
	github.com/urfave/cli/v3 v3.3.3
	github.com/xdb-dev/xdb v0.0.0-00010101000000-000000000000
)

require github.com/gojekfarm/xtools/errors v0.10.0 // indirect

replace github.com/xdb-dev/xdb => ../..
