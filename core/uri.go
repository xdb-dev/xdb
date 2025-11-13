package core

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrInvalidURI is returned when an invalid URI is encountered.
	ErrInvalidURI = errors.New("[xdb/core] invalid URI")
)

// URI is a reference to XDB data.
//
// The general format is:
//
//	xdb:// NS [ / SCHEMA ] [ / ID ] [ #ATTRIBUTE ]
//
// NS identifies the data repository.
// SCHEMA is the collection name.
// ID is the record identifier.
// ATTRIBUTE is a specific attribute of a record.
type URI struct {
	ns     *NS
	schema *Schema
	id     *ID
	attr   *Attr
}

// NS returns the namespace part of the URI.
func (u *URI) NS() *NS { return u.ns }

// Schema returns the schema part of the URI.
func (u *URI) Schema() *Schema { return u.schema }

// ID returns the ID part of the URI.
func (u *URI) ID() *ID { return u.id }

// Attr returns the attribute part of the URI.
func (u *URI) Attr() *Attr { return u.attr }

// Equals returns true if this URI is equal to the other URI.
func (u *URI) Equals(other *URI) bool {
	return u.ns.Equals(other.ns) &&
		(u.schema == nil && other.schema == nil) ||
		(u.schema != nil && other.schema != nil && u.schema.Equals(other.schema)) &&
			(u.id == nil && other.id == nil) ||
		(u.id != nil && other.id != nil && u.id.Equals(other.id)) &&
			(u.attr == nil && other.attr == nil) ||
		(u.attr != nil && other.attr != nil && u.attr.Equals(other.attr))
}

// Path returns the URI without the scheme.
func (u *URI) Path() string {
	var b strings.Builder
	if u.ns != nil {
		b.WriteString(u.ns.String())
	}
	if u.schema != nil {
		b.WriteString("/")
		b.WriteString(u.schema.String())
	}
	if u.id != nil {
		b.WriteString("/")
		b.WriteString(u.id.String())
	}
	if u.attr != nil {
		b.WriteString("#")
		b.WriteString(u.attr.String())
	}
	return b.String()
}

// String returns the URI as a string.
func (u *URI) String() string {
	return "xdb://" + u.Path()
}

// MarshalJSON implements the json.Marshaler interface.
func (u *URI) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (u *URI) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	uri, err := ParseURI(str)
	if err != nil {
		return err
	}

	*u = *uri

	return nil
}

// uriRegex matches XDB URIs in the format: xdb://NS[/SCHEMA][/ID][#ATTRIBUTE]
var uriRegex = regexp.MustCompile(`^xdb://([^/]+)(?:/([^#]+))?(?:#(.+))?$`)

// ParsePath parses a path string into a URI struct.
// The path format is: NS[/SCHEMA][/ID][#ATTRIBUTE]
func ParsePath(path string) (*URI, error) {
	matches := uriRegex.FindStringSubmatch(path)
	if matches == nil {
		return nil, ErrInvalidURI
	}

	var ns *NS
	var schema *Schema
	var id *ID
	var attr *Attr
	var err error

	if len(matches) > 1 {
		ns, err = ParseNS(matches[1])
		if err != nil {
			return nil, err
		}
	}

	if len(matches) > 2 && matches[2] != "" {
		schema, err = ParseSchema(matches[2])
		if err != nil {
			return nil, err
		}
	}

	if len(matches) > 3 && matches[3] != "" {
		id, err = ParseID(matches[3])
		if err != nil {
			return nil, err
		}
	}

	if len(matches) > 4 && matches[4] != "" {
		attr, err = ParseAttr(matches[4])
		if err != nil {
			return nil, err
		}
	}

	return &URI{
		ns:     ns,
		schema: schema,
		id:     id,
		attr:   attr,
	}, nil
}

// ParseURI parses a URI string into a URI struct.
// The URI format is: xdb://NS[/SCHEMA][/ID][#ATTRIBUTE]
func ParseURI(uri string) (*URI, error) {
	matches := uriRegex.FindStringSubmatch(uri)
	if matches == nil {
		return nil, ErrInvalidURI
	}

	var ns *NS
	var schema *Schema
	var id *ID
	var attr *Attr
	var err error

	if len(matches) > 1 {
		ns, err = ParseNS(matches[1])
		if err != nil {
			return nil, err
		}
	}

	if len(matches) > 2 && matches[2] != "" {
		schema, err = ParseSchema(matches[2])
		if err != nil {
			return nil, err
		}
	}

	if len(matches) > 3 && matches[3] != "" {
		id, err = ParseID(matches[3])
		if err != nil {
			return nil, err
		}
	}

	if len(matches) > 4 && matches[4] != "" {
		attr, err = ParseAttr(matches[4])
		if err != nil {
			return nil, err
		}
	}

	return &URI{
		ns:     ns,
		schema: schema,
		id:     id,
		attr:   attr,
	}, nil
}

var componentRegex = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

func isValidComponent(raw string) bool {
	return componentRegex.MatchString(raw)
}
