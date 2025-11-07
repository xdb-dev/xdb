// Package driver defines core driver interfaces for XDB database backends.
package driver

import (
	"context"
	"errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("xdb/driver: not found")
)

// RepoReader is an interface for reading repositories.
type RepoReader interface {
	GetRepo(ctx context.Context, name string) (*core.Repo, error)
	ListRepos(ctx context.Context) ([]*core.Repo, error)
}

// RepoWriter is an interface for creating repositories.
type RepoWriter interface {
	MakeRepo(ctx context.Context, repo *core.Repo) error
	DeleteRepo(ctx context.Context, name string) error
}

// RepoDriver is an interface for managing repositories.
type RepoDriver interface {
	RepoReader
	RepoWriter
}

// TupleReader is an interface for reading tuples.
type TupleReader interface {
	GetTuples(ctx context.Context, uris []*core.URI) ([]*core.Tuple, []*core.URI, error)
}

// TupleWriter is an interface for writing & deleting tuples.
type TupleWriter interface {
	PutTuples(ctx context.Context, tuples []*core.Tuple) error
	DeleteTuples(ctx context.Context, uris []*core.URI) error
}

// TupleDriver is an interface for managing tuples.
type TupleDriver interface {
	TupleReader
	TupleWriter
}

// RecordReader is an interface for reading records.
type RecordReader interface {
	GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error)
}

// RecordWriter is an interface for writing & deleting records.
type RecordWriter interface {
	PutRecords(ctx context.Context, records []*core.Record) error
	DeleteRecords(ctx context.Context, uris []*core.URI) error
}

// RecordDriver is an interface for managing records.
type RecordDriver interface {
	RecordReader
	RecordWriter
}
