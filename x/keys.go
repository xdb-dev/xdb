package x

import "github.com/xdb-dev/xdb/types"

// Keys extracts keys from a list of tuples and/or records.
func Keys[T interface {
	Key() *types.Key
}](items ...T) []*types.Key {
	keys := make([]*types.Key, len(items))
	for i, item := range items {
		keys[i] = item.Key()
	}
	return keys
}
