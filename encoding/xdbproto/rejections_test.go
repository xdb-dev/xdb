package xdbproto

import (
	"errors"
	"testing"

	xerrors "github.com/gojekfarm/xtools/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// oneofFile declares a message with a real (non-synthetic) oneof.
func oneofFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    fs("acme/one/one.proto"),
		Package: fs("acme.one"),
		Syntax:  fs("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:      fs("Payment"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: fs("method")}},
				Field: []*descriptorpb.FieldDescriptorProto{
					oneofField("card", 1, tString, 0),
					oneofField("cash", 2, tBool, 0),
				},
			},
		},
	}
	return compileFile(t, fdp)
}

func oneofField(name string, num int32, t descriptorpb.FieldDescriptorProto_Type, oneofIndex int32) *descriptorpb.FieldDescriptorProto {
	f := field(name, num, t, optional)
	f.OneofIndex = fi(oneofIndex)
	return f
}

func TestReject_Oneof(t *testing.T) {
	fd := oneofFile(t)
	_, err := ImportMessage(messageByName(fd, "Payment"))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOneof)
	assert.Contains(t, fieldOf(err), "card")
}

// cyclicFile declares a self-referential message: Node has a repeated Node.
func cyclicFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    fs("acme/tree/tree.proto"),
		Package: fs("acme.tree"),
		Syntax:  fs("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: fs("Node"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("label", 1, tString, optional),
					ref("children", 2, tMessage, repeated, ".acme.tree.Node"),
				},
			},
		},
	}
	return compileFile(t, fdp)
}

func TestReject_CyclicWithoutOptIn(t *testing.T) {
	fd := cyclicFile(t)
	_, err := ImportMessage(messageByName(fd, "Node"))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRecursive)
	assert.Contains(t, fieldOf(err), "children")
}

func TestCyclic_AllowJSONEscapeHatch(t *testing.T) {
	fd := cyclicFile(t)
	def, err := ImportMessage(
		messageByName(fd, "Node"),
		WithAllowJSON("acme.tree.Node"),
	)
	require.NoError(t, err)
	// children becomes an ARRAY of opaque JSON (repeated allow-json message).
	assert.Equal(t, schema.ModeStrict, def.Mode)
	assert.Equal(t, core.TIDString, def.Fields["label"].Type.ID())
	children := def.Fields["children"]
	assert.Equal(t, core.TIDArray, children.Type.ID())
	assert.Equal(t, core.TIDJSON, children.Type.ElemTypeID())
}

func TestReject_RenameOnReimport(t *testing.T) {
	v1 := messageByName(shopFile(t), "Order")
	def1, err := ImportMessage(v1)
	require.NoError(t, err)

	// v2 renames field number 1 from "id" to "order_id".
	fd2 := renamedShopFile(t)
	def2, err := ImportMessage(messageByName(fd2, "Order"))
	require.NoError(t, err)

	err = CheckRename(def1, def2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRename)
	assert.Equal(t, "id", errValue(err, "from"))
	assert.Equal(t, "order_id", errValue(err, "to"))
}

// renamedShopFile is shopFile with Order.id (number 1) renamed to order_id.
func renamedShopFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    fs("acme/shop/shop2.proto"),
		Package: fs("acme.shop"),
		Syntax:  fs("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: fs("Order"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("order_id", 1, tString, optional),
				},
			},
		},
	}
	return compileFile(t, fdp)
}

// fieldOf extracts the "field" attribute recorded on a wrapped error.
func fieldOf(err error) string {
	return errValue(err, "field")
}

// errValue extracts a keyed attribute from an xtools/errors-wrapped error.
func errValue(err error, key string) string {
	var e *xerrors.ErrorTags
	if errors.As(err, &e) {
		return e.All()[key]
	}
	return ""
}
