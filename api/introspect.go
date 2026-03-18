package api

import (
	"context"
	"fmt"
)

// MethodLister returns the names of all registered methods.
// Satisfied by [rpc.Router].
type MethodLister interface {
	Methods() []string
}

// IntrospectService provides API introspection operations.
type IntrospectService struct {
	lister MethodLister
}

// NewIntrospectService creates an [IntrospectService] backed by the given [MethodLister].
func NewIntrospectService(l MethodLister) *IntrospectService {
	return &IntrospectService{lister: l}
}

// DescribeMethodRequest is the request for introspection of a method.
type DescribeMethodRequest struct {
	Method string `json:"method"`
}

// DescribeMethodResponse describes a single API method.
type DescribeMethodResponse struct {
	Method  string `json:"method"`
	Summary string `json:"summary"`
}

// DescribeMethod describes a single API method.
func (s *IntrospectService) DescribeMethod(_ context.Context, _ *DescribeMethodRequest) (*DescribeMethodResponse, error) {
	return nil, fmt.Errorf("api: describe_method not implemented")
}

// DescribeTypeRequest is the request for introspection of a type.
type DescribeTypeRequest struct {
	Type string `json:"type"`
}

// DescribeTypeResponse describes a single API type.
type DescribeTypeResponse struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// DescribeType describes a single API type.
func (s *IntrospectService) DescribeType(_ context.Context, _ *DescribeTypeRequest) (*DescribeTypeResponse, error) {
	return nil, fmt.Errorf("api: describe_type not implemented")
}

// ListMethodsRequest is the request for listing all API methods.
type ListMethodsRequest struct{}

// ListMethodsResponse lists all available API methods.
type ListMethodsResponse struct {
	Methods []string `json:"methods"`
}

// ListMethods lists all registered API methods.
func (s *IntrospectService) ListMethods(_ context.Context, _ *ListMethodsRequest) (*ListMethodsResponse, error) {
	return nil, fmt.Errorf("api: list_methods not implemented")
}

// ListTypesRequest is the request for listing all API types.
type ListTypesRequest struct{}

// ListTypesResponse lists all available API types.
type ListTypesResponse struct {
	Types []string `json:"types"`
}

// ListTypes lists all available API types.
func (s *IntrospectService) ListTypes(_ context.Context, _ *ListTypesRequest) (*ListTypesResponse, error) {
	return nil, fmt.Errorf("api: list_types not implemented")
}
