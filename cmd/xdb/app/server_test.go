package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
)

func TestHealthEndpointIntegration(t *testing.T) {
	t.Run("health endpoint returns healthy response via HTTP", func(t *testing.T) {
		// Arrange: Create test server configuration
		cfg := &Config{
			Daemon: DaemonConfig{
				Addr:   "",
				Socket: "",
			},
		}

		// Create server
		server, err := NewServer(cfg)
		require.NoError(t, err, "failed to create server")

		// Start test HTTP server
		testServer := httptest.NewServer(server.mux)
		defer testServer.Close()

		// Act: Make HTTP GET request to health endpoint with empty JSON body
		client := &http.Client{
			Timeout: 2 * time.Second,
		}

		// xapi requires JSON body even for GET requests
		req, err := http.NewRequest("GET", testServer.URL+"/v1/health", bytes.NewReader([]byte("{}")))
		require.NoError(t, err, "failed to create request")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)

		// Assert: Verify HTTP response
		require.NoError(t, err, "failed to make HTTP request")
		defer resp.Body.Close()

		// Check HTTP status code is 200
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK status")

		// Check response is valid JSON
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/json", "expected JSON content type")

		// Parse and unmarshal response body to HealthResponse struct
		var healthResp api.HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		require.NoError(t, err, "failed to decode JSON response")

		// Verify response status field is "healthy"
		assert.Equal(t, "healthy", healthResp.Status, "expected healthy status")

		// Verify process info fields are populated
		assert.Greater(t, healthResp.ProcessInfo.PID, 0, "PID should be positive")
		assert.NotEmpty(t, healthResp.ProcessInfo.Uptime, "uptime should be populated")
		assert.Greater(t, healthResp.ProcessInfo.Goroutines, 0, "goroutines should be positive")

		// Verify store health status is "healthy"
		assert.Equal(t, "healthy", healthResp.StoreHealth.Status, "store should be healthy")
		assert.Nil(t, healthResp.StoreHealth.Error, "store should have no errors")

		// Verify timestamp is populated and valid
		assert.NotEmpty(t, healthResp.Timestamp, "timestamp should be populated")
		_, err = time.Parse(time.RFC3339, healthResp.Timestamp)
		assert.NoError(t, err, "timestamp should be valid RFC3339 format")
	})

	t.Run("health endpoint via direct handler invocation", func(t *testing.T) {
		// Arrange: Create test server
		cfg := &Config{
			Daemon: DaemonConfig{
				Addr:   "",
				Socket: "",
			},
		}

		server, err := NewServer(cfg)
		require.NoError(t, err, "failed to create server")

		// Act: Make request directly to the mux using httptest.ResponseRecorder
		// xapi requires JSON body even for GET requests
		req := httptest.NewRequest("GET", "/v1/health", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		server.mux.ServeHTTP(rr, req)

		// Assert: Verify response
		assert.Equal(t, http.StatusOK, rr.Code, "expected 200 OK status")
		assert.NotEmpty(t, rr.Body.Bytes(), "response body should not be empty")

		// Parse response
		var healthResp api.HealthResponse
		err = json.Unmarshal(rr.Body.Bytes(), &healthResp)
		require.NoError(t, err, "failed to decode JSON response")

		// Verify all response fields
		assert.Equal(t, "healthy", healthResp.Status, "expected healthy status")
		assert.NotEmpty(t, healthResp.Timestamp, "timestamp should be populated")
		assert.Greater(t, healthResp.ProcessInfo.PID, 0, "PID should be positive")
		assert.NotEmpty(t, healthResp.ProcessInfo.Uptime, "uptime should be populated")
		assert.Greater(t, healthResp.ProcessInfo.Goroutines, 0, "goroutines should be positive")
		assert.Equal(t, "healthy", healthResp.StoreHealth.Status, "store should be healthy")
		assert.Nil(t, healthResp.StoreHealth.Error, "store should have no errors")
	})
}
