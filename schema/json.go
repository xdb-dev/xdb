package schema

import (
	"encoding/json"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

// jsonField is the wire representation of a [Field]. The Type is split into a
// scalar type name plus an optional element type (arrays only), preserving
// backward compatibility with schemas stored before core.Type existed.
type jsonField struct {
	Annotations map[string]string    `json:"annotations,omitempty"`
	Items       map[string]jsonField `json:"items,omitempty"`
	Type        string               `json:"type"`
	ElemType    string               `json:"elem_type,omitempty"`
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
}

type jsonDef struct {
	Fields      map[string]jsonField `json:"fields,omitempty"`
	Annotations map[string]string    `json:"annotations,omitempty"`
	URI         string               `json:"uri"`
	Mode        string               `json:"mode"`
	Description string               `json:"description,omitempty"`
	Revision    int64                `json:"revision,omitempty"`
}

// MarshalJSON implements the [json.Marshaler] interface.
func (d *Def) MarshalJSON() ([]byte, error) {
	mode := d.Mode
	if mode == "" {
		mode = ModeStrict
	}

	jd := jsonDef{
		URI:         d.URI.String(),
		Mode:        string(mode),
		Description: d.Description,
		Revision:    d.Revision,
		Annotations: d.Annotations,
	}

	if len(d.Fields) > 0 {
		jd.Fields = marshalFields(d.Fields)
	}

	return json.Marshal(jd)
}

// marshalFields converts a set of [Field] values to their wire form,
// recursing into the Items of object-array fields.
func marshalFields(fields map[string]Field) map[string]jsonField {
	out := make(map[string]jsonField, len(fields))
	for name, field := range fields {
		jf := jsonField{
			Type:        field.Type.ID().Lower(),
			Required:    field.Required,
			Description: field.Description,
			Annotations: field.Annotations,
		}
		if field.Type.ID() == core.TIDArray && field.Type.ElemTypeID() != "" {
			jf.ElemType = field.Type.ElemTypeID().Lower()
		}
		if len(field.Items) > 0 {
			jf.Items = marshalFields(field.Items)
		}
		out[name] = jf
	}
	return out
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (d *Def) UnmarshalJSON(data []byte) error {
	var jd jsonDef
	if err := json.Unmarshal(data, &jd); err != nil {
		return err
	}

	uri, err := core.ParseURI(jd.URI)
	if err != nil {
		return err
	}

	mode := Mode(jd.Mode)
	if mode == "" {
		mode = ModeStrict
	}
	if _, ok := validModes[mode]; !ok {
		return errors.Wrap(ErrInvalidMode, "mode", jd.Mode)
	}

	d.URI = uri
	d.Mode = mode
	d.Description = jd.Description
	d.Revision = jd.Revision
	d.Annotations = jd.Annotations

	if len(jd.Fields) > 0 {
		fields, err := unmarshalFields(jd.Fields)
		if err != nil {
			return err
		}
		d.Fields = fields
	}

	return nil
}

// unmarshalFields reconstructs a set of [Field] values from their wire form,
// recursing into the Items of object-array fields.
func unmarshalFields(jfs map[string]jsonField) (map[string]Field, error) {
	out := make(map[string]Field, len(jfs))
	for name, jf := range jfs {
		t, err := parseFieldType(jf)
		if err != nil {
			return nil, err
		}
		field := Field{
			Type:        t,
			Required:    jf.Required,
			Description: jf.Description,
			Annotations: jf.Annotations,
		}
		if len(jf.Items) > 0 {
			items, err := unmarshalFields(jf.Items)
			if err != nil {
				return nil, err
			}
			field.Items = items
		}
		out[name] = field
	}
	return out, nil
}

// parseFieldType reconstructs a [core.Type] from the wire representation,
// preserving the array element type when present.
func parseFieldType(jf jsonField) (core.Type, error) {
	tid, err := core.ParseType(jf.Type)
	if err != nil {
		return core.Type{}, err
	}

	if tid != core.TIDArray {
		return core.NewType(tid), nil
	}

	if jf.ElemType == "" {
		return core.NewArrayType(""), nil
	}

	elemTID, err := core.ParseType(jf.ElemType)
	if err != nil {
		return core.Type{}, err
	}
	return core.NewArrayType(elemTID), nil
}
