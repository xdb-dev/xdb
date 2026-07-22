package api

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

// parseURI parses raw as a [core.URI] and enforces the calling method's
// expected depth and attribute policy.
//
// minDepth and maxDepth bound (*core.URI).Depth(): 1 = namespace only,
// 2 = namespace+schema, 3 = namespace+schema+id. When allowAttr is false,
// a URI carrying a #attr fragment is rejected.
//
// A malformed URI is returned as-is from [core.ParseURI] (already wrapping
// [core.ErrInvalidURI]). A depth or attribute violation returns a new error
// that also wraps [core.ErrInvalidURI], naming method and the expected
// shape, so callers map to the same RPC error code as a malformed URI.
func parseURI(raw, method string, minDepth, maxDepth int, allowAttr bool) (*core.URI, error) {
	uri, err := core.ParseURI(raw)
	if err != nil {
		return nil, err
	}

	if depth := uri.Depth(); depth < minDepth || depth > maxDepth {
		return nil, fmt.Errorf("%s expects %s, got %q: %w",
			method, shapeHint(minDepth, maxDepth), raw, core.ErrInvalidURI)
	}

	if !allowAttr && uri.Attr() != "" {
		return nil, fmt.Errorf("%s does not accept an attribute (#%s): %w",
			method, uri.Attr(), core.ErrInvalidURI)
	}

	return uri, nil
}

// shapeHint describes the URI shape a [minDepth, maxDepth] range accepts,
// for use in parseURI error messages.
func shapeHint(minDepth, maxDepth int) string {
	if minDepth == maxDepth {
		return depthShape(minDepth)
	}

	switch {
	case minDepth == 1 && maxDepth == 2:
		return "a namespace or schema URI xdb://ns[/schema]"
	case minDepth == 1 && maxDepth == 3:
		return "a namespace, schema, or record URI xdb://ns[/schema[/id]]"
	default:
		return fmt.Sprintf("a URI of depth %d-%d", minDepth, maxDepth)
	}
}

// depthShape describes a single exact depth, for use in parseURI error
// messages.
func depthShape(depth int) string {
	switch depth {
	case 1:
		return "a namespace URI xdb://ns"
	case 2:
		return "a schema URI xdb://ns/schema"
	case 3:
		return "a record URI xdb://ns/schema/id"
	default:
		return fmt.Sprintf("a URI of depth %d", depth)
	}
}
