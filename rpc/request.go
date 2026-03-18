package rpc

import "encoding/json"

// Request is a JSON-RPC 2.0 request.
type Request struct {
	ID      string          `json:"id,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	Error   *Error          `json:"error,omitempty"`
	ID      string          `json:"id,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// NewResponse creates a success response for the given request ID.
func NewResponse(id string, result json.RawMessage) *Response {
	return &Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

// NewErrorResponse creates an error response for the given request ID.
func NewErrorResponse(id string, err *Error) *Response {
	return &Response{
		JSONRPC: "2.0",
		Error:   err,
		ID:      id,
	}
}
