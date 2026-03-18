package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/api"
)

func TestSystemService_Health(t *testing.T) {
	svc := api.NewSystemService("1.0.0")

	resp, err := svc.Health(context.Background(), &api.HealthRequest{})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestSystemService_Version(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "semantic version",
			version: "1.2.3",
		},
		{
			name:    "dev version",
			version: "dev",
		},
		{
			name:    "empty version",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := api.NewSystemService(tt.version)

			resp, err := svc.Version(context.Background(), &api.VersionRequest{})
			require.NoError(t, err)
			assert.Equal(t, tt.version, resp.Version)
		})
	}
}
