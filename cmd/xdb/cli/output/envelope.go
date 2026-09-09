package output

import "fmt"

// ErrorEnvelope is the structured error shape rendered by the CLI.
// Every error in every output format has this same shape.
//
// Details carries the server's error tags — field, pointer, expected, got,
// and the rest of the vocabulary in the core package doc. It is what an
// agent should read; Message is prose and its wording is not a contract.
// The fix tag is lifted out into Hint rather than repeated in Details.
type ErrorEnvelope struct {
	Details  map[string]string `json:"details,omitempty" yaml:"details,omitempty"`
	Code     string            `json:"code" yaml:"code"`
	Message  string            `json:"message" yaml:"message"`
	Resource string            `json:"resource,omitempty" yaml:"resource,omitempty"`
	Action   string            `json:"action,omitempty" yaml:"action,omitempty"`
	URI      string            `json:"uri,omitempty" yaml:"uri,omitempty"`
	Hint     string            `json:"hint,omitempty" yaml:"hint,omitempty"`
}

// Error implements the error interface with a human-readable one-line form.
func (e *ErrorEnvelope) Error() string {
	if e == nil {
		return ""
	}

	if e.URI != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.URI)
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
