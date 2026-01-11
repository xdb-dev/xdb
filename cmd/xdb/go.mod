module github.com/xdb-dev/xdb/cmd/xdb

go 1.25.1

replace (
	github.com/xdb-dev/xdb => ../../
	github.com/xdb-dev/xdb/store/xdbsqlite => ../../store/xdbsqlite
)

require (
	github.com/gojekfarm/xtools/xapi v0.11.0-alpha.1
	github.com/jedib0t/go-pretty/v6 v6.7.8
	github.com/phsym/console-slog v0.3.1
	github.com/stretchr/testify v1.11.1
	github.com/urfave/cli/v3 v3.5.0
	github.com/xdb-dev/xdb v0.0.0
	golang.org/x/term v0.38.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gojekfarm/xtools/errors v0.10.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)
