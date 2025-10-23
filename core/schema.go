package core

// Schema is a definition of a record's schema.
type Schema struct {
	Name        string             `json:"name"`
	Type        string             `json:"type"`
	Description string             `json:"description"`
	Fields      map[string]*Schema `json:"fields"`
	Items       *Schema            `json:"items"`
}
