package core

// Collection is a group of records with the same schema.
type Collection struct {
	Repo   string  `json:"repo"`
	ID     string  `json:"id"`
	Schema *Schema `json:"schema"`
}

func (c *Collection) URI() *URI {
	return &URI{repo: c.Repo, coll: c.ID}
}
