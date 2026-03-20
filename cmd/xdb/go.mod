module github.com/xdb-dev/xdb/cmd/xdb

go 1.26

require (
	github.com/ncruces/go-sqlite3 v0.32.0
	github.com/redis/go-redis/v9 v9.18.0
	github.com/stretchr/testify v1.11.1
	github.com/urfave/cli/v3 v3.3.3
	github.com/xdb-dev/xdb v0.0.0
	github.com/xdb-dev/xdb/store/xdbredis v0.0.0-00010101000000-000000000000
	github.com/xdb-dev/xdb/store/xdbsqlite v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cel.dev/expr v0.25.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gojekfarm/xtools/errors v0.10.0 // indirect
	github.com/google/cel-go v0.27.0 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/sys v0.41.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace (
	github.com/xdb-dev/xdb => ../..
	github.com/xdb-dev/xdb/store/xdbredis => ../../store/xdbredis
	github.com/xdb-dev/xdb/store/xdbsqlite => ../../store/xdbsqlite
)
