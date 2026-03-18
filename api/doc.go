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
// Request types may optionally implement [Validator] for input validation
// and [Extracter] for HTTP-specific data extraction. Response types may
// optionally implement [StatusCoder] and [RawWriter] for HTTP customization.
package api
