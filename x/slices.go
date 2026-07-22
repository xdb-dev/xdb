package x

// Map transforms each item using fn, preserving order.
func Map[T, R any](items []T, fn func(T) R) []R {
	mapped := make([]R, 0, len(items))

	for _, item := range items {
		mapped = append(mapped, fn(item))
	}

	return mapped
}

// Filter returns the items for which fn reports true, preserving order.
func Filter[T any](items []T, fn func(T) bool) []T {
	filtered := make([]T, 0, len(items))

	for _, item := range items {
		if fn(item) {
			filtered = append(filtered, item)
		}
	}

	return filtered
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

// Diff returns the items in a whose key (per fn) does not appear in b.
func Diff[T any](a, b []T, fn func(T) string) []T {
	bmap := Index(b, fn)

	diff := make([]T, 0)

	for _, item := range a {
		if _, ok := bmap[fn(item)]; !ok {
			diff = append(diff, item)
		}
	}

	return diff
}

// Join appends lists into a single list.
func Join[T any](lists ...[]T) []T {
	var result []T

	for _, list := range lists {
		result = append(result, list...)
	}

	return result
}
