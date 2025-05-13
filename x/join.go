package x

// Join appends lists into a single list.
func Join[T any](lists ...[]T) []T {
	var result []T
	for _, list := range lists {
		result = append(result, list...)
	}
	return result
}
