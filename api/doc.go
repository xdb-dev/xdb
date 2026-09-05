// Package api provides transport-agnostic services for XDB operations.
//
// Each resource has its own service type with plain method signatures:
//
//	records := api.NewRecordService(store)
//	res, err := records.Create(ctx, &api.CreateRecordRequest{...})
//
// Services can be wired to JSON-RPC via [rpc.RegisterHandler]:
//
//	r := rpc.NewRouter()
//	rpc.RegisterHandler(r, "records.create", records.Create)
//
// A request type can implement [Validator] for input validation and
// [Extracter] for HTTP-specific data extraction. A response type can
// implement [StatusCoder] and [RawWriter] to customize the HTTP response.
package api
