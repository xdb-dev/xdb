module github.com/xdb-dev/xdb/cmd/xdb

go 1.25.1

replace (
	github.com/xdb-dev/xdb => ../../
	github.com/xdb-dev/xdb/driver/xdbsqlite => ../../driver/xdbsqlite
)

require (
	github.com/gojekfarm/xtools/xapi v0.0.0-20251010114542-6ce2a14093c8
	github.com/gojekfarm/xtools/xload v0.10.0
	github.com/rs/zerolog v1.34.0
	github.com/urfave/cli/v2 v2.27.7
	github.com/xdb-dev/xdb v0.0.0
	github.com/xdb-dev/xdb/driver/xdbsqlite v0.0.0-00010101000000-000000000000
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/doug-martin/goqu/v9 v9.19.0 // indirect
	github.com/gojekfarm/xtools/errors v0.10.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)
