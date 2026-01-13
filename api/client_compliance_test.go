package api_test

import (
	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/store"
)

var (
	_ store.SchemaStore  = (*api.Client)(nil)
	_ store.TupleStore   = (*api.Client)(nil)
	_ store.RecordStore  = (*api.Client)(nil)
	_ store.HealthChecker = (*api.Client)(nil)
)
