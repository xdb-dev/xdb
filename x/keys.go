package x

import "github.com/xdb-dev/xdb/core"

// Keys extracts keys from a list of tuples and/or records.
func Keys[T interface {
	Key() *core.Key
}](items ...T) []*core.Key {
	keys := make([]*core.Key, len(items))
	for i, item := range items {
		keys[i] = item.Key()
	}
	return keys
}
