package schema

import (
	"encoding/json"
	"os"
)

type Schema struct {
	ID   string                 `json:"id"`
	Defs map[string]*TypeSchema `json:"defs"`
}

func ReadSchema(path string) (*Schema, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var schema Schema
	if err := json.NewDecoder(file).Decode(&schema); err != nil {
		return nil, err
	}

	return &schema, nil
}
