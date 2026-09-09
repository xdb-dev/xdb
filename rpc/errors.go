package rpc

import "fmt"

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// XDB-specific error codes.
const (
	CodeNotFound        = -32000
	CodeAlreadyExists   = -32001
	CodeSchemaViolation = -32002
	CodeConflict        = -32003
	CodeNotImplemented  = -32004
	CodeUniqueViolation = -32005
)

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// NewError creates a new [Error] with the given code and message.
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// ParseError creates a parse error.
func ParseError(msg string) *Error {
	return NewError(CodeParseError, msg)
}

// InvalidRequest creates an invalid request error.
func InvalidRequest(msg string) *Error {
	return NewError(CodeInvalidRequest, msg)
}

// MethodNotFound creates a method not found error.
func MethodNotFound(method string) *Error {
	return NewError(CodeMethodNotFound, fmt.Sprintf("method not found: %s", method))
}

// InvalidParams creates an invalid params error.
func InvalidParams(msg string) *Error {
	return NewError(CodeInvalidParams, msg)
}

// InvalidParamsData creates an invalid params error carrying structured data,
// e.g. {"reason": "invalid_uri"}.
func InvalidParamsData(msg string, data any) *Error {
	err := InvalidParams(msg)
	err.Data = data
	return err
}

// InternalError creates an internal error.
func InternalError(msg string) *Error {
	return NewError(CodeInternalError, msg)
}

// NotFound creates a not found error.
func NotFound(msg string) *Error {
	return NewError(CodeNotFound, msg)
}

// AlreadyExists creates an already exists error.
func AlreadyExists(msg string) *Error {
	return NewError(CodeAlreadyExists, msg)
}

// SchemaViolation creates a schema violation error.
func SchemaViolation(msg string) *Error {
	return NewError(CodeSchemaViolation, msg)
}

// Conflict creates a conflict error.
func Conflict(msg string) *Error {
	return NewError(CodeConflict, msg)
}

// UniqueViolation creates a unique constraint violation error. It is
// distinct from [Conflict]: a conflict is resolved by re-reading and
// retrying, while a unique violation needs a different value.
func UniqueViolation(msg string) *Error {
	return NewError(CodeUniqueViolation, msg)
}

// NotImplemented creates a not implemented error.
func NotImplemented(msg string) *Error {
	return NewError(CodeNotImplemented, msg)
}
