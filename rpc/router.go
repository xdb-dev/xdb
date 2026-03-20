package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/xdb-dev/xdb/core"
)

// ParamMeta describes a single request or response parameter.
type ParamMeta struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
}

// MethodMeta describes an RPC method's interface.
type MethodMeta struct {
	Parameters  map[string]ParamMeta `json:"parameters,omitempty"`
	Response    map[string]ParamMeta `json:"response,omitempty"`
	Description string               `json:"description"`
	Mutating    bool                 `json:"mutating,omitempty"`
}

// methodEntry is the internal representation of a registered method.
type methodEntry struct {
	fn     func(ctx context.Context, params json.RawMessage, send func(string, json.RawMessage)) (json.RawMessage, error)
	meta   MethodMeta
	stream bool
}

// Router maps JSON-RPC method names to handlers and serves
// JSON-RPC 2.0 requests over HTTP.
type Router struct {
	methods map[string]methodEntry
}

// NewRouter creates an empty [Router].
func NewRouter() *Router {
	return &Router{
		methods: make(map[string]methodEntry),
	}
}

// Methods returns the names of all registered methods.
func (r *Router) Methods() []string {
	names := make([]string, 0, len(r.methods))
	for name := range r.methods {
		names = append(names, name)
	}

	return names
}

// Meta returns the metadata for the named method.
func (r *Router) Meta(method string) (MethodMeta, bool) {
	entry, ok := r.methods[method]
	if !ok {
		return MethodMeta{}, false
	}

	return entry.meta, true
}

// ServeHTTP implements [http.Handler] for JSON-RPC 2.0 over HTTP.
// Normal methods return a single JSON response.
// Streaming methods return SSE (text/event-stream).
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, NewErrorResponse(
			"",
			InvalidRequest("method must be POST"),
		))

		return
	}

	var rpcReq Request
	if err := json.NewDecoder(req.Body).Decode(&rpcReq); err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse(
			"",
			ParseError(err.Error()),
		))

		return
	}

	entry, ok := r.methods[rpcReq.Method]
	if !ok {
		writeJSON(w, http.StatusOK, NewErrorResponse(
			rpcReq.ID,
			MethodNotFound(rpcReq.Method),
		))

		return
	}

	if entry.stream {
		r.serveStream(w, req, rpcReq, entry)

		return
	}

	result, err := entry.fn(req.Context(), rpcReq.Params, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, NewErrorResponse(
			rpcReq.ID,
			InternalError(err.Error()),
		))

		return
	}

	writeJSON(w, http.StatusOK, NewResponse(rpcReq.ID, result))
}

func (r *Router) serveStream(
	w http.ResponseWriter,
	httpReq *http.Request,
	rpcReq Request,
	entry methodEntry,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusOK, NewErrorResponse(
			rpcReq.ID,
			InternalError("streaming not supported"),
		))

		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	send := func(event string, data json.RawMessage) {
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	ctx := httpReq.Context()

	_, fnErr := entry.fn(ctx, rpcReq.Params, send)
	if fnErr != nil {
		errPayload, marshalErr := json.Marshal(MapError(fnErr))
		if marshalErr != nil {
			errPayload = []byte(`{"code":-32603,"message":"internal error"}`)
		}

		_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", errPayload)
		flusher.Flush()

		return
	}

	_, _ = fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// RegisterHandler registers a typed handler function for the given method.
// It handles JSON unmarshaling of params into the request type and delegates
// to the handler function.
func RegisterHandler[Req, Res any](
	r *Router,
	method string,
	h func(ctx context.Context, req *Req) (*Res, error),
) {
	RegisterHandlerWithMeta(r, method, h, MethodMeta{})
}

// RegisterHandlerWithMeta registers a typed handler with method metadata.
func RegisterHandlerWithMeta[Req, Res any](
	r *Router,
	method string,
	h func(ctx context.Context, req *Req) (*Res, error),
	meta MethodMeta,
) {
	r.methods[method] = methodEntry{
		fn: func(ctx context.Context, params json.RawMessage, _ func(string, json.RawMessage)) (json.RawMessage, error) {
			var req Req

			if len(params) > 0 {
				if err := json.Unmarshal(params, &req); err != nil {
					return nil, InvalidParams(err.Error())
				}
			}

			res, err := h(ctx, &req)
			if err != nil {
				return nil, err
			}

			data, err := json.Marshal(res)
			if err != nil {
				return nil, fmt.Errorf("rpc: marshal response: %w", err)
			}

			return data, nil
		},
		meta: meta,
	}
}

// RegisterStream registers a streaming handler for the given method.
// Streaming handlers call send to push events to the client and return
// when the stream is complete.
func RegisterStream[Req any](
	r *Router,
	method string,
	h func(ctx context.Context, req *Req, send func(string, json.RawMessage)) error,
) {
	RegisterStreamWithMeta(r, method, h, MethodMeta{})
}

// RegisterStreamWithMeta registers a streaming handler with method metadata.
func RegisterStreamWithMeta[Req any](
	r *Router,
	method string,
	h func(ctx context.Context, req *Req, send func(string, json.RawMessage)) error,
	meta MethodMeta,
) {
	r.methods[method] = methodEntry{
		fn: func(ctx context.Context, params json.RawMessage, send func(string, json.RawMessage)) (json.RawMessage, error) {
			var req Req

			if len(params) > 0 {
				if err := json.Unmarshal(params, &req); err != nil {
					return nil, InvalidParams(err.Error())
				}
			}

			return nil, h(ctx, &req, send)
		},
		meta:   meta,
		stream: true,
	}
}

// MapError converts a Go error to a JSON-RPC [Error].
// It maps well-known store errors to XDB-specific error codes
// and falls back to [InternalError] for unknown errors.
func MapError(err error) *Error {
	if rpcErr, ok := err.(*Error); ok {
		return rpcErr
	}

	msg := err.Error()

	switch {
	case errors.Is(err, core.ErrNotFound):
		return NotFound(msg)
	case errors.Is(err, core.ErrAlreadyExists):
		return AlreadyExists(msg)
	case errors.Is(err, core.ErrSchemaViolation):
		return SchemaViolation(msg)
	default:
		return InternalError(msg)
	}
}

func writeJSON(w http.ResponseWriter, status int, v *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
