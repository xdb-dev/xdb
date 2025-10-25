package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
)

type Server struct {
	http  *http.Server
	store *xdbsqlite.Store
}

func NewServer(cfg *Config) *Server {
	store, err := xdbsqlite.NewStore(*cfg.Store.SQLite, xdbmemory.New())
	if err != nil {
		panic(err)
	}

	mux := createRoutesHandler(store)
	http := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}

	return &Server{
		http:  http,
		store: store,
	}
}

func (s *Server) Run(ctx context.Context) error {
	s.start()

	<-ctx.Done()

	return s.http.Shutdown(ctx)
}

func createRoutesHandler(store *xdbsqlite.Store) *http.ServeMux {
	mux := http.NewServeMux()

	// middlewares := xapi.MiddlewareStack{
	// 	xapi.MiddlewareFunc(LoggingMiddleware),
	// }

	// tupleAPI := api.NewTupleAPI(store)

	// getTuples := xapi.NewEndpoint(
	// 	xapi.EndpointFunc[api.GetTuplesRequest, api.GetTuplesResponse](tupleAPI.GetTuples()),
	// 	xapi.WithMiddleware(middlewares...),
	// )
	// putTuples := xapi.NewEndpoint(
	// 	xapi.EndpointFunc[api.PutTuplesRequest, api.PutTuplesResponse](tupleAPI.PutTuples()),
	// 	xapi.WithMiddleware(middlewares...),
	// )
	// deleteTuples := xapi.NewEndpoint(
	// 	xapi.EndpointFunc[api.DeleteTuplesRequest, api.DeleteTuplesResponse](tupleAPI.DeleteTuples()),
	// 	xapi.WithMiddleware(middlewares...),
	// )

	// mux.Handle("POST /v1/tuples:get", getTuples.Handler())
	// mux.Handle("PUT /v1/tuples", putTuples.Handler())
	// mux.Handle("DELETE /v1/tuples", deleteTuples.Handler())

	return mux
}

func (s *Server) start() {
	slog.Info("[HTTP] Starting server", "addr", s.http.Addr)

	err := s.http.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		path := fmt.Sprintf("[HTTP] %s %s", r.Method, r.URL.Path)

		slog.Info(path, "duration", duration)
	})
}
