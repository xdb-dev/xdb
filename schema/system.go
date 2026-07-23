package schema

import (
	"strings"

	"github.com/xdb-dev/xdb/core"
)

// SystemPrefix marks names reserved for XDB system metadata. User
// schemas may not declare top-level fields starting with it, which keeps
// the namespace permanently free for [FieldID], [FieldVersion], and
// [FieldUpdated].
const SystemPrefix = "_"

// Reserved system field names, present on every record.
//
// [FieldID] is virtual: it is never stored as a tuple, because every
// backend already holds it as the record's addressing key (SQLite's _id
// column, xdbfs's filename, xdbredis's key suffix). It is projected from
// the record path on read.
//
// [FieldVersion] and [FieldUpdated] are stored fields, stamped into
// every definition and written by the versioning middleware.
const (
	// FieldID is the record's id, projected from its path.
	FieldID = "_id"

	// FieldVersion is the record's revision counter: 1 on create,
	// incremented on every successful write. Supplied on a write it
	// acts as an optimistic-concurrency precondition.
	FieldVersion = "_version"

	// FieldUpdated is the timestamp of the record's last write.
	FieldUpdated = "_updated"
)

// IsSystemField reports whether name is reserved for system metadata,
// i.e. begins with [SystemPrefix]. The whole name is judged, not its
// segments, so a dotted path like "profile._id" is a user field.
func IsSystemField(name string) bool {
	return strings.HasPrefix(name, SystemPrefix)
}

// storedSystemFields returns the system fields carried in a definition.
// [FieldID] is absent by design: it is projected from the record path,
// never stored, and declaring it would collide with the physical id
// column SQLite already uses.
func storedSystemFields() map[string]Field {
	return map[string]Field{
		FieldVersion: {
			Type:        core.TypeInt,
			Description: "Record revision, incremented on every write.",
		},
		FieldUpdated: {
			Type:        core.TypeTime,
			Description: "Timestamp of the record's last write.",
		},
	}
}

// StampSystemFields returns a copy of d carrying the system fields.
// Stamping is idempotent and never mutates d. Definitions are stamped on
// the way into storage, so every backend materializes the fields the way
// it materializes any other declared field (a SQLite column, a KV row).
func StampSystemFields(d *Def) *Def {
	stamped := d.clone()
	for name, field := range storedSystemFields() {
		stamped.Fields[name] = field
	}
	return stamped
}

// StripSystemFields returns a copy of d with every system field removed.
// It is the inverse of [StampSystemFields] and the way a stored
// definition is turned back into the user's own declaration — for
// [Def.Validate], which rejects reserved names, and for any surface that
// reports what the user declared.
func StripSystemFields(d *Def) *Def {
	stripped := d.clone()
	for name := range stripped.Fields {
		if IsSystemField(name) {
			delete(stripped.Fields, name)
		}
	}
	return stripped
}

// HasSystemFields reports whether d already carries every stored system
// field. A definition written before versioning existed does not, and is
// upgraded in place on next use.
func HasSystemFields(d *Def) bool {
	for name := range storedSystemFields() {
		if _, ok := d.Fields[name]; !ok {
			return false
		}
	}
	return true
}
