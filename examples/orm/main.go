package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/encoding/xdbstruct"
)

type User struct {
	ID    string `xdb:"id,primary_key"`
	Name  string `xdb:"name"`
	Email string `xdb:"email"`
	Age   int    `xdb:"age"`
}

type Post struct {
	Website   string    `xdb:"website,ns=com.example,schema=posts"`
	ID        string    `xdb:"id,primary_key"`
	Title     string    `xdb:"title"`
	Content   string    `xdb:"content"`
	Author    User      `xdb:"author"`
	CreatedAt time.Time `xdb:"created_at"`
}

func main() {
	ctx := context.Background()
	user := User{
		ID:    "123",
		Name:  "John Doe",
		Email: "john.doe@example.com",
		Age:   30,
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{Tag: "xdb"})
	record, err := encoder.Encode(user)
	if err != nil {
		panic(err)
	}

	store := xdbmemory.New()
	err = store.PutRecords(ctx, []*core.Record{record})
	if err != nil {
		panic(err)
	}

	records, _, err := store.GetRecords(ctx, []*core.URI{record.URI()})
	if err != nil {
		panic(err)
	}

	fmt.Println(records)
}
