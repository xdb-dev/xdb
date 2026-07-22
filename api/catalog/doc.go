// Package catalog is the single source of truth for XDB's JSON-RPC method
// and type metadata. [Methods] backs daemon registration
// (cmd/xdb/daemon) and [Types] backs the introspect.type/introspect.types
// API ([api.IntrospectService]), so the daemon's live behavior and its own
// self-description can never drift apart.
package catalog
