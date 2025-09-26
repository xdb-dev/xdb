module github.com/xdb-dev/xdb/driver/xdbredis

go 1.25.0

replace (
	github.com/xdb-dev/xdb => ../..
	github.com/xdb-dev/xdb/codec/msgpack => ../../codec/msgpack
)

require (
	github.com/redis/go-redis/v9 v9.8.0
	github.com/stretchr/testify v1.10.0
	github.com/xdb-dev/xdb v0.0.0
	github.com/xdb-dev/xdb/codec/msgpack v0.0.0-20250918115628-3b35b981f3d0
)

require (
	github.com/brianvoe/gofakeit/v7 v7.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gojekfarm/xtools/errors v0.10.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
