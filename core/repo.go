package core

import (
	"regexp"

	"github.com/gojekfarm/xtools/errors"
)

// ErrInvalidRepo is returned when an invalid repo name is encountered.
var ErrInvalidRepo = errors.New("[xdb/core] invalid repo name")

// Repo is a data repository.
type Repo struct {
	name string
}

// NewRepo creates a new repo.
func NewRepo(name string) (*Repo, error) {
	if !isValidRepo(name) {
		return nil, ErrInvalidRepo
	}

	return &Repo{name: name}, nil
}

// MustNewRepo creates a new repo.
// It panics if the repo name is invalid.
func MustNewRepo(name string) *Repo {
	repo, err := NewRepo(name)
	if err != nil {
		panic(err)
	}
	return repo
}

// Name returns the name of the repo.
func (r *Repo) Name() string {
	return r.name
}

// String returns the repo as a string.
func (r Repo) String() string {
	return r.name
}

// URI returns the URI of the repo.
func (r Repo) URI() *URI {
	return &URI{repo: r.name}
}

var repoRegex = regexp.MustCompile(`^[a-zA-Z0-9.-_]+$`)

func isValidRepo(name string) bool {
	return repoRegex.MatchString(name)
}
