package store

import "github.com/xdb-dev/xdb/core"

// MergeTuples overlays puts onto current, replacing tuples that share
// an attr. It is a helper for drivers that reconstruct a record's full
// tuple set (file- or row-per-record backends) to apply an [OpPatch].
func MergeTuples(current, puts []*core.Tuple) []*core.Tuple {
	merged := make(map[string]*core.Tuple, len(current)+len(puts))
	for _, tuple := range current {
		merged[tuple.Attr()] = tuple
	}
	for _, tuple := range puts {
		merged[tuple.Attr()] = tuple
	}

	out := make([]*core.Tuple, 0, len(merged))
	for _, tuple := range merged {
		out = append(out, tuple)
	}
	return out
}

// RemoveAttrs returns current without the named attrs — the tuple-set
// form of an [OpDelete] with attrs, for drivers that reconstruct the
// full set before writing it back.
func RemoveAttrs(current []*core.Tuple, attrs []string) []*core.Tuple {
	drop := make(map[string]struct{}, len(attrs))
	for _, attr := range attrs {
		drop[attr] = struct{}{}
	}

	out := make([]*core.Tuple, 0, len(current))
	for _, tuple := range current {
		if _, ok := drop[tuple.Attr()]; !ok {
			out = append(out, tuple)
		}
	}
	return out
}
