package xdbjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/tests"
)

// runDocRoundTrip checks JSON document conversion through the shared
// importer tests. It compares decoded maps, with numbers as float64, so
// JSON formatting and key order do not affect equality.
func runDocRoundTrip(t *testing.T, def *schema.Def, id string, doc []byte) {
	t.Helper()

	decOpts := []Option{
		WithNS(def.URI.NS()),
		WithSchema(def.URI.Schema()),
		WithDef(def),
	}

	var want map[string]any
	require.NoError(t, json.Unmarshal(doc, &want))
	want["_id"] = id // the encoder emits _id

	tests.RunRoundTrip(t, tests.RoundTrip{
		Def:   def,
		Value: want,
		Marshal: func(v any) (*core.Record, error) {
			raw, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			return Marshal(raw, decOpts...)
		},
		Unmarshal: func(rec *core.Record, dst any) error {
			out, err := Unmarshal(rec)
			if err != nil {
				return err
			}
			var m map[string]any
			if err := json.Unmarshal(out, &m); err != nil {
				return err
			}
			// The store stamps these on write; they are not part of
			// the document that went in.
			delete(m, schema.FieldVersion)
			delete(m, schema.FieldUpdated)
			*(dst.(*map[string]any)) = m
			return nil
		},
	})
}

func TestRoundTrip_Product(t *testing.T) {
	def, err := ImportSchema(readFixture(t, "product.schema.json"), WithNS("com.acme"))
	require.NoError(t, err)

	doc := []byte(`{
		"productId": 1,
		"productName": "Widget",
		"price": 9.99,
		"tags": ["red", "round"],
		"dimensions": {"length": 1.5, "width": 2.0, "height": 3.25}
	}`)

	runDocRoundTrip(t, def, "p1", doc)
}

func TestRoundTrip_Event(t *testing.T) {
	def, err := ImportSchema(readFixture(t, "event.schema.json"), WithNS("com.acme"))
	require.NoError(t, err)

	doc := []byte(`{
		"title": "Launch",
		"startsAt": "2021-01-01T08:00:00Z",
		"priority": "high",
		"organizer": {"name": "Ann", "email": "ann@example.com"},
		"attendees": [
			{"name": "Bob", "rsvpAt": "2021-01-02T09:00:00Z", "guests": 2},
			{"name": "Cara"}
		]
	}`)

	runDocRoundTrip(t, def, "e1", doc)
}

func TestRoundTrip_FlexibleUndeclared(t *testing.T) {
	// Product is flexible (no additionalProperties), so an undeclared field
	// round-trips as an inferred value.
	def, err := ImportSchema(readFixture(t, "product.schema.json"), WithNS("com.acme"))
	require.NoError(t, err)

	doc := []byte(`{
		"productId": 7,
		"productName": "Extra",
		"notes": "handle with care",
		"inStock": true
	}`)

	runDocRoundTrip(t, def, "p7", doc)
}
