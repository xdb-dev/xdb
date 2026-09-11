---
title: Import your types
description: Create a schema from a Go struct, a protobuf message, or a JSON Schema file, and find drift in CI.
package: encoding/xdbstruct, encoding/xdbproto, encoding/xdbjson
read_when:
  - Your types exist as Go structs, .proto files, or JSON Schema
  - You want CI to fail when a schema file and the stored schema differ
---

# Import your types

XDB creates a schema from a tagged Go struct, a protobuf message, or a JSON Schema document. The same packages convert data between these types and records.

<figure class="frame">
  <svg id="fig-byot" role="img" aria-label="A proto file, a JSON Schema and a Go struct all import into one schema definition, which the store keeps. A dashed loop shows schemas diff checking for drift."></svg>
  <figcaption>bring your own types</figcaption>
</figure>

## From a Go struct

Tag each field with `xdb`. A nested struct becomes dotted attributes, and a slice becomes an array.

```go
type Author struct {
	Name string `xdb:"name"`
}

type Post struct {
	Title  string   `xdb:"title,required"`
	Author Author   `xdb:"author"` // the attribute author.name
	Tags   []string `xdb:"tags"`   // an array of string
	Views  int64    `xdb:"views"`
}
```

Create the schema from the struct:

```go
def, err := xdbstruct.Def[Post]("xdb://com.example/posts")
if err != nil {
	return err
}
if err := st.CreateSchema(ctx, def.URI, def); err != nil {
	return err
}
```

Store a `Post` and read it back:

```go
rec, err := xdbstruct.Marshal("xdb://com.example/posts/p-1", post)
if err != nil {
	return err
}
if err := st.CreateRecord(ctx, rec); err != nil {
	return err
}

rec, err = st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/p-1"))
if err != nil {
	return err
}

var got Post
err = xdbstruct.Unmarshal(rec, &got)
```

## From protobuf or JSON Schema

`xdb schemas import` reads a `.proto` file or a JSON Schema file:

```bash
xdb schemas import ./api/user.proto --ns com.example --dry-run
xdb schemas import ./api/user.proto --ns com.example
xdb schemas import ./user.schema.json --ns com.example
```

`--dry-run` shows the change to the stored schema and writes nothing. If the file renames a field, give `--rename old:new`. `--yes` skips the confirmation and accepts a suspected rename as a drop.

The same importers are Go functions. `doc` is a JSON Schema document as bytes, and `md` is a `protoreflect.MessageDescriptor`:

```go
def, err := xdbjson.ImportSchema(doc, xdbjson.WithNS("com.example"))

def, err := xdbproto.ImportMessage(md, xdbproto.WithNamespace("com.example"))
```

## Find drift in CI

`xdb schemas diff` compares a schema file with the stored schema. With `--check`, it exits with a code that is not 0 when they differ:

```bash
xdb schemas diff ./api/user.proto --ns com.example --check
```

Add this command to CI. Then a build fails when the file and the stored schema differ.

[Bring your own types](../concepts/bring-your-own-types.md) lists the type mappings and the constructs that XDB cannot import.
