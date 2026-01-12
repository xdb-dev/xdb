module github.com/xdb-dev/xdb

go 1.25.0

replace github.com/xdb-dev/xdb/store/xdbsqlite => ./store/xdbsqlite

require (
	github.com/brianvoe/gofakeit/v7 v7.2.1
	github.com/gojekfarm/xtools/errors v0.10.0
	github.com/spf13/cast v1.7.1
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
