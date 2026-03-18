package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler is the core contract for a transport-agnostic endpoint.
// TReq is the request type (decoded from JSON), TRes is the response type.
type Handler[TReq, TRes any] interface {
	Handle(ctx context.Context, req *TReq) (*TRes, error)
}

// HandlerFunc is a function adapter for [Handler].
type HandlerFunc[TReq, TRes any] func(ctx context.Context, req *TReq) (*TRes, error)

// Handle calls the function.
func (f HandlerFunc[TReq, TRes]) Handle(ctx context.Context, req *TReq) (*TRes, error) {
	return f(ctx, req)
}

// Endpoint wraps a [Handler] and implements [http.Handler].
// It handles JSON decoding, optional validation, optional extraction,
// and JSON encoding of the response.
type Endpoint[TReq, TRes any] struct {
	handler      Handler[TReq, TRes]
	errorHandler ErrorHandler
	middleware   MiddlewareStack
}

// NewEndpoint creates a new [Endpoint] with the given handler and options.
func NewEndpoint[TReq, TRes any](
	h Handler[TReq, TRes],
	opts ...EndpointOption,
) *Endpoint[TReq, TRes] {
	o := defaultOptions()

	for _, opt := range opts {
		opt.apply(o)
	}

	return &Endpoint[TReq, TRes]{
		handler:      h,
		errorHandler: o.errorHandler,
		middleware:   o.middleware,
	}
}

// ServeHTTP implements [http.Handler].
func (e *Endpoint[TReq, TRes]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req TReq

	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			e.errorHandler.HandleError(w, err)

			return
		}
	}

	if ext, ok := any(&req).(Extracter); ok {
		if err := ext.Extract(r); err != nil {
			e.errorHandler.HandleError(w, err)

			return
		}
	}

	if v, ok := any(&req).(Validator); ok {
		if err := v.Validate(ctx); err != nil {
			e.errorHandler.HandleError(w, err)

			return
		}
	}

	res, err := e.handler.Handle(ctx, &req)
	if err != nil {
		e.errorHandler.HandleError(w, err)

		return
	}

	if rw, ok := any(res).(RawWriter); ok {
		if err := rw.Write(w); err != nil {
			e.errorHandler.HandleError(w, err)
		}

		return
	}

	status := http.StatusOK
	if sc, ok := any(res).(StatusCoder); ok {
		status = sc.StatusCode()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		_ = fmt.Errorf("api: failed to encode response: %w", err)
	}
}
