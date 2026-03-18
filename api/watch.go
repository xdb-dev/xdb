package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xdb-dev/xdb/store"
)

// WatchService provides change-stream operations.
type WatchService struct {
	store store.Store
}

// NewWatchService creates a [WatchService] backed by the given [store.Store].
func NewWatchService(s store.Store) *WatchService {
	return &WatchService{store: s}
}

// WatchRequest is the request for watch.
type WatchRequest struct {
	URI string `json:"uri"`
}

// Watch streams change events for the given URI.
// It calls send to push each event to the client and returns when
// the context is canceled or the stream ends.
func (s *WatchService) Watch(_ context.Context, _ *WatchRequest, _ func(string, json.RawMessage)) error {
	return fmt.Errorf("api: watch not implemented")
}

// WatchEvent represents a single change notification.
type WatchEvent struct {
	Data any    `json:"data,omitempty"`
	Type string `json:"type"`
	URI  string `json:"uri"`
}
