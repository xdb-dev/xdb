# Schema Validation

**Date:** 2026-03-10
**Status:** Draft
**Reference:** `xdbx/xdb/schema/` (validate.go, def.go)

## Goal

Add validation logic to the `schema` package so that tuples and records can be checked against a `schema.Def` before being persisted. Also add field inference for dynamic schemas.

## Background

The current `schema` package defines `Def`, `FieldDef`, and `Mode` but has no validation logic. The reference implementation in `xdbx` provides tuple-level validation and field inference that we should adapt to our codebase.

## Current vs Target State

### Current `schema.Def`

```go
type Def struct {
    URI    *core.URI
    Fields map[string]FieldDef
    Mode   Mode  // ModeOpen, ModeStrict (int)
}

type FieldDef struct {
    Type     core.TID
    Required bool
}
```

### Target `schema.Def`

Align with reference — richer types, named fields, three modes:

```go
type Mode string

const (
    ModeFlexible Mode = "flexible"  // no validation, arbitrary attributes
    ModeStrict   Mode = "strict"    // only declared fields allowed, types checked
    ModeDynamic  Mode = "dynamic"   // like flexible but auto-infers new fields
)

type Def struct {
    URI    *core.URI
    Fields []*FieldDef  // slice, not map — order matters for display/serialization
    Mode   Mode
}

type FieldDef struct {
    Name string
    Type core.Type  // full Type (supports arrays with element types)
}
```

**Key changes:**
- `Mode` becomes a string enum with three values.
- `Fields` changes from `map[string]FieldDef` to `[]*FieldDef`.
- `FieldDef.Type` changes from `core.TID` to `core.Type`.
- `FieldDef` gains a `Name` field.
- `Required` field is dropped (not in reference; can be added later).

### New methods on `Def`

```go
// GetField returns the field definition for the given name, or nil.
func (d *Def) GetField(name string) *FieldDef

// AddFields adds field definitions, skipping duplicates by name.
func (d *Def) AddFields(fields ...*FieldDef)
```

## Validation API

### Functions (in `schema/validate.go`)

```go
// ValidateTuples validates tuples against the schema definition.
// In ModeFlexible, returns nil (no validation).
// In ModeStrict/ModeDynamic, checks each tuple's attribute exists in the schema
// and that the value type matches the field type.
func ValidateTuples(def *Def, tuples []*core.Tuple) error

// ValidateRecords validates all tuples in the given records.
// Delegates to ValidateTuples after flattening records into tuples.
func ValidateRecords(def *Def, records []*core.Record) error

// InferFields returns new FieldDef entries for attributes not yet in the schema.
// Used by ModeDynamic to auto-discover fields from data.
func InferFields(def *Def, tuples []*core.Tuple) ([]*FieldDef, error)
```

### Error Sentinels (in `schema/validate.go`)

```go
var (
    ErrUnknownField = errors.New("[xdb/schema] unknown field")
    ErrTypeMismatch = errors.New("[xdb/schema] type mismatch")
)
```

Errors are wrapped with context using `errors.Wrap` (from `github.com/gojekfarm/xtools/errors` or equivalent). If we don't have that dependency, use `fmt.Errorf` with `%w`.

### Validation Rules

| Rule | Condition | Error |
|------|-----------|-------|
| Unknown field | Tuple attribute not in `Def.Fields` | `ErrUnknownField` |
| Type mismatch | Tuple value type ≠ field type | `ErrTypeMismatch` |

- **ModeFlexible**: Skip all validation, return nil immediately.
- **ModeStrict**: Apply both rules, return on first violation.
- **ModeDynamic**: Same as strict for existing fields; unknown fields handled by `InferFields` at the caller level (not inside `ValidateTuples`).

### InferFields Logic

1. Iterate tuples.
2. Skip if attribute already exists in `def.GetField(attr)`.
3. Skip if attribute already seen in this batch.
4. Create `FieldDef{Name: attr, Type: tuple.Value().Type()}`.
5. Return the collected new field definitions.

## Implementation Steps

### Step 1: Update `schema.go` types

- Change `Mode` from `int` to `string`, add `ModeFlexible`, `ModeStrict`, `ModeDynamic`.
- Change `Fields` from `map[string]FieldDef` to `[]*FieldDef`.
- Update `FieldDef` — add `Name`, change `Type` to `core.Type`, drop `Required`.
- Add `GetField(name)` and `AddFields(fields...)` methods on `Def`.

### Step 2: Add validation (`schema/validate.go`)

- Add `ErrUnknownField`, `ErrTypeMismatch` sentinels.
- Implement `ValidateTuples`.
- Implement `ValidateRecords`.
- Implement `InferFields`.

### Step 3: Tests (`schema/validate_test.go`)

Table-driven tests covering:
- `ValidateTuples` with strict mode — unknown field → `ErrUnknownField`.
- `ValidateTuples` with strict mode — type mismatch → `ErrTypeMismatch`.
- `ValidateTuples` with strict mode — valid tuples → no error.
- `ValidateTuples` with flexible mode — unknown field → no error.
- `ValidateRecords` — delegates correctly.
- `InferFields` — returns new fields, skips existing and duplicates.

## Dependencies

- Need `core.Type` (struct with `ID()`, `Equals()`, `String()`) — verify this exists in our codebase.
- Need `Record.Tuples()` to return `[]*Tuple` — verify this exists.
- Error wrapping: use whatever error wrapping pattern is established in the project.

## Out of Scope

- Wiring validation into the store layer (follow-up task).
- Required field validation (can be added later if needed).
- Schema evolution, versioning, migration.
- Nested/dot-path attribute validation.
- `Def.Clone()`, `FieldDef.Clone()`, `FieldDef.Equals()` — nice to have, add if needed.
