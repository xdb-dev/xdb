package output

import "fmt"

// ErrorEnvelope is the structured error shape rendered by the CLI.
// Every error in every output format has this same shape.
type ErrorEnvelope struct {
	Code     string `json:"code" yaml:"code"`
	Message  string `json:"message" yaml:"message"`
	Resource string `json:"resource,omitempty" yaml:"resource,omitempty"`
	Action   string `json:"action,omitempty" yaml:"action,omitempty"`
	URI      string `json:"uri,omitempty" yaml:"uri,omitempty"`
	Hint     string `json:"hint,omitempty" yaml:"hint,omitempty"`
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
