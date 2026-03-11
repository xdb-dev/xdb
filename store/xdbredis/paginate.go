package xdbredis

import "github.com/xdb-dev/xdb/store"

// paginate applies offset/limit to a sorted slice and returns a [store.Page].
func paginate[T any](items []T, q *store.ListQuery) *store.Page[T] {
	total := len(items)

	offset := 0
	limit := total
	if q != nil {
		offset = q.Offset
		if q.Limit > 0 {
			limit = q.Limit
		}
	}

	offset = min(offset, total)
	end := min(offset+limit, total)

	var nextOffset int
	if end < total {
		nextOffset = end
	}

	return &store.Page[T]{
		Items:      items[offset:end],
		Total:      total,
		NextOffset: nextOffset,
	}
}
