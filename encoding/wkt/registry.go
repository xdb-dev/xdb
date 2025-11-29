package wkt

import (
	"fmt"
	"reflect"
	"sync"
)

// MarshalFunc converts a Go value to a format suitable for storage in core.Value.
// The returned value can be any type that core.Value supports (string, int64, float64, bool, []byte, etc.).
type MarshalFunc func(v reflect.Value) (any, error)

// UnmarshalFunc populates a Go value from a stored value.
// The input v is the value retrieved from core.Value, and target is the destination reflect.Value.
type UnmarshalFunc func(v any, target reflect.Value) error

// Registry manages custom type handlers for well-known types.
// It maps Go types to their marshal/unmarshal functions, allowing custom
// encoding behavior for specific types.
type Registry struct {
	mu      sync.RWMutex
	types   map[reflect.Type]*typeHandler
	enabled bool
}

type typeHandler struct {
	marshal   MarshalFunc
	unmarshal UnmarshalFunc
}

// NewRegistry creates a new empty registry.
// Use Register to add custom type handlers.
func NewRegistry() *Registry {
	return &Registry{
		types:   make(map[reflect.Type]*typeHandler),
		enabled: true,
	}
}

// Register adds a custom type handler to the registry.
// If a type is already registered, it will be replaced with the new handler.
//
// The marshal function converts a reflect.Value to a storable value.
// The unmarshal function populates a reflect.Value from a stored value.
//
// Example:
//
//	registry.Register(
//	    reflect.TypeOf(Money{}),
//	    func(v reflect.Value) (any, error) {
//	        m := v.Interface().(Money)
//	        return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
//	    },
//	    func(v any, target reflect.Value) error {
//	        str := v.(string)
//	        // Parse and set Money value
//	        return nil
//	    },
//	)
func (r *Registry) Register(typ reflect.Type, marshal MarshalFunc, unmarshal UnmarshalFunc) {
	if typ == nil {
		panic("wkt: cannot register nil type")
	}
	if marshal == nil {
		panic("wkt: marshal function cannot be nil")
	}
	if unmarshal == nil {
		panic("wkt: unmarshal function cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.types[typ] = &typeHandler{
		marshal:   marshal,
		unmarshal: unmarshal,
	}
}

// Unregister removes a type handler from the registry.
// Returns true if the type was registered and removed, false otherwise.
func (r *Registry) Unregister(typ reflect.Type) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.types[typ]; exists {
		delete(r.types, typ)
		return true
	}
	return false
}

// IsRegistered checks if a type has a registered handler.
func (r *Registry) IsRegistered(typ reflect.Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.types[typ]
	return exists
}

// Marshal converts a value using its registered marshal function.
// Returns an error if the type is not registered or if marshaling fails.
func (r *Registry) Marshal(v reflect.Value) (any, error) {
	r.mu.RLock()
	handler, exists := r.types[v.Type()]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("wkt: type %v is not registered", v.Type())
	}

	return handler.marshal(v)
}

// Unmarshal populates a value using its registered unmarshal function.
// Returns an error if the type is not registered or if unmarshaling fails.
func (r *Registry) Unmarshal(v any, target reflect.Value) error {
	r.mu.RLock()
	handler, exists := r.types[target.Type()]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("wkt: type %v is not registered", target.Type())
	}

	return handler.unmarshal(v, target)
}

// Enable activates the registry for use.
// A disabled registry will not be consulted during encoding/decoding.
func (r *Registry) Enable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = true
}

// Disable deactivates the registry.
// A disabled registry will not be consulted during encoding/decoding.
func (r *Registry) Disable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = false
}

// IsEnabled returns whether the registry is currently enabled.
func (r *Registry) IsEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled
}

// Count returns the number of registered types.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.types)
}

// Clear removes all registered type handlers.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types = make(map[reflect.Type]*typeHandler)
}
