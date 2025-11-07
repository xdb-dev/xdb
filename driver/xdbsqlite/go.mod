module github.com/xdb-dev/xdb/driver/xdbsqlite

go 1.25.0

replace github.com/xdb-dev/xdb => ../..

require (
	github.com/doug-martin/goqu/v9 v9.19.0
	github.com/gojekfarm/xtools/errors v0.10.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/stretchr/testify v1.11.1
	github.com/xdb-dev/xdb v0.0.0
)

require (
	github.com/brianvoe/gofakeit/v7 v7.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
