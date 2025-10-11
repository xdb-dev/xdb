package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gojekfarm/xtools/xapi"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
)

type Server struct {
	http  *http.Server
	store *xdbsqlite.KVStore
}

func NewServer(cfg *Config) *Server {
	db, err := sql.Open("sqlite3", "file:xdb.db")
	if err != nil {
		panic(err)
	}

	store := xdbsqlite.NewKVStore(db)

	err = store.Migrate(context.Background())
	if err != nil {
		panic(err)
	}

	mux := createRoutes(store)
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

func createRoutes(store *xdbsqlite.KVStore) *http.ServeMux {
	mux := http.NewServeMux()

	middlewares := xapi.MiddlewareStack{
		xapi.MiddlewareFunc(LoggingMiddleware),
	}

	getTuples := xapi.NewEndpoint(
		xapi.EndpointFunc[api.GetTuplesRequest, api.GetTuplesResponse](api.GetTuples(store)),
		xapi.WithMiddleware(middlewares...),
	)
	putTuples := xapi.NewEndpoint(
		xapi.EndpointFunc[api.PutTuplesRequest, api.PutTuplesResponse](api.PutTuples(store)),
		xapi.WithMiddleware(middlewares...),
	)

	mux.Handle("POST /v1/tuples:get", getTuples.Handler())
	mux.Handle("PUT /v1/tuples", putTuples.Handler())

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
