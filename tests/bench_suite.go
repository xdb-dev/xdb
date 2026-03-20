package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/xdb-dev/xdb/store"
)

// BenchmarkSuite runs a standard set of benchmarks against any [store.Store].
type BenchmarkSuite struct {
	newStore func() store.Store
}

// NewBenchmarkSuite creates a new suite using the given factory.
// The factory is called during setup to provide a fresh store.
func NewBenchmarkSuite(fn func() store.Store) *BenchmarkSuite {
	return &BenchmarkSuite{newStore: fn}
}

// Run runs all store benchmarks as sub-benchmarks of b.
func (s *BenchmarkSuite) Run(b *testing.B) {
	b.Helper()

	b.Run("CreateRecord", s.benchCreateRecord)
	b.Run("GetRecord", s.benchGetRecord)
	b.Run("UpdateRecord", s.benchUpdateRecord)
	b.Run("UpsertRecord", s.benchUpsertRecord)
	b.Run("DeleteRecord", s.benchDeleteRecord)
	b.Run("ListRecords/10", func(b *testing.B) { s.benchListRecords(b, 10) })
	b.Run("ListRecords/100", func(b *testing.B) { s.benchListRecords(b, 100) })
	b.Run("ListRecords/1000", func(b *testing.B) { s.benchListRecords(b, 1000) })
}

func (s *BenchmarkSuite) benchCreateRecord(b *testing.B) {
	ctx := context.Background()
	st := s.newStore()

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		r := FakePost(fmt.Sprintf("bench-%d", i))
		if err := st.CreateRecord(ctx, r); err != nil {
			b.Fatal(err)
		}
	}
}

func (s *BenchmarkSuite) benchGetRecord(b *testing.B) {
	ctx := context.Background()
	st := s.newStore()

	r := FakePost("bench-get")
	if err := st.CreateRecord(ctx, r); err != nil {
		b.Fatal(err)
	}

	uri := r.URI()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := st.GetRecord(ctx, uri); err != nil {
			b.Fatal(err)
		}
	}
}

func (s *BenchmarkSuite) benchUpdateRecord(b *testing.B) {
	ctx := context.Background()
	st := s.newStore()

	r := FakePost("bench-update")
	if err := st.CreateRecord(ctx, r); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		r.Set("title", fmt.Sprintf("Updated %d", i))
		if err := st.UpdateRecord(ctx, r); err != nil {
			b.Fatal(err)
		}
	}
}

func (s *BenchmarkSuite) benchUpsertRecord(b *testing.B) {
	ctx := context.Background()
	st := s.newStore()

	r := FakePost("bench-upsert")

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		r.Set("title", fmt.Sprintf("Upserted %d", i))
		if err := st.UpsertRecord(ctx, r); err != nil {
			b.Fatal(err)
		}
	}
}

func (s *BenchmarkSuite) benchDeleteRecord(b *testing.B) {
	ctx := context.Background()
	st := s.newStore()

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		r := FakePost(fmt.Sprintf("bench-del-%d", i))
		if err := st.CreateRecord(ctx, r); err != nil {
			b.Fatal(err)
		}
		if err := st.DeleteRecord(ctx, r.URI()); err != nil {
			b.Fatal(err)
		}
	}
}

func (s *BenchmarkSuite) benchListRecords(b *testing.B, n int) {
	ctx := context.Background()
	st := s.newStore()

	for i := range n {
		r := FakePost(fmt.Sprintf("bench-list-%d", i))
		if err := st.CreateRecord(ctx, r); err != nil {
			b.Fatal(err)
		}
	}

	uri := FakePost("unused").URI()
	q := &store.Query{URI: uri, Limit: 20}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := st.ListRecords(ctx, q); err != nil {
			b.Fatal(err)
		}
	}
}
