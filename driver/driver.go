// Package driver defines core driver interfaces for XDB database backends.
package driver

import (
	"context"
	"errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrRepoNotFound is returned when a repository is not found.
	ErrRepoNotFound = errors.New("xdb/driver: repo not found")
)

// RepoReader is an interface for reading repositories.
type RepoReader interface {
	GetRepo(ctx context.Context, name string) (*core.Repo, error)
	ListRepos(ctx context.Context) ([]*core.Repo, error)
}

// RepoWriter is an interface for writing & deleting repositories.
type RepoWriter interface {
	PutRepo(ctx context.Context, repo *core.Repo) error
	DeleteRepo(ctx context.Context, name string) error
}

// RepoDriver is an interface for managing repositories.
type RepoDriver interface {
	RepoReader
	RepoWriter
}

// TupleReader is an interface for reading tuples.
type TupleReader interface {
	GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error)
}

// TupleWriter is an interface for writing & deleting tuples.
type TupleWriter interface {
	PutTuples(ctx context.Context, tuples []*core.Tuple) error
	DeleteTuples(ctx context.Context, keys []*core.Key) error
}

// RecordReader is an interface for reading records.
type RecordReader interface {
	GetRecords(ctx context.Context, keys []*core.Key) ([]*core.Record, []*core.Key, error)
}

// RecordWriter is an interface for writing & deleting records.
type RecordWriter interface {
	PutRecords(ctx context.Context, records []*core.Record) error
	DeleteRecords(ctx context.Context, keys []*core.Key) error
}
