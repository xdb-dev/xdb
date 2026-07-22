package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/rpc"
)

// fakeDescriber is a minimal [api.MethodDescriber] for testing
// [api.IntrospectService] without a live [rpc.Router].
type fakeDescriber struct {
	meta map[string]rpc.MethodMeta
}

func newFakeDescriber() *fakeDescriber {
	return &fakeDescriber{
		meta: map[string]rpc.MethodMeta{
			"records.get": {Description: "Retrieve a record by URI."},
		},
	}
}

func (d *fakeDescriber) Methods() []string {
	names := make([]string, 0, len(d.meta))
	for name := range d.meta {
		names = append(names, name)
	}
	return names
}

func (d *fakeDescriber) Meta(method string) (rpc.MethodMeta, bool) {
	meta, ok := d.meta[method]
	return meta, ok
}

func newIntrospectService() *api.IntrospectService {
	return api.NewIntrospectService(newFakeDescriber())
}

func TestIntrospectService_DescribeMethod_Unknown(t *testing.T) {
	svc := newIntrospectService()

	_, err := svc.DescribeMethod(context.Background(), &api.DescribeMethodRequest{
		Method: "does.not.exist",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrNotFound)
	assert.Contains(t, err.Error(), "does.not.exist")
	assert.Contains(t, err.Error(), "introspect.methods")
}

func TestIntrospectService_DescribeMethod_Known(t *testing.T) {
	svc := newIntrospectService()

	resp, err := svc.DescribeMethod(context.Background(), &api.DescribeMethodRequest{
		Method: "records.get",
	})

	require.NoError(t, err)
	assert.Equal(t, "records.get", resp.Method)
}

func TestIntrospectService_DescribeType_Unknown(t *testing.T) {
	svc := newIntrospectService()

	_, err := svc.DescribeType(context.Background(), &api.DescribeTypeRequest{
		Type: "NotAType",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrNotFound)
	assert.Contains(t, err.Error(), "NotAType")
	assert.Contains(t, err.Error(), "introspect.types")
}

func TestIntrospectService_DescribeType_Known(t *testing.T) {
	svc := newIntrospectService()

	resp, err := svc.DescribeType(context.Background(), &api.DescribeTypeRequest{
		Type: "Value",
	})

	require.NoError(t, err)
	assert.Contains(t, resp.Description, "boolean")
}

func TestIntrospectService_ListTypes_IncludesFilterAndMode(t *testing.T) {
	svc := newIntrospectService()

	resp, err := svc.ListTypes(context.Background(), &api.ListTypesRequest{})
	require.NoError(t, err)

	names := make([]string, len(resp.Types))
	for i, ts := range resp.Types {
		names[i] = ts.Type
	}

	assert.Contains(t, names, "Filter")
	assert.Contains(t, names, "Mode")
}
