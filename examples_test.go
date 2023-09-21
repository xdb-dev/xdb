package xdb_test

import (
	"context"
	"log"
	"time"

	"github.com/xdb-dev/xdb"
)

func ExampleTupleStore_PutTuples() {
	ctx := context.Background()
	var store xdb.TupleStore

	// Create a new post
	post := xdb.NewKey("Post", "9bsv0s45jdj002vjlkt0")
	author := xdb.NewKey("User", "9bsv0s3p32i002qvnf50")
	tags := []*xdb.Key{
		xdb.NewKey("Tag", "xdb"),
		xdb.NewKey("Tag", "database"),
	}

	// Create post's attributes and edges
	attrs := []*xdb.Tuple{
		xdb.NewAttr(post, "title", xdb.String("Introduction to xdb")),
		xdb.NewAttr(post, "body", xdb.String("...")),
		xdb.NewAttr(post, "published_at", xdb.Time(time.Now())),
		xdb.NewEdge(post, "author", author),
		xdb.NewEdge(post, "tags", tags[0]),
		xdb.NewEdge(post, "tags", tags[1]),
	}

	// Save attributes
	if err := store.PutTuples(ctx, attrs...); err != nil {
		panic(err)
	}
}

// ... assume there are posts in the store
var posts = []*xdb.Key{
	xdb.NewKey("Post", "9bsv0s0r2gc002vs02tg"),
	xdb.NewKey("Post", "9bsv0s0r2gc002vs02u0"),
	xdb.NewKey("Post", "9bsv0s0r2gc002vs02ug"),
}

func ExampleTupleStore_GetTuples_attributes() {
	ctx := context.Background()
	var store xdb.TupleStore

	// attribute references can be created individually
	refs := []*xdb.Ref{
		xdb.AttrRef(posts[0], "title"),
		xdb.AttrRef(posts[1], "title"),
		xdb.AttrRef(posts[2], "title"),
	}

	// or using the MakeAttrRefs helper, which creates a
	// cross-product of keys and attribute names
	refs = xdb.MakeAttrRefs(posts, "title", "body", "published_at")

	attrs, err := store.GetTuples(ctx, refs...)
	if err != nil {
		panic(err)
	}

	for _, attr := range attrs {
		log.Printf("%s: %s", attr.Ref(), attr.Value())
	}
}

func ExampleTupleStore_GetTuples_edges() {
	ctx := context.Background()
	var store xdb.TupleStore

	user := xdb.NewKey("User", "9bsv0s3p32i002qvnf50")

	// 1:1 edges behave like attributes
	refs := []*xdb.Ref{
		xdb.EdgeRef(posts[0], "author"),
		xdb.EdgeRef(posts[1], "author"),
		xdb.EdgeRef(posts[2], "author"),
	}

	// 1:N edges can be fetched by specifying the target key.
	// This is useful to check for the existence of an edge.
	refs = append(refs,
		xdb.EdgeRef(user, "likes", posts[0]),
		xdb.EdgeRef(user, "likes", posts[1]),
		xdb.EdgeRef(user, "likes", posts[2]),
	)

	// or by just specifying the edge name, which will fetch
	// all edges with that name
	refs = append(refs,
		xdb.EdgeRef(posts[0], "tags"),
		xdb.EdgeRef(posts[1], "tags"),
		xdb.EdgeRef(posts[2], "tags"),
	)

	edges, err := store.GetTuples(ctx, refs...)
	if err != nil {
		panic(err)
	}

	for _, edge := range edges {
		log.Printf("%s: %s", edge.Ref(), edge.Value())
	}
}
