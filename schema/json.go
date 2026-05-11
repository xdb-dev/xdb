package schema

import (
	"encoding/json"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

type jsonFieldDef struct {
	Type     string `json:"type"`
	ElemType string `json:"elem_type,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type jsonDef struct {
	Fields map[string]jsonFieldDef `json:"fields,omitempty"`
	URI    string                  `json:"uri"`
	Mode   string                  `json:"mode"`
}

// MarshalJSON implements the [json.Marshaler] interface.
func (d *Def) MarshalJSON() ([]byte, error) {
	mode := d.Mode
	if mode == "" {
		mode = ModeStrict
	}

	jd := jsonDef{
		URI:  d.URI.String(),
		Mode: string(mode),
	}

	if len(d.Fields) > 0 {
		jd.Fields = make(map[string]jsonFieldDef, len(d.Fields))
		for name, field := range d.Fields {
			jf := jsonFieldDef{
				Type:     field.Type.Lower(),
				Required: field.Required,
			}
			if field.Type == core.TIDArray && field.ElemType != "" {
				jf.ElemType = field.ElemType.Lower()
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

	if len(jd.Fields) > 0 {
		d.Fields = make(map[string]FieldDef, len(jd.Fields))
		for name, jf := range jd.Fields {
			tid, err := core.ParseType(jf.Type)
			if err != nil {
				return err
			}
			field := FieldDef{
				Type:     tid,
				Required: jf.Required,
			}
			if tid == core.TIDArray && jf.ElemType != "" {
				elemTID, err := core.ParseType(jf.ElemType)
				if err != nil {
					return err
				}
				field.ElemType = elemTID
			}
			d.Fields[name] = field
		}
	}

	return nil
}
