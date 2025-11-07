package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

// ErrInvalidURI is returned when an invalid URI is encountered.
var ErrInvalidURI = errors.New("[xdb/core] invalid URI")

// URI is a reference to XDB data.
//
// The general format is:
//
//	xdb:// REPOSITORY [ / RECORD ] [ #ATTRIBUTE ]
//
// REPOSITORY is a data repository.
// RECORD is a group of tuples with the same ID.
// ATTRIBUTE is a specific attribute of a record.
type URI struct {
	repo string
	id   ID
	attr Attr
}

// NewURI creates a new URI.
func NewURI(parts ...any) *URI {
	if len(parts) == 0 || len(parts) > 3 {
		panic(ErrInvalidURI)
	}

	var repo string
	var id ID
	var attr Attr

	if len(parts) >= 1 {
		if parts[0] == nil {
			panic(ErrInvalidURI)
		}
		r, ok := parts[0].(string)
		if !ok {
			panic(ErrInvalidURI)
		}
		repo = r
	}

	if len(parts) >= 2 {
		if parts[1] == nil {
			panic(ErrInvalidURI)
		}
		switch v := parts[1].(type) {
		case string:
			id = NewID(v)
		case []string:
			id = NewID(v...)
		case ID:
			id = v
		default:
			panic(ErrInvalidURI)
		}
	}

	if len(parts) >= 3 {
		if parts[2] == nil {
			panic(ErrInvalidURI)
		}
		switch v := parts[2].(type) {
		case string:
			attr = NewAttr(v)
		case []string:
			attr = NewAttr(v...)
		case Attr:
			attr = v
		default:
			panic(ErrInvalidURI)
		}
	}

	return &URI{repo: repo, id: id, attr: attr}
}

// Repo returns the repository name part of the URI.
func (u *URI) Repo() string {
	return u.repo
}

// ID returns the ID part of the URI.
func (u *URI) ID() ID {
	return u.id
}

// Attr returns the attribute part of the URI.
func (u *URI) Attr() Attr {
	return u.attr
}

// String returns the URI as a string.
func (u *URI) String() string {
	if len(u.attr) > 0 && len(u.id) > 0 && len(u.repo) > 0 {
		return fmt.Sprintf("xdb://%s/%s#%s", u.repo, u.id.String(), u.attr.String())
	} else if len(u.id) > 0 && len(u.repo) > 0 {
		return fmt.Sprintf("xdb://%s/%s", u.repo, u.id.String())
	} else if len(u.repo) > 0 {
		return fmt.Sprintf("xdb://%s", u.repo)
	}

	panic(ErrInvalidURI)
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

// uriRegex matches XDB URIs in the format: xdb://REPOSITORY[/RECORD][#ATTRIBUTE]
var uriRegex = regexp.MustCompile(`^xdb://([^/]+)(?:/([^#]+))?(?:#(.+))?$`)

// ParseURI parses a URI string into a URI struct.
// The URI format is: xdb://REPOSITORY[/RECORD][#ATTRIBUTE]
// where RECORD can be a hierarchical ID (e.g., 123/456/789)
// and ATTRIBUTE can be a nested attribute (e.g., profile.name)
func ParseURI(uri string) (*URI, error) {
	matches := uriRegex.FindStringSubmatch(uri)
	if matches == nil {
		return nil, ErrInvalidURI
	}

	if len(matches) > 1 && !isValidRepo(matches[1]) {
		return nil, ErrInvalidRepo
	}

	var id ID
	var err error

	if matches[2] != "" {
		id, err = ParseID(matches[2])
		if err != nil {
			return nil, err
		}
	}

	var attr Attr
	if matches[3] != "" {
		attr, err = ParseAttr(matches[3])
		if err != nil {
			return nil, err
		}
	}

	return &URI{
		repo: matches[1],
		id:   id,
		attr: attr,
	}, nil
}

// MustParseURI parses a URI string into a URI struct.
// It panics if the URI is invalid.
func MustParseURI(raw string) *URI {
	uri, err := ParseURI(raw)
	if err != nil {
		panic(err)
	}
	return uri
}

// splitNonEmpty splits a string by the given separator and returns the parts.
// Returns nil if the string is empty to distinguish between "" and "a/b".
func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, sep)
}
