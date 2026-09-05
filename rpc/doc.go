// Package rpc provides a JSON-RPC 2.0 server for XDB.
//
// [Router] maps method names to handlers and implements [http.Handler]
// for serving JSON-RPC 2.0 requests over HTTP.
//
// Register typed service methods with [RegisterHandler]. Then serve
// the router directly:
//
//	r := rpc.NewRouter()
//	rpc.RegisterHandler(r, "records.create", records.Create)
//	http.ListenAndServe(":8080", r)
package rpc
