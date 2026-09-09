package x

// Map transforms each item using fn, preserving order.
func Map[T, R any](items []T, fn func(T) R) []R {
	mapped := make([]R, 0, len(items))

	for _, item := range items {
		mapped = append(mapped, fn(item))
	}

	return mapped
}

// Index maps each item by the key returned by fn. Later items win on
// key collision.
func Index[T any](items []T, fn func(T) string) map[string]T {
	index := make(map[string]T, len(items))

	for _, item := range items {
		index[fn(item)] = item
	}

	return index
}
