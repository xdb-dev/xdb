package core

// Schema is a definition of a record's schema.
type Schema struct {
	Type        string             `json:"type"`
	Description string             `json:"description"`
	Record      *Schema            `json:"record"`
	Properties  map[string]*Schema `json:"properties"`
	Items       *Schema            `json:"items"`
	Ref         string             `json:"ref"`
}
