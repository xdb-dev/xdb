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
	Annotations map[string]string `json:"annotations,omitempty"`
	Type        string            `json:"type"`
	ElemType    string            `json:"elem_type,omitempty"`
	Description string            `json:"description,omitempty"`
	Required    bool              `json:"required,omitempty"`
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
		jd.Fields = make(map[string]jsonField, len(d.Fields))
		for name, field := range d.Fields {
			jf := jsonField{
				Type:        field.Type.ID().Lower(),
				Required:    field.Required,
				Description: field.Description,
				Annotations: field.Annotations,
			}
			if field.Type.ID() == core.TIDArray && field.Type.ElemTypeID() != "" {
				jf.ElemType = field.Type.ElemTypeID().Lower()
			}
			jd.Fields[name] = jf
		}
	}

	return json.Marshal(jd)
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
		d.Fields = make(map[string]Field, len(jd.Fields))
		for name, jf := range jd.Fields {
			t, err := parseFieldType(jf)
			if err != nil {
				return err
			}
			d.Fields[name] = Field{
				Type:        t,
				Required:    jf.Required,
				Description: jf.Description,
				Annotations: jf.Annotations,
			}
		}
	}

	return nil
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
