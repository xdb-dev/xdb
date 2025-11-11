package core

import (
	"regexp"

	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrInvalidRepo is returned when an invalid repo name is encountered.
	ErrInvalidRepo = errors.New("[xdb/core] invalid repo name")
)

// Mode defines whether a repository enforces a schema or not.
type Mode string

const (
	// ModeFlexible represents schemaless repositories where records can have any attributes.
	ModeFlexible Mode = "flexible"

	// ModeStrict represents structured repositories with enforced schema.
	ModeStrict Mode = "strict"
)

// Repo is a data repository.
type Repo struct {
	name   string
	mode   Mode
	schema *Schema
}

// NewRepo creates a new repository.
//
//	NewRepo(schema) creates a strict repository with a schema.
//	NewRepo(name) creates a flexible repository without a schema.
func NewRepo(param any) (*Repo, error) {
	switch v := param.(type) {
	case *Schema:
		if !isValidRepo(v.Name) {
			return nil, ErrInvalidRepo
		}
		return &Repo{
			name:   v.Name,
			mode:   ModeStrict,
			schema: v,
		}, nil
	case Schema:
		if !isValidRepo(v.Name) {
			return nil, ErrInvalidRepo
		}
		return &Repo{
			name:   v.Name,
			mode:   ModeStrict,
			schema: &v,
		}, nil
	case string:
		if !isValidRepo(v) {
			return nil, ErrInvalidRepo
		}
		return &Repo{
			name: v,
			mode: ModeFlexible,
		}, nil
	default:
		return nil, ErrInvalidRepo
	}
}

// Name returns the name of the repo.
func (r *Repo) Name() string {
	return r.name
}

// Schema returns the schema of the repo.
func (r *Repo) Schema() *Schema {
	return r.schema
}

// Mode returns the mode of the repo.
func (r *Repo) Mode() Mode {
	return r.mode
}

// String returns the repo as a string.
func (r *Repo) String() string {
	return r.name
}

// URI returns the URI of the repo.
func (r *Repo) URI() *URI {
	return &URI{repo: r.name}
}

var repoRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// isValidRepo checks if a repository name is valid.
// Valid names contain only alphanumeric characters, dots, hyphens, and underscores.
func isValidRepo(name string) bool {
	return repoRegex.MatchString(name)
}
