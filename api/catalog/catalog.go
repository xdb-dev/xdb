package catalog

import (
	"fmt"
	"maps"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/rpc"
)

// Methods returns the metadata for every JSON-RPC method the daemon
// registers, keyed by method name. It is the single source of truth for
// daemon registration ([rpc.RegisterHandlerWithMeta] /
// [rpc.RegisterStreamWithMeta]) and offline CLI describe.
func Methods() map[string]rpc.MethodMeta {
	methods := make(map[string]rpc.MethodMeta)

	maps.Copy(methods, recordMethods())
	maps.Copy(methods, schemaMethods())
	maps.Copy(methods, namespaceMethods())
	maps.Copy(methods, batchMethods())
	maps.Copy(methods, watchMethods())
	maps.Copy(methods, systemMethods())
	maps.Copy(methods, introspectMethods())

	return methods
}

// recordMethods returns metadata for the records.* methods.
func recordMethods() map[string]rpc.MethodMeta {
	return map[string]rpc.MethodMeta{
		"records.create": {
			Description: "Create a new record. Identical re-create is an idempotent success; " +
				"creating over an existing resource with different data fails with CONFLICT.",
			Mutating: true,
			Parameters: map[string]rpc.ParamMeta{
				"uri":     {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
				"data":    {Description: "Record data as JSON object", Type: "object"},
				"dry_run": {Description: "Validate without writing; response carries dry_run{valid,would}", Type: "boolean"},
			},
			Response: map[string]rpc.ParamMeta{
				"data":    {Description: "The created or existing record data", Type: "object"},
				"dry_run": {Description: "Present only on dry-run: {valid, would}", Type: "object"},
			},
		},
		"records.get": {
			Description: "Retrieve a record by URI.",
			Parameters: map[string]rpc.ParamMeta{
				"uri":    {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
				"fields": {Description: "Field projection list", Type: "array"},
			},
			Response: map[string]rpc.ParamMeta{
				"data": {Description: "The record data", Type: "object"},
			},
		},
		"records.list": {
			Description: "List records matching a query.",
			Parameters: map[string]rpc.ParamMeta{
				"uri":    {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
				"filter": {Description: "CEL filter expression", Type: "string"},
				"fields": {Description: "Field projection list", Type: "array"},
				"limit":  {Description: "Max items per page", Type: "integer"},
				"offset": {Description: "Page offset", Type: "integer"},
			},
			Response: map[string]rpc.ParamMeta{
				"items":       {Description: "Matching records", Type: "array"},
				"next_offset": {Description: "Offset for next page", Type: "integer"},
				"total":       {Description: "Total matching records", Type: "integer"},
			},
		},
		"records.update": {
			Description: "Update an existing record (patch semantics). Only supplied fields change.",
			Mutating:    true,
			Parameters: map[string]rpc.ParamMeta{
				"uri":     {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
				"data":    {Description: "Patch data as JSON object", Type: "object", Required: true},
				"dry_run": {Description: "Validate without writing; response carries dry_run{valid,would}", Type: "boolean"},
			},
			Response: map[string]rpc.ParamMeta{
				"data":    {Description: "The updated record data", Type: "object"},
				"dry_run": {Description: "Present only on dry-run: {valid, would}", Type: "object"},
			},
		},
		"records.upsert": {
			Description: "Create or replace a record (full replace). Sets complete state.",
			Mutating:    true,
			Parameters: map[string]rpc.ParamMeta{
				"uri":     {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
				"data":    {Description: "Record data as JSON object", Type: "object"},
				"dry_run": {Description: "Validate without writing; response carries dry_run{valid,would}", Type: "boolean"},
			},
			Response: map[string]rpc.ParamMeta{
				"data":    {Description: "The upserted record data", Type: "object"},
				"dry_run": {Description: "Present only on dry-run: {valid, would}", Type: "object"},
			},
		},
		"records.delete": {
			Description: "Delete a record. Idempotent: succeeds even if not found.",
			Mutating:    true,
			Parameters: map[string]rpc.ParamMeta{
				"uri":     {Description: "Record URI (xdb://ns/schema/id)", Type: "string", Required: true},
				"dry_run": {Description: "Validate without writing; response carries dry_run{valid,would}", Type: "boolean"},
			},
		},
	}
}

// schemaMethods returns metadata for the schemas.* methods.
func schemaMethods() map[string]rpc.MethodMeta {
	return map[string]rpc.MethodMeta{
		"schemas.create": {
			Description: "Create a new schema definition. Identical re-create is an idempotent success; " +
				"creating over an existing resource with different data fails with CONFLICT.",
			Mutating: true,
			Parameters: map[string]rpc.ParamMeta{
				"uri": {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
				"data": {
					Description: "Schema definition as JSON object (fields may declare items " +
						"for ARRAY<JSON> members)",
					Type: "object",
				},
				"dry_run": {Description: "Validate without writing; response carries dry_run{valid,would}", Type: "boolean"},
			},
			Response: map[string]rpc.ParamMeta{
				"data":    {Description: "The created or existing schema definition", Type: "object"},
				"dry_run": {Description: "Present only on dry-run: {valid, would}", Type: "object"},
			},
		},
		"schemas.get": {
			Description: "Retrieve a schema definition by URI.",
			Parameters: map[string]rpc.ParamMeta{
				"uri": {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
			},
			Response: map[string]rpc.ParamMeta{
				"data": {Description: "The schema definition", Type: "object"},
			},
		},
		"schemas.list": {
			Description: "List schemas in a namespace.",
			Parameters: map[string]rpc.ParamMeta{
				"uri":    {Description: "Namespace URI (xdb://ns)", Type: "string"},
				"limit":  {Description: "Max items per page", Type: "integer"},
				"offset": {Description: "Page offset", Type: "integer"},
			},
			Response: map[string]rpc.ParamMeta{
				"items":       {Description: "Schema definitions", Type: "array"},
				"next_offset": {Description: "Offset for next page", Type: "integer"},
				"total":       {Description: "Total schemas", Type: "integer"},
			},
		},
		"schemas.update": {
			Description: "Update a schema definition (patch semantics). Fields (including items) " +
				"are added or replaced; removal is not supported.",
			Mutating: true,
			Parameters: map[string]rpc.ParamMeta{
				"uri": {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
				"data": {
					Description: "Patch data as JSON object (fields to add or replace, optional items " +
						"for ARRAY<JSON> members, optional revision for optimistic-concurrency CAS — " +
						"omit or 0 for unconditional update)",
					Type:     "object",
					Required: true,
				},
			},
			Response: map[string]rpc.ParamMeta{
				"data": {Description: "The updated schema definition", Type: "object"},
			},
		},
		"schemas.delete": {
			Description: "Delete a schema. Idempotent: succeeds even if not found.",
			Mutating:    true,
			Parameters: map[string]rpc.ParamMeta{
				"uri":     {Description: "Schema URI (xdb://ns/schema)", Type: "string", Required: true},
				"cascade": {Description: "Delete all records in the schema", Type: "boolean"},
				"dry_run": {Description: "Validate without writing; response carries dry_run{valid,would}", Type: "boolean"},
			},
		},
	}
}

// namespaceMethods returns metadata for the namespaces.* methods.
func namespaceMethods() map[string]rpc.MethodMeta {
	return map[string]rpc.MethodMeta{
		"namespaces.get": {
			Description: "Retrieve namespace metadata by URI.",
			Parameters: map[string]rpc.ParamMeta{
				"uri": {Description: "Namespace URI (xdb://ns)", Type: "string", Required: true},
			},
			Response: map[string]rpc.ParamMeta{
				"data": {Description: "The namespace metadata", Type: "object"},
			},
		},
		"namespaces.list": {
			Description: "List all known namespaces.",
			Parameters: map[string]rpc.ParamMeta{
				"limit":  {Description: "Max items per page", Type: "integer"},
				"offset": {Description: "Page offset", Type: "integer"},
			},
			Response: map[string]rpc.ParamMeta{
				"items":       {Description: "Namespaces", Type: "array"},
				"next_offset": {Description: "Offset for next page", Type: "integer"},
				"total":       {Description: "Total namespaces", Type: "integer"},
			},
		},
	}
}

// batchMethods returns metadata for the batch.* methods.
func batchMethods() map[string]rpc.MethodMeta {
	return map[string]rpc.MethodMeta{
		"batch.execute": {
			Description: "Run multiple operations in a single atomic transaction on transactional " +
				"backends (sqlite, memory); other backends require non_atomic:true for sequential " +
				"best-effort execution.",
			Mutating: true,
			Parameters: map[string]rpc.ParamMeta{
				"operations": {
					Description: "Array of operation objects {op, uri, data}. op is one of: " +
						"records.create, records.update, records.upsert, records.delete, " +
						"schemas.create, schemas.update, schemas.delete.",
					Type:     "array",
					Required: true,
				},
				"dry_run":    {Description: "Validate every operation without writing; per-op dry_run results", Type: "boolean"},
				"non_atomic": {Description: "Allow sequential best-effort execution on non-transactional backends", Type: "boolean"},
			},
			Response: map[string]rpc.ParamMeta{
				"total":       {Description: "Total operations", Type: "integer"},
				"succeeded":   {Description: "Successful operations", Type: "integer"},
				"failed":      {Description: "Failed operations", Type: "integer"},
				"rolled_back": {Description: "True when a failure rolled back the whole batch", Type: "boolean"},
				"results": {
					Description: "Per-operation results {index, uri, status: ok|error|skipped, error?, dry_run?}",
					Type:        "array",
				},
			},
		},
	}
}

// watchMethods returns metadata for the watch method.
func watchMethods() map[string]rpc.MethodMeta {
	return map[string]rpc.MethodMeta{
		"watch": {
			Description: "Stream changes matching a URI pattern.",
			Parameters: map[string]rpc.ParamMeta{
				"uri": {Description: "URI pattern to watch", Type: "string", Required: true},
			},
		},
	}
}

// systemMethods returns metadata for the system.* methods.
func systemMethods() map[string]rpc.MethodMeta {
	return map[string]rpc.MethodMeta{
		"system.health": {
			Description: "Report system health status.",
			Response: map[string]rpc.ParamMeta{
				"status": {Description: "Health status", Type: "string"},
			},
		},
		"system.version": {
			Description: "Report system version.",
			Response: map[string]rpc.ParamMeta{
				"version": {Description: "Daemon version string", Type: "string"},
			},
		},
	}
}

// introspectMethods returns metadata for the introspect.* methods.
func introspectMethods() map[string]rpc.MethodMeta {
	return map[string]rpc.MethodMeta{
		"introspect.method": {
			Description: "Describe a single API method.",
			Parameters: map[string]rpc.ParamMeta{
				"method": {Description: "Method name (e.g. records.create)", Type: "string", Required: true},
			},
			Response: map[string]rpc.ParamMeta{
				"method":      {Description: "The method name", Type: "string"},
				"description": {Description: "What the method does", Type: "string"},
				"parameters":  {Description: "Parameter metadata by name", Type: "object"},
				"response":    {Description: "Response field metadata by name", Type: "object"},
				"mutating":    {Description: "Whether the method mutates state", Type: "boolean"},
			},
		},
		"introspect.type": {
			Description: "Describe a single API type.",
			Parameters: map[string]rpc.ParamMeta{
				"type": {Description: "Type name (e.g. Record)", Type: "string", Required: true},
			},
			Response: map[string]rpc.ParamMeta{
				"type":        {Description: "The type name", Type: "string"},
				"description": {Description: "What the type represents", Type: "string"},
			},
		},
		"introspect.methods": {
			Description: "List all registered API methods.",
			Response: map[string]rpc.ParamMeta{
				"methods": {Description: "Method name and summary pairs", Type: "array"},
			},
		},
		"introspect.types": {
			Description: "List all available API types.",
			Response: map[string]rpc.ParamMeta{
				"types": {Description: "Type name and description pairs", Type: "array"},
			},
		},
	}
}

// Method returns the metadata for the named method.
func Method(name string) (rpc.MethodMeta, bool) {
	meta, ok := Methods()[name]
	return meta, ok
}

// Types returns the description of every API-facing type, keyed by name.
//
// BatchOperation, Event, and DryRunResult are added by later phases
// (batch, watch, and dry-run implementations, respectively).
func Types() map[string]string {
	return map[string]string{
		"Record":    "A collection of tuples sharing the same ID within a schema.",
		"Schema":    "A definition of attributes and their types for a schema.",
		"Namespace": "A logical grouping of schemas (e.g., com.example).",
		"Tuple":     "A single attribute-value pair within a record.",
		"Value":     fmt.Sprintf("A typed value (%s).", strings.Join(core.ValueTypeNames(), ", ")),
		"URI":       "A reference to XDB data: xdb://NS/SCHEMA/ID#ATTR",
		"Filter":    "A CEL filter expression used to match records in records.list.",
		"Mode":      "Schema validation mode: flexible, strict, or dynamic.",
	}
}
