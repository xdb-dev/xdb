package x

import (
	"github.com/xdb-dev/xdb/core"
)

// GroupTuples groups tuples by their record path (ns/schema/id).
func GroupTuples(tuples []*core.Tuple) map[string][]*core.Tuple {
	return GroupBy(tuples, func(t *core.Tuple) string {
		return t.Path().RecordPath()
	})
}

// GroupAttrs groups attr-level URIs by their record path (ns/schema/id),
// collecting the attribute names.
func GroupAttrs(uris []*core.URI) map[string][]string {
	grouped := make(map[string][]string)

	for _, uri := range uris {
		key := uri.RecordPath()
		grouped[key] = append(grouped[key], uri.Attr())
	}

	return grouped
}

// GroupBy groups items by the key returned by fn, preserving input
// order within each group.
func GroupBy[T any](items []T, fn func(T) string) map[string][]T {
	grouped := make(map[string][]T)

	for _, item := range items {
		key := fn(item)
		grouped[key] = append(grouped[key], item)
	}

	return grouped
}
