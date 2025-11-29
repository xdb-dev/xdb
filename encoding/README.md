# XDB Encodings

Encodings convert between XDB types and external formats.

## Packages

Each encoding lives in its own subdirectory:

```
encoding/
├── xdbjson/       # JSON encoding/decoding
├── xdbproto/      # Protocol Buffer support
├── xdbstruct/     # Go struct ↔ XDB record conversion
└── wkt/           # Well-known types registry
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

## wkt

Provides a registry system for well-known types (custom types with special handling during encoding/decoding).

### Basic Usage

```go
import "github.com/xdb-dev/xdb/encoding/wkt"

// Use the default registry with built-in types (time.Time, time.Duration)
encoder := xdbstruct.NewEncoder(xdbstruct.Options{
    Tag:      "xdb",
    Registry: wkt.DefaultRegistry,
})
```

### Custom Type Registration

```go
type Money struct {
    Amount   int64
    Currency string
}

registry := wkt.NewRegistry()
registry.Register(
    reflect.TypeOf(Money{}),
    func(v reflect.Value) (any, error) {
        m := v.Interface().(Money)
        return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
    },
    func(v any, target reflect.Value) error {
        // Parse string back to Money
        return nil
    },
)
```

### Built-in Types

The `DefaultRegistry` includes:
- `time.Time` (marshaled as RFC3339Nano strings)
- `time.Duration` (marshaled as int64 nanoseconds)

## Implementation Guide

When creating new encodings:

- Handle all XDB types (tuples, records, values, URIs)
- Preserve type information
- Follow the pattern of existing encoders
- Add comprehensive tests
