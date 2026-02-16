package xdbsqlite_test

import (
	"context"
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store/xdbsqlite"
)

func TestStoreHealth(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(t *testing.T) *xdbsqlite.Store
		wantErr     bool
		errContains string
	}{
		{
			name: "health check on healthy database",
			setupStore: func(t *testing.T) *xdbsqlite.Store {
				cfg := xdbsqlite.Config{
					InMemory: true,
				}
				store, err := xdbsqlite.New(cfg)
				require.NoError(t, err)
				return store
			},
			wantErr: false,
		},
		{
			name: "health check on closed database",
			setupStore: func(t *testing.T) *xdbsqlite.Store {
				cfg := xdbsqlite.Config{
					InMemory: true,
				}
				store, err := xdbsqlite.New(cfg)
				require.NoError(t, err)
				err = store.Close()
				require.NoError(t, err)
				return store
			},
			wantErr:     true,
			errContains: "sql: database is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore(t)
			if !tt.wantErr {
				defer store.Close()
			}

			ctx := context.Background()
			err := store.Health(ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
