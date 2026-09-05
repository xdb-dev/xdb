package jsonschemaimport

import (
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/schema"
)

// Marshal decodes a JSON document into a [core.Record] addressed by uri. The
// uri must carry ns, schema, and id. If one of them is missing, Marshal
// panics. Declared fields are typed against def through encoding/xdbjson.
// Nested objects flatten to dotted attributes. Object arrays decode through
// their element schemas, so time.Time and integers inside elements are typed.
// Metadata fields (_id, _ns, _schema) in the document are ignored. The
// identity of the record comes from uri.
func Marshal(uri string, doc []byte, def *schema.Def) (*core.Record, error) {
	u, err := core.ParseURI(uri)
	if err != nil {
		return nil, err
	}

	rec := core.NewRecord(u.NS(), u.Schema(), u.ID())

	dec := xdbjson.NewDecoder(
		xdbjson.WithDef(def),
		xdbjson.WithNumberInference(),
	)
	if err := dec.ToExistingRecord(doc, rec); err != nil {
		return nil, err
	}

	return rec, nil
}

// Unmarshal encodes a record back into a JSON document. Dotted attributes
// unflatten into nested objects and object arrays render as arrays of nested
// objects. The record ID is emitted as the "_id" field.
func Unmarshal(rec *core.Record) ([]byte, error) {
	return xdbjson.New().FromRecord(rec)
}
