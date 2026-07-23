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
// Version is the record's version after the change. It is carried
// explicitly rather than left inside Data for two reasons: delete events
// have no Data to read it from, and a consumer should not have to parse
// a payload to order events.
//
// Delivery is at-most-once and lossy — a subscriber that falls behind
// misses events silently. Versions make that loss detectable: a jump
// from 3 to 7 means writes were dropped and the record should be
// re-read. Schema events carry 0.
type WatchEvent struct {
	TS      time.Time       `json:"ts"`
	Type    string          `json:"type"`
	URI     string          `json:"uri"`
	Data    json.RawMessage `json:"data,omitempty"`
	Version int64           `json:"version,omitempty"`
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
