// Package rpc provides a JSON-RPC 2.0 server for XDB.
//
// [Router] maps method names to handlers and implements [http.Handler]
// for serving JSON-RPC 2.0 requests over HTTP.
//
// Use [RegisterHandler] to register typed service methods, then serve
// the router directly:
//
//	r := rpc.NewRouter()
//	rpc.RegisterHandler(r, "records.create", records.Create)
//	http.ListenAndServe(":8080", r)
package rpc
