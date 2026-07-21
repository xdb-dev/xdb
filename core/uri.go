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
// NS identifies the namespace.
// SCHEMA is the schema name.
// ID is the record identifier.
// ATTRIBUTE is a specific attribute of a record.
//
// URIs are immutable; construct via [NewURI], [ParseURI], or [ParsePath].
type URI struct {
	ns     string
	schema string
	id     string
	attr   string
}

// NS returns the namespace part of the URI.
func (u *URI) NS() string { return u.ns }

// Schema returns the schema part of the URI, or "" if absent.
func (u *URI) Schema() string { return u.schema }

// ID returns the record ID part of the URI, or "" if absent.
func (u *URI) ID() string { return u.id }

// Attr returns the attribute part of the URI, or "" if absent.
func (u *URI) Attr() string { return u.attr }

// SchemaURI returns a new URI containing only the namespace and schema components.
func (u *URI) SchemaURI() *URI {
	return &URI{ns: u.ns, schema: u.schema}
}

// RecordURI returns a new URI containing only the namespace, schema,
// and ID components — the record path with any attr dropped.
func (u *URI) RecordURI() *URI {
	return &URI{ns: u.ns, schema: u.schema, id: u.id}
}

// RecordPath returns the ns/schema/id record-path string, dropping any
// attr. It is the canonical record key and equals RecordURI().Path()
// without the intermediate URI allocation.
func (u *URI) RecordPath() string {
	return u.ns + "/" + u.schema + "/" + u.id
}

// Path returns the URI without the scheme.
func (u *URI) Path() string {
	var b strings.Builder
	b.WriteString(u.ns)
	if u.schema != "" {
		b.WriteString("/")
		b.WriteString(u.schema)
	}
	if u.id != "" {
		b.WriteString("/")
		b.WriteString(u.id)
	}
	if u.attr != "" {
		b.WriteString("#")
		b.WriteString(u.attr)
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

// NewURI builds a record- or schema-level URI from parts.
//
// The first part (if present) is the schema, the second is the ID.
// Empty trailing parts are allowed: NewURI("ns"), NewURI("ns", "posts").
// Returns an error if any component is invalid.
func NewURI(ns string, parts ...string) (*URI, error) {
	if len(parts) > 2 {
		return nil, errors.Wrap(ErrInvalidURI, "parts", strings.Join(parts, "/"))
	}

	if err := validateComponent("ns", ns, false); err != nil {
		return nil, err
	}

	uri := &URI{ns: ns}

	if len(parts) >= 1 {
		if err := validateComponent("schema", parts[0], false); err != nil {
			return nil, err
		}
		uri.schema = parts[0]
	}

	if len(parts) == 2 {
		if err := validateComponent("id", parts[1], true); err != nil {
			return nil, err
		}
		uri.id = parts[1]
	}

	return uri, nil
}

// MustNewURI is like [NewURI] but panics if any component is invalid.
func MustNewURI(ns string, parts ...string) *URI {
	uri, err := NewURI(ns, parts...)
	if err != nil {
		panic(err)
	}
	return uri
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

	if err := validateComponent("ns", parsed.Host, false); err != nil {
		return nil, err
	}

	out := &URI{ns: parsed.Host}

	if len(parsed.Path) == 0 {
		return out, nil
	}

	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")

	if err := validateComponent("schema", pathParts[0], false); err != nil {
		return nil, err
	}
	out.schema = pathParts[0]

	if len(pathParts) > 1 {
		id := strings.Join(pathParts[1:], "/")
		if err := validateComponent("id", id, true); err != nil {
			return nil, err
		}
		out.id = id
	}

	if parsed.Fragment != "" {
		if err := validateComponent("attr", parsed.Fragment, false); err != nil {
			return nil, err
		}
		out.attr = parsed.Fragment
	}

	return out, nil
}

// MustParseURI is like ParseURI but panics if the URI is invalid.
func MustParseURI(uri string) *URI {
	parsed, err := ParseURI(uri)
	if err != nil {
		panic(err)
	}
	return parsed
}

// validateComponent validates a single URI component. NS, schema, and attr
// forbid '/'; ID allows it (trailing path segments join into the ID).
func validateComponent(kind, raw string, allowSlash bool) error {
	if raw == "" {
		return errors.Wrap(ErrInvalidURI, kind, "empty")
	}
	for _, ch := range raw {
		ok := (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '.' || ch == '_' || ch == '-' ||
			(allowSlash && ch == '/')
		if !ok {
			return errors.Wrap(ErrInvalidURI, kind, raw)
		}
	}
	return nil
}
