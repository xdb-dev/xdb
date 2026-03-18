package api

import (
	"context"
)

// SystemService provides system operations (health, version).
type SystemService struct {
	version string
}

// NewSystemService creates a [SystemService] with the given version string.
func NewSystemService(version string) *SystemService {
	return &SystemService{version: version}
}

// HealthRequest is the request for system.health.
type HealthRequest struct{}

// HealthResponse is the response for system.health.
type HealthResponse struct {
	Status string `json:"status"`
}

// Health reports the system health status.
func (s *SystemService) Health(_ context.Context, _ *HealthRequest) (*HealthResponse, error) {
	return &HealthResponse{Status: "ok"}, nil
}

// VersionRequest is the request for system.version.
type VersionRequest struct{}

// VersionResponse is the response for system.version.
type VersionResponse struct {
	Version string `json:"version"`
}

// Version reports the system version.
func (s *SystemService) Version(_ context.Context, _ *VersionRequest) (*VersionResponse, error) {
	return &VersionResponse{Version: s.version}, nil
}
