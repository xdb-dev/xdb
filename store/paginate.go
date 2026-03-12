package store

// Paginate applies offset/limit from a [ListQuery] to a sorted slice
// and returns a [Page].
func Paginate[T any](items []T, q *ListQuery) *Page[T] {
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

	return &Page[T]{
		Items:      items[offset:end],
		Total:      total,
		NextOffset: nextOffset,
	}
}
