module github.com/xdb-dev/xdb/driver/xdbsqlite

go 1.24.1

replace (
	github.com/xdb-dev/xdb => ../..
	github.com/xdb-dev/xdb/codec/msgpack => ../../codec/msgpack
)

require (
	github.com/doug-martin/goqu/v9 v9.19.0
	github.com/gojekfarm/xtools/errors v0.10.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/spf13/cast v1.7.1
	github.com/stretchr/testify v1.10.0
	github.com/xdb-dev/xdb v0.0.0
	github.com/xdb-dev/xdb/codec/msgpack v0.0.0-00010101000000-000000000000
)

require (
	github.com/brianvoe/gofakeit/v7 v7.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
