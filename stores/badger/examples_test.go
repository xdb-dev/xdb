package badger_test

import (
	"context"
	"fmt"
	"time"

	badgerv4 "github.com/dgraph-io/badger/v4"
	"github.com/xdb-dev/xdb"
	"github.com/xdb-dev/xdb/stores/badger"
)

func Example() {
	ctx := context.Background()
	key := xdb.NewKey("User", "1")

	user := xdb.NewRecord(key,
		xdb.NewTuple(key, "name", "Alice"),
		xdb.NewTuple(key, "age", 30),
		xdb.NewTuple(key, "birthday",
			time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC),
		),
	)

	db, err := badgerv4.Open(
		badgerv4.DefaultOptions("").WithInMemory(true),
	)
	if err != nil {
		panic(err)
	}

	store := badger.NewBadgerStore(db)

	// save the user
	if err := store.PutRecord(ctx, user); err != nil {
		panic(err)
	}

	// retrieve the user
	got, err := store.GetRecord(ctx, key)
	if err != nil {
		panic(err)
	}

	for _, t := range got.Tuples() {
		fmt.Printf("%s: %v\n", t.Name(), t.Value())
	}

	// delete the user
	if err := store.DeleteRecord(ctx, key); err != nil {
		panic(err)
	}
}
