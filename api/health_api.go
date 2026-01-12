package api

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/xdb-dev/xdb/store"
)

// HealthAPI provides health check endpoints for the XDB server.
type HealthAPI struct {
	store     any
	startTime time.Time
}

// NewHealthAPI creates a new HealthAPI instance with the given store and start time.
// The store can be any type; if it implements store.HealthChecker, health checks will be performed.
// The startTime is used to calculate server uptime.
func NewHealthAPI(store any, startTime time.Time) *HealthAPI {
	return &HealthAPI{
		store:     store,
		startTime: startTime,
	}
}

// GetHealth returns an endpoint function that performs health checks on the server and store.
// It collects process information (PID, uptime, goroutines) and optionally checks store health
// if the store implements the HealthChecker interface.
func (a *HealthAPI) GetHealth() EndpointFunc[HealthRequest, HealthResponse] {
	return func(ctx context.Context, req *HealthRequest) (*HealthResponse, error) {
		resp := &HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().Format(time.RFC3339),
			ProcessInfo: ProcessInfo{
				PID:        os.Getpid(),
				Uptime:     time.Since(a.startTime).String(),
				Goroutines: runtime.NumGoroutine(),
			},
		}

		storeHealth := StoreHealth{
			Status: "healthy",
		}

		if healthChecker, ok := a.store.(store.HealthChecker); ok {
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := healthChecker.Health(checkCtx); err != nil {
				storeHealth.Status = "unhealthy"
				errMsg := err.Error()
				storeHealth.Error = &errMsg
				resp.Status = "unhealthy"
			}
		}

		resp.StoreHealth = storeHealth

		return resp, nil
	}
}

// HealthRequest is the request type for the health endpoint.
type HealthRequest struct{}

// HealthResponse contains the health status of the server and store.
type HealthResponse struct {
	Status      string      `json:"status"`
	Timestamp   string      `json:"timestamp"`
	ProcessInfo ProcessInfo `json:"process_info"`
	StoreHealth StoreHealth `json:"store_health"`
}

// ProcessInfo contains information about the server process.
type ProcessInfo struct {
	PID        int    `json:"pid"`
	Uptime     string `json:"uptime"`
	Goroutines int    `json:"goroutines"`
}

// StoreHealth contains the health status of the underlying store.
type StoreHealth struct {
	Status string  `json:"status"`
	Error  *string `json:"error,omitempty"`
}
