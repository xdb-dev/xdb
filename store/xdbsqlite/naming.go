package xdbsqlite

import (
	"context"
	"sort"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// kvTableName returns the quoted table name for a KV-strategy schema.
// Format: "kv:<ns>/<schema>".
func kvTableName(uri *core.URI) string {
	return `"kv:` + uri.NS() + `/` + uri.Schema() + `"`
}

// columnTableName returns the quoted table name for a column-strategy schema.
// Format: "t:<ns>/<schema>".
func columnTableName(uri *core.URI) string {
	return `"t:` + uri.NS() + `/` + uri.Schema() + `"`
}

// columnIndexName returns the quoted name of the index backing an indexed
// or unique field on a column table. Format: "ix:t:<ns>/<schema>:<field>".
// The name is unique per database, mirroring [kvIndexName].
func columnIndexName(uri *core.URI, field string) string {
	return `"ix:t:` + uri.NS() + `/` + uri.Schema() + `:` + field + `"`
}

// parseKVTable parses an unquoted "kv:<ns>/<schema>" table name into
// its schema URI. Returns false for any other name.
func parseKVTable(name string) (*core.URI, bool) {
	rest, ok := strings.CutPrefix(name, "kv:")
	if !ok {
		return nil, false
	}
	ns, schemaName, ok := strings.Cut(rest, "/")
	if !ok {
		return nil, false
	}
	return core.MustNewURI(ns, schemaName), true
}

// scanTarget is one schema to scan: its URI and governing def (nil for
// schema-less KV data). The def picks the engine via [engineFor].
type scanTarget struct {
	uri *core.URI
	def *schema.Def
}

// scanTargets enumerates the schemas to scan under scope, ordered by
// (ns, schema): every registered definition, plus KV tables that carry
// records without a definition (schema-less data). Column tables have
// no discovery path — a column table is unreadable without its def.
func scanTargets(
	ctx context.Context,
	q *xsql.Queries,
	scope *core.URI,
) ([]scanTarget, error) {
	var targets []scanTarget
	seen := make(map[string]bool)

	for def, err := range scanSchemas(ctx, q, scope) {
		if err != nil {
			return nil, err
		}
		// scanSchemas filters by namespace only; narrow to the schema
		// when the scope names one, so a schema-scoped scan does not
		// leak sibling schemas.
		if scope != nil && scope.Schema() != "" && def.URI.Schema() != scope.Schema() {
			continue
		}
		targets = append(targets, scanTarget{uri: def.URI, def: def})
		seen[def.URI.SchemaURI().Path()] = true
	}

	names, err := q.ListTables(ctx, xsql.ListTablesParams{
		Pattern: "kv:" + kvScopeSuffix(scope),
	})
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		uri, ok := parseKVTable(name)
		if !ok || seen[uri.SchemaURI().Path()] {
			continue
		}
		targets = append(targets, scanTarget{uri: uri, def: nil})
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].uri.NS() != targets[j].uri.NS() {
			return targets[i].uri.NS() < targets[j].uri.NS()
		}
		return targets[i].uri.Schema() < targets[j].uri.Schema()
	})

	return targets, nil
}

// kvScopeSuffix builds the GLOB suffix for KV table discovery under a
// namespace ("ns/*") or a single schema ("ns/schema").
func kvScopeSuffix(scope *core.URI) string {
	if scope != nil && scope.Schema() != "" {
		return scope.NS() + "/" + scope.Schema()
	}
	if scope != nil && scope.NS() != "" {
		return scope.NS() + "/*"
	}
	return "*/*"
}
