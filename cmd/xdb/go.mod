module github.com/xdb-dev/xdb/cmd/xdb

go 1.26

require (
	github.com/stretchr/testify v1.11.1
	github.com/urfave/cli/v3 v3.3.3
	github.com/xdb-dev/xdb v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gojekfarm/xtools/errors v0.10.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)

replace github.com/xdb-dev/xdb => ../..
