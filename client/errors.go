package client

import "github.com/gojekfarm/xtools/errors"

var (
	ErrNoStoresConfigured       = errors.New("[xdb/client] at least one store must be configured")
	ErrNoAddressConfigured      = errors.New("[xdb/client] address or socket path must be configured")
	ErrSchemaStoreNotConfigured = errors.New("[xdb/client] schema store not configured")
	ErrTupleStoreNotConfigured  = errors.New("[xdb/client] tuple store not configured")
	ErrRecordStoreNotConfigured = errors.New("[xdb/client] record store not configured")
	ErrHealthStoreNotConfigured = errors.New("[xdb/client] health store not configured")
)
