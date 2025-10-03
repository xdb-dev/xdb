package api

type Tuple struct {
	ID    []string `json:"id"`
	Attr  []string `json:"attr"`
	Value any      `json:"value"`
}
