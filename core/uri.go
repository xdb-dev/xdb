package core

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/gojekfarm/xtools/errors"
)

// ErrInvalidURI is returned when an invalid URI is encountered.
var ErrInvalidURI = errors.New("[xdb/core] invalid URI")

// URI is a reference to XDB data.
//
// The general format is:
//
//	xdb:// REPOSITORY [ / COLLECTION ] [ / RECORD ] [ #ATTRIBUTE ]
//
// REPOSITORY is a data repository.
// COLLECTION is a group of records with the same schema.
// RECORD is a group of tuples with the same ID.
// ATTRIBUTE is a specific attribute of a tuple.
type URI struct {
	repo string
	coll string
	id   ID
	attr Attr
}

// NewURI creates a new URI.
func NewURI(parts ...any) *URI {
	if len(parts) == 0 || len(parts) > 4 {
		panic(ErrInvalidURI)
	}

	var repo string
	var coll string
	var id ID
	var attr Attr

	if len(parts) >= 1 && parts[0] != nil {
		r, ok := parts[0].(string)
		if !ok {
			panic(ErrInvalidURI)
		}
		repo = r
	}

	if len(parts) >= 2 && parts[1] != nil {
		c, ok := parts[1].(string)
		if !ok {
			panic(ErrInvalidURI)
		}
		coll = c
	}

	if len(parts) >= 3 && parts[2] != nil {
		switch v := parts[2].(type) {
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

	if len(parts) >= 4 && parts[3] != nil {
		switch v := parts[3].(type) {
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

	return &URI{repo: repo, coll: coll, id: id, attr: attr}
}

// Repo returns the repository of the URI.
func (u *URI) Repo() string {
	return u.repo
}

// Collection returns the collection of the URI.
func (u *URI) Collection() string {
	return u.coll
}

// ID returns the ID of the URI.
func (u *URI) ID() ID {
	return u.id
}

// Attr returns the attribute of the URI.
func (u *URI) Attr() Attr {
	return u.attr
}

// String returns the URI as a string.
func (u *URI) String() string {
	if len(u.attr) > 0 && len(u.id) > 0 && len(u.coll) > 0 && len(u.repo) > 0 {
		return fmt.Sprintf("xdb://%s/%s/%s#%s", u.repo, u.coll, u.id.String(), u.attr.String())
	} else if len(u.id) > 0 && len(u.coll) > 0 && len(u.repo) > 0 {
		return fmt.Sprintf("xdb://%s/%s/%s", u.repo, u.coll, u.id.String())
	} else if u.coll != "" && u.repo != "" {
		return fmt.Sprintf("xdb://%s/%s", u.repo, u.coll)
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

// uriRegex matches XDB URIs in the format: xdb://REPOSITORY[/COLLECTION][/RECORD][#ATTRIBUTE]
var uriRegex = regexp.MustCompile(`^xdb://([^/]+)(?:/([^/#]+))?(?:/([^#]+))?(?:#(.+))?$`)

// ParseURI parses a URI string into a URI struct.
// The URI format is: xdb://REPOSITORY[/COLLECTION][/RECORD][#ATTRIBUTE]
// where RECORD can be a hierarchical ID (e.g., 123/456/789)
// and ATTRIBUTE can be a nested attribute (e.g., profile.name)
func ParseURI(uri string) (*URI, error) {
	matches := uriRegex.FindStringSubmatch(uri)
	if matches == nil {
		return nil, ErrInvalidURI
	}

	if len(matches) > 1 && !isValidRepo(matches[0]) {
		return nil, ErrInvalidRepo
	}

	return &URI{
		repo: matches[1],
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
