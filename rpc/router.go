package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// methodEntry is the internal representation of a registered method.
type methodEntry struct {
	fn     func(ctx context.Context, params json.RawMessage, send func(string, json.RawMessage)) (json.RawMessage, error)
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
		stream: true,
	}
}

// MapError converts a Go error to a JSON-RPC [Error].
func MapError(err error) *Error {
	if rpcErr, ok := err.(*Error); ok {
		return rpcErr
	}

	return InternalError(err.Error())
}

func writeJSON(w http.ResponseWriter, status int, v *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
