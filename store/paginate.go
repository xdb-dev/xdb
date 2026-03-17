package store

const (
	// DefaultLimit is the default number of items per page.
	DefaultLimit = 20
	// MaxLimit is the maximum number of items per page.
	MaxLimit = 1000
)

// Paginate applies offset/limit from a [ListQuery] to a sorted slice
// and returns a [Page]. Defaults to [DefaultLimit], capped at [MaxLimit].
func Paginate[T any](items []T, q *ListQuery) *Page[T] {
	total := len(items)

	offset := 0
	limit := DefaultLimit

	if q != nil {
		offset = q.Offset
		limit = q.Limit
		if limit <= 0 {
			limit = DefaultLimit
		}
		if limit > MaxLimit {
			limit = MaxLimit
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
