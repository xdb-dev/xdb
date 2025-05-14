package x

import "github.com/xdb-dev/xdb/types"

// GroupTuples groups a list of tuples by their kind and id.
func GroupTuples(tuples ...*types.Tuple) map[string]map[string][]*types.Tuple {
	grouped := make(map[string]map[string][]*types.Tuple)

	for _, tuple := range tuples {
		kind := tuple.Kind()
		id := tuple.ID()

		if _, ok := grouped[kind]; !ok {
			grouped[kind] = make(map[string][]*types.Tuple)
		}

		if _, ok := grouped[kind][id]; !ok {
			grouped[kind][id] = make([]*types.Tuple, 0)
		}

		grouped[kind][id] = append(grouped[kind][id], tuple)
	}

	return grouped
}
