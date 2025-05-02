module github.com/xdb-dev/xdb/encoding/xdbproto

go 1.24.1

require (
	github.com/xdb-dev/xdb v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.31.0
)

require github.com/spf13/cast v1.7.1 // indirect

replace github.com/xdb-dev/xdb => ../..
