package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xdb-dev/xdb/core"
)

// WatchService streams change notifications from an event [Bus].
type WatchService struct {
	bus *Bus
}

// NewWatchService creates a [WatchService] over the given event bus.
func NewWatchService(bus *Bus) *WatchService {
	return &WatchService{bus: bus}
}

// WatchRequest is the request for watch.
type WatchRequest struct {
	URI string `json:"uri"`
}

// WatchEvent represents a single change notification.
type WatchEvent struct {
	TS   time.Time       `json:"ts"`
	Type string          `json:"type"`
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Watch streams change events for the given URI scope. The first frame
// is always "ready" (subscription live); each matching change arrives
// as an "event" frame. Delivery is at-most-once with no replay. The
// stream ends cleanly when the context is canceled or the daemon
// stops.
func (s *WatchService) Watch(
	ctx context.Context,
	req *WatchRequest,
	send func(string, json.RawMessage),
) error {
	uri, err := parseURI(req.URI, "watch", 1, 3, true)
	if err != nil {
		return err
	}

	if s.bus == nil {
		return fmt.Errorf("%w: watch requires an event bus", core.ErrNotImplemented)
	}

	ch, cancel := s.bus.Subscribe(uri)
	defer cancel()

	ready, err := json.Marshal(map[string]string{"uri": uri.String()})
	if err != nil {
		return fmt.Errorf("api: watch: marshal ready frame: %w", err)
	}
	send("ready", ready)

	for {
		select {
		case <-ctx.Done():
			return nil

		case e, ok := <-ch:
			if !ok {
				return nil
			}

			payload, marshalErr := json.Marshal(e)
			if marshalErr != nil {
				continue
			}

			send("event", payload)
		}
	}
}
