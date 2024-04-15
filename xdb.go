package xdb

import "context"

// RecordStorer defines the interface for storing and retrieving records.
type RecordStorer interface {
	GetRecord(ctx context.Context, key *Key) (*Record, error)
	PutRecord(ctx context.Context, record *Record) error
	DeleteRecord(ctx context.Context, key *Key) error
}
