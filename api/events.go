package api

import (
	"sync"

	"github.com/xdb-dev/xdb/core"
)

// ServiceOption configures a service at construction.
type ServiceOption func(*serviceOptions)

type serviceOptions struct {
	events *Bus
}

// WithEvents wires an event [Bus] into a service: mutations publish
// change notifications after they succeed (post-commit for batches).
func WithEvents(bus *Bus) ServiceOption {
	return func(o *serviceOptions) {
		o.events = bus
	}
}

func applyServiceOptions(opts []ServiceOption) serviceOptions {
	var o serviceOptions
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// subscriberBuffer is the per-subscriber channel capacity. A subscriber
// that falls further behind than this drops events (at-most-once
// delivery; the drop is counted, never blocking the publisher).
const subscriberBuffer = 64

// Bus is an in-process event bus for change notifications. Services
// publish after successful mutations; watch streams subscribe with a
// URI scope. Delivery is at-most-once with no replay: events published
// before Subscribe or after a buffer overflow are not seen.
type Bus struct {
	subs    map[int]*subscriber
	nextID  int
	dropped int64
	mu      sync.Mutex
	closed  bool
}

type subscriber struct {
	scope *core.URI
	ch    chan WatchEvent
}

// NewBus creates an empty event bus.
func NewBus() *Bus {
	return &Bus{subs: make(map[int]*subscriber)}
}

// Publish delivers the event to every subscriber whose scope matches.
// It never blocks: a subscriber with a full buffer misses the event.
func (b *Bus) Publish(e WatchEvent) {
	uri, err := core.ParseURI(e.URI)
	if err != nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	for _, sub := range b.subs {
		if !matchScope(sub.scope, uri) {
			continue
		}

		select {
		case sub.ch <- e:
		default:
			b.dropped++
		}
	}
}

// Subscribe registers a subscriber for events under scope. The cancel
// function unregisters it and closes the channel.
func (b *Bus) Subscribe(scope *core.URI) (<-chan WatchEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++

	sub := &subscriber{
		scope: scope,
		ch:    make(chan WatchEvent, subscriberBuffer),
	}

	if b.closed {
		close(sub.ch)
		return sub.ch, func() {}
	}

	b.subs[id] = sub

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if _, ok := b.subs[id]; ok {
			delete(b.subs, id)
			close(sub.ch)
		}
	}

	return sub.ch, cancel
}

// Close closes every subscriber channel and rejects future publishes.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true

	for id, sub := range b.subs {
		delete(b.subs, id)
		close(sub.ch)
	}
}

// matchScope reports whether an event URI falls under a subscription
// scope, comparing component-wise (never by string prefix): the
// namespace must match, and schema/id components constrain the match
// only when the scope declares them.
func matchScope(scope, event *core.URI) bool {
	if scope.NS() != event.NS() {
		return false
	}

	if scope.Schema() != "" && scope.Schema() != event.Schema() {
		return false
	}

	if scope.ID() != "" && scope.ID() != event.ID() {
		return false
	}

	return true
}
