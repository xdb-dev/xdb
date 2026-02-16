package client_test

import (
	"github.com/xdb-dev/xdb/client"
	"github.com/xdb-dev/xdb/store"
)

var (
	_ store.SchemaStore   = (*client.Client)(nil)
	_ store.TupleStore    = (*client.Client)(nil)
	_ store.RecordStore   = (*client.Client)(nil)
	_ store.HealthChecker = (*client.Client)(nil)
)
