package x

import "github.com/xdb-dev/xdb/core"

// URIs extracts URIs from a list of tuples and/or records.
func URIs[T interface {
	URI() *core.URI
}](items ...T) []*core.URI {
	uris := make([]*core.URI, len(items))
	for i, item := range items {
		uris[i] = item.URI()
	}
	return uris
}
