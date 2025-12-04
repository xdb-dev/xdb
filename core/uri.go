package core

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

// ErrInvalidURI is returned when an invalid URI is encountered.
var ErrInvalidURI = errors.New("[xdb/core] invalid URI")

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
		((u.schema == nil && other.schema == nil) ||
			(u.schema != nil && other.schema != nil && u.schema.Equals(other.schema))) &&
		((u.id == nil && other.id == nil) ||
			(u.id != nil && other.id != nil && u.id.Equals(other.id))) &&
		((u.attr == nil && other.attr == nil) ||
			(u.attr != nil && other.attr != nil && u.attr.Equals(other.attr)))
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

// ParsePath parses a path string into a URI struct.
// The path format is: NS[/SCHEMA][/ID][#ATTRIBUTE].
func ParsePath(path string) (*URI, error) {
	return ParseURI("xdb://" + path)
}

// ParseURI parses a URI string into a URI struct.
// The URI format is: xdb://NS[/SCHEMA][/ID][#ATTRIBUTE]
func ParseURI(uri string) (*URI, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Join(err, ErrInvalidURI)
	}

	if parsed.Scheme != "xdb" {
		return nil, ErrInvalidURI
	}

	if parsed.Host == "" {
		return nil, ErrInvalidURI
	}

	if parsed.User != nil {
		return nil, ErrInvalidURI
	}

	var ns *NS
	var schema *Schema
	var id *ID
	var attr *Attr

	ns, err = ParseNS(parsed.Host)
	if err != nil {
		return nil, err
	}

	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")

	if len(parsed.Path) == 0 {
		return &URI{ns: ns}, nil
	}

	schema, err = ParseSchema(pathParts[0])
	if err != nil {
		return nil, err
	}

	if len(pathParts) > 1 {
		id, err = ParseID(strings.Join(pathParts[1:], "/"))
		if err != nil {
			return nil, err
		}
	}

	if parsed.Fragment != "" {
		attr, err = ParseAttr(parsed.Fragment)
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

func isValidComponent(raw string) bool {
	if raw == "" {
		return false
	}
	for _, ch := range raw {
		if (ch < 'a' || ch > 'z') &&
			(ch < 'A' || ch > 'Z') &&
			(ch < '0' || ch > '9') &&
			ch != '.' && ch != '_' && ch != '-' && ch != '/' {
			return false
		}
	}
	return true
}
