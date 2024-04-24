package xdb

import "context"

// RecordStorer defines the interface for storing and retrieving records.
type RecordStorer interface {
	GetRecord(ctx context.Context, key *Key) (*Record, error)
	PutRecord(ctx context.Context, record *Record) error
	DeleteRecord(ctx context.Context, key *Key) error
}

type RecordSet = ResultSet[Record]

// RecordQueryer defines the interface for querying records.
type RecordQueryer interface {
	QueryRecords(ctx context.Context, q *Query) (*RecordSet, error)
}
