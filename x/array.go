// Package x provides utility and helper functions used throughout XDB.
package x

import "github.com/xdb-dev/xdb/core"

// GroupTuples groups a list of tuples by their ID.
func GroupTuples(tuples ...*core.Tuple) map[string][]*core.Tuple {
	grouped := make(map[string][]*core.Tuple)

	for _, tuple := range tuples {
		id := tuple.ID().String()

		if _, ok := grouped[id]; !ok {
			grouped[id] = make([]*core.Tuple, 0)
		}

		grouped[id] = append(grouped[id], tuple)
	}

	return grouped
}

// GroupAttrs groups a list of attributes by their ID.
func GroupAttrs(uris ...*core.URI) map[string][]string {
	grouped := make(map[string][]string)

	for _, uri := range uris {
		id := uri.ID().String()

		if _, ok := grouped[id]; !ok {
			grouped[id] = make([]string, 0)
		}

		grouped[id] = append(grouped[id], uri.Attr().String())
	}

	return grouped
}

// GroupBy groups a list using a function.
func GroupBy[T any](items []T, fn func(T) string) map[string][]T {
	grouped := make(map[string][]T)

	for _, item := range items {
		grouped[fn(item)] = append(grouped[fn(item)], item)
	}

	return grouped
}

// Map maps a list using a function.
func Map[T any, R any](items []T, fn func(T) R) []R {
	mapped := make([]R, 0, len(items))

	for _, item := range items {
		mapped = append(mapped, fn(item))
	}

	return mapped
}

// Filter filters a list using a function.
func Filter[T any](items []T, fn func(T) bool) []T {
	filtered := make([]T, 0, len(items))

	for _, item := range items {
		if fn(item) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// Index creates a map from a list using a function.
func Index[T any](items []T, fn func(T) string) map[string]T {
	index := make(map[string]T)

	for _, item := range items {
		index[fn(item)] = item
	}

	return index
}
