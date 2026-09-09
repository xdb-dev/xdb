package api

import (
	"context"
	"fmt"
	"slices"

	"github.com/xdb-dev/xdb/api/catalog"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/rpc"
)

// MethodDescriber provides method names and metadata.
// Satisfied by [rpc.Router].
type MethodDescriber interface {
	Methods() []string
	Meta(method string) (rpc.MethodMeta, bool)
}

// IntrospectService provides API introspection operations.
type IntrospectService struct {
	describer MethodDescriber
}

// NewIntrospectService creates an [IntrospectService] backed by the given [MethodDescriber].
func NewIntrospectService(d MethodDescriber) *IntrospectService {
	return &IntrospectService{describer: d}
}

// DescribeMethodRequest is the request for introspection of a method.
type DescribeMethodRequest struct {
	Method string `json:"method"`
}

// DescribeMethodResponse describes a single API method.
type DescribeMethodResponse struct {
	Parameters  map[string]rpc.ParamMeta `json:"parameters,omitempty"`
	Response    map[string]rpc.ParamMeta `json:"response,omitempty"`
	Method      string                   `json:"method"`
	Description string                   `json:"description"`
	Mutating    bool                     `json:"mutating,omitempty"`
}

// DescribeMethod describes a single API method.
func (s *IntrospectService) DescribeMethod(_ context.Context, req *DescribeMethodRequest) (*DescribeMethodResponse, error) {
	meta, ok := s.describer.Meta(req.Method)
	if !ok {
		return nil, fmt.Errorf("%w: unknown method %q; run introspect.methods to list available methods", core.ErrNotFound, req.Method)
	}

	return &DescribeMethodResponse{
		Method:      req.Method,
		Description: meta.Description,
		Parameters:  meta.Parameters,
		Response:    meta.Response,
		Mutating:    meta.Mutating,
	}, nil
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
func (s *IntrospectService) DescribeType(_ context.Context, req *DescribeTypeRequest) (*DescribeTypeResponse, error) {
	desc, ok := catalog.Types()[req.Type]
	if !ok {
		return nil, fmt.Errorf("%w: unknown type %q; run introspect.types to list available types", core.ErrNotFound, req.Type)
	}

	return &DescribeTypeResponse{
		Type:        req.Type,
		Description: desc,
	}, nil
}

// MethodSummary is a method name with its summary for list responses.
type MethodSummary struct {
	Method      string `json:"method"`
	Description string `json:"description"`
	Mutating    bool   `json:"mutating,omitempty"`
}

// ListMethodsRequest is the request for listing all API methods.
type ListMethodsRequest struct{}

// ListMethodsResponse lists all available API methods.
type ListMethodsResponse struct {
	Methods []MethodSummary `json:"methods"`
}

// ListMethods lists all registered API methods.
func (s *IntrospectService) ListMethods(_ context.Context, _ *ListMethodsRequest) (*ListMethodsResponse, error) {
	names := s.describer.Methods()
	slices.Sort(names)

	methods := make([]MethodSummary, len(names))
	for i, name := range names {
		meta, _ := s.describer.Meta(name)
		methods[i] = MethodSummary{
			Method:      name,
			Description: meta.Description,
			Mutating:    meta.Mutating,
		}
	}

	return &ListMethodsResponse{Methods: methods}, nil
}

// TypeSummary is a type name with its description for list responses.
type TypeSummary struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ListTypesRequest is the request for listing all API types.
type ListTypesRequest struct{}

// ListTypesResponse lists all available API types.
type ListTypesResponse struct {
	Types []TypeSummary `json:"types"`
}

// ListTypes lists all available API types.
func (s *IntrospectService) ListTypes(_ context.Context, _ *ListTypesRequest) (*ListTypesResponse, error) {
	typeDescriptions := catalog.Types()

	names := make([]string, 0, len(typeDescriptions))
	for name := range typeDescriptions {
		names = append(names, name)
	}

	slices.Sort(names)

	types := make([]TypeSummary, len(names))
	for i, name := range names {
		types[i] = TypeSummary{
			Type:        name,
			Description: typeDescriptions[name],
		}
	}

	return &ListTypesResponse{Types: types}, nil
}
