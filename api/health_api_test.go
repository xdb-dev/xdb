package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// mockUnhealthyStore implements store.HealthChecker and always returns an error.
type mockUnhealthyStore struct {
	*xdbmemory.MemoryStore
}

func (m *mockUnhealthyStore) Health(ctx context.Context) error {
	return errors.New("mock store is unhealthy")
}

func TestHealthAPI_GetHealth_UptimeTracking(t *testing.T) {
	t.Parallel()

	store := xdbmemory.New()
	startTime := time.Now().Add(-5 * time.Second)
	healthAPI := api.NewHealthAPI(store, startTime)

	endpoint := healthAPI.GetHealth()
	req := &api.HealthRequest{}

	resp, err := endpoint(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.ProcessInfo.Uptime)

	duration, err := time.ParseDuration(resp.ProcessInfo.Uptime)
	require.NoError(t, err, "uptime should be a valid duration string")
	require.Greater(t, duration, 4*time.Second, "uptime should be at least 4 seconds")
	require.Less(t, duration, 10*time.Second, "uptime should be less than 10 seconds")
}

func TestHealthAPI_GetHealth(t *testing.T) {
	tests := []struct {
		name               string
		setupStore         func(t *testing.T) (store.HealthChecker, func())
		expectedStatus     string
		expectedStoreError bool
		verifyStoreError   func(t *testing.T, errMsg string)
	}{
		{
			name: "memory store without HealthChecker",
			setupStore: func(t *testing.T) (store.HealthChecker, func()) {
				return xdbmemory.New(), func() {}
			},
			expectedStatus:     "healthy",
			expectedStoreError: false,
		},
		{
			name: "store with HealthChecker - healthy",
			setupStore: func(t *testing.T) (store.HealthChecker, func()) {
				return xdbmemory.New(), func() {}
			},
			expectedStatus:     "healthy",
			expectedStoreError: false,
		},
		{
			name: "store with HealthChecker - unhealthy",
			setupStore: func(t *testing.T) (store.HealthChecker, func()) {
				s := &mockUnhealthyStore{
					MemoryStore: xdbmemory.New(),
				}
				return s, func() {}
			},
			expectedStatus:     "unhealthy",
			expectedStoreError: true,
			verifyStoreError: func(t *testing.T, errMsg string) {
				require.Contains(t, errMsg, "mock store is unhealthy")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, cleanup := tt.setupStore(t)
			defer cleanup()

			healthAPI := api.NewHealthAPI(s, time.Now())
			endpoint := healthAPI.GetHealth()
			req := &api.HealthRequest{}

			resp, err := endpoint(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, tt.expectedStatus, resp.Status)
			require.Equal(t, tt.expectedStatus, resp.StoreHealth.Status)

			if tt.expectedStoreError {
				require.NotNil(t, resp.StoreHealth.Error)
				if tt.verifyStoreError != nil {
					tt.verifyStoreError(t, *resp.StoreHealth.Error)
				}
			} else {
				require.Nil(t, resp.StoreHealth.Error)
			}

			require.Greater(t, resp.ProcessInfo.PID, 0)
			require.Greater(t, resp.ProcessInfo.Goroutines, 0)
			require.NotEmpty(t, resp.ProcessInfo.Uptime)

			_, err = time.Parse(time.RFC3339, resp.Timestamp)
			require.NoError(t, err, "timestamp should be valid RFC3339 format")
		})
	}
}
