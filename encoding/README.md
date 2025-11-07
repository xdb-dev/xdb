# XDB Encodings

Encodings convert between XDB types and external formats.

## Packages

Each encoding lives in its own subdirectory:

```
encoding/
├── xdbjson/       # JSON encoding/decoding
├── xdbproto/      # Protocol Buffer support
└── xdbstruct/     # Go struct ↔ XDB record conversion
```

## xdbstruct

Converts Go structs to XDB records and vice versa.

### Struct Tags

Use `xdb` struct tags to map struct fields to record attributes:

```go
type User struct {
    ID       string `xdb:"id,primary_key"`
    Name     string `xdb:"name"`
    Email    string `xdb:"email"`
    Age      int    `xdb:"age"`
}
```

Tag format: `xdb:"field_name[,option1,option2,...]"`

Options:

- `primary_key`: Marks field as the record ID
- `-`: Skips the field

### Nested Structs

Nested structs are flattened using dot notation:

```go
type User struct {
    ID      string  `xdb:"id,primary_key"`
    Profile Profile `xdb:"profile"`
}

type Profile struct {
    Email string `xdb:"email"`
    Bio   string `xdb:"bio"`
}

// Results in attributes: profile.email, profile.bio
```

## Implementation Guide

When creating new encodings:

- Handle all XDB types (tuples, records, values, URIs)
- Preserve type information
- Follow the pattern of existing encoders
- Add comprehensive tests
