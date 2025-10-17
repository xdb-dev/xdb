package schema

type TypeSchema struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Record      *TypeSchema            `json:"record"`
	Properties  map[string]*TypeSchema `json:"properties"`
	Items       *TypeSchema            `json:"items"`
	Ref         string                 `json:"ref"`
}
