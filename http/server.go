package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type Config struct {
	Addr string
}

type Server struct {
	mux  *http.ServeMux
	http *http.Server
}

func NewServer(cfg *Config) *Server {
	http := &http.Server{
		Addr: cfg.Addr,
	}

	return &Server{http: http}
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func RegisterEndpoint[TReq, TRes any](s *Server, pattern string, endpoint *Endpoint[TReq, TRes]) {
	s.mux.Handle(pattern, endpoint.Handler())
}

type EndpointHandler[TReq, TRes any] interface {
	Handle(ctx context.Context, req *TReq) (*TRes, error)
}

type EndpointFunc[TReq, TRes any] func(ctx context.Context, req *TReq) (*TRes, error)

func (e EndpointFunc[TReq, TRes]) Handle(ctx context.Context, req *TReq) (*TRes, error) {
	return e(ctx, req)
}

type MiddlewareHandler interface {
	Middleware(next http.Handler) http.Handler
}

type MiddlewareFunc func(next http.Handler) http.Handler

func (m MiddlewareFunc) Middleware(next http.Handler) http.Handler {
	return m(next)
}

type MiddlewareStack []MiddlewareHandler

func (m MiddlewareStack) Middleware(next http.Handler) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		next = m[i].Middleware(next)
	}
	return next
}

type ErrorHandler interface {
	HandleError(w http.ResponseWriter, err error)
}

type ErrorFunc func(w http.ResponseWriter, err error)

func (e ErrorFunc) HandleError(w http.ResponseWriter, err error) {
	e(w, err)
}

type EndpointOption[TReq, TRes any] interface {
	apply(e *Endpoint[TReq, TRes])
}

type Endpoint[TReq, TRes any] struct {
	handler      EndpointHandler[TReq, TRes]
	middleware   MiddlewareStack
	errorHandler ErrorHandler
}

func NewEndpoint[TReq, TRes any](handler EndpointHandler[TReq, TRes], options ...EndpointOption[TReq, TRes]) *Endpoint[TReq, TRes] {
	e := &Endpoint[TReq, TRes]{
		handler: handler,
	}

	for _, option := range options {
		option.apply(e)
	}

	if e.errorHandler == nil {
		e.errorHandler = ErrorFunc(DefaultErrorHandler)
	}

	return e
}

func (e *Endpoint[TReq, TRes]) Handler() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req TReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			e.errorHandler.HandleError(w, err)
			return
		}

		res, err := e.handler.Handle(r.Context(), &req)
		if err != nil {
			e.errorHandler.HandleError(w, err)
			return
		}

		if err := json.NewEncoder(w).Encode(res); err != nil {
			e.errorHandler.HandleError(w, err)
			return
		}
	})

	if len(e.middleware) > 0 {
		return e.middleware.Middleware(h)
	}

	return h
}

func DefaultErrorHandler(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, &json.SyntaxError{}):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, &json.UnmarshalTypeError{}):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, &json.InvalidUnmarshalError{}):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type endpointOptionFunc[TReq, TRes any] func(e *Endpoint[TReq, TRes])

func (f endpointOptionFunc[TReq, TRes]) apply(e *Endpoint[TReq, TRes]) {
	f(e)
}

func WithMiddleware[TReq, TRes any](middlewares ...MiddlewareHandler) EndpointOption[TReq, TRes] {
	return endpointOptionFunc[TReq, TRes](func(e *Endpoint[TReq, TRes]) {
		e.middleware = append(e.middleware, middlewares...)
	})
}

func WithErrorHandler[TReq, TRes any](errorHandler ErrorHandler) EndpointOption[TReq, TRes] {
	return endpointOptionFunc[TReq, TRes](func(e *Endpoint[TReq, TRes]) {
		e.errorHandler = errorHandler
	})
}
