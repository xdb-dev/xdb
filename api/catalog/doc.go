// Package catalog is the single source of truth for the JSON-RPC method
// and type metadata of XDB. [Methods] backs daemon registration
// (cmd/xdb/daemon). [Types] backs the introspect.type and
// introspect.types methods ([api.IntrospectService]). As a result, the
// live behavior of the daemon and its self-description cannot drift
// apart.
package catalog
