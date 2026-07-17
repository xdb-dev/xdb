package protoimport

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	// Blank imports register the well-known-type file descriptors in
	// protoregistry.GlobalFiles so the fixtures below can depend on them
	// WITHOUT invoking protoc.
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	_ "google.golang.org/protobuf/types/known/wrapperspb"
)

// compileFile turns a FileDescriptorProto literal into a live FileDescriptor,
// resolving any well-known-type dependencies against the global registry.
func compileFile(t *testing.T, fdp *descriptorpb.FileDescriptorProto) protoreflect.FileDescriptor {
	t.Helper()
	fd, err := protodesc.NewFile(fdp, protoregistry.GlobalFiles)
	require.NoError(t, err)
	return fd
}

// --- FileDescriptorProto literal builders ---

func fs(s string) *string { return &s }
func fi(i int32) *int32   { return &i }

func ftype(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &t
}

func flabel(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &l
}

const (
	optional = descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	repeated = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
)

// field builds a scalar/enum/message field descriptor proto.
func field(name string, num int32, t descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:   fs(name),
		Number: fi(num),
		Type:   ftype(t),
		Label:  flabel(label),
	}
}

// ref builds a message/enum-typed field referencing typeName.
func ref(name string, num int32, t descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label, typeName string) *descriptorpb.FieldDescriptorProto {
	f := field(name, num, t, label)
	f.TypeName = fs(typeName)
	return f
}

const (
	tString  = descriptorpb.FieldDescriptorProto_TYPE_STRING
	tInt32   = descriptorpb.FieldDescriptorProto_TYPE_INT32
	tInt64   = descriptorpb.FieldDescriptorProto_TYPE_INT64
	tUint32  = descriptorpb.FieldDescriptorProto_TYPE_UINT32
	tDouble  = descriptorpb.FieldDescriptorProto_TYPE_DOUBLE
	tBool    = descriptorpb.FieldDescriptorProto_TYPE_BOOL
	tBytes   = descriptorpb.FieldDescriptorProto_TYPE_BYTES
	tMessage = descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	tEnum    = descriptorpb.FieldDescriptorProto_TYPE_ENUM
)

// shopFile is a googleapis-style order file: scalars, an enum, a Timestamp, a
// repeated nested message, a repeated scalar, and bytes.
func shopFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:       fs("acme/shop/shop.proto"),
		Package:    fs("acme.shop"),
		Syntax:     fs("proto3"),
		Dependency: []string{"google/protobuf/timestamp.proto"},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: fs("Status"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: fs("STATUS_UNKNOWN"), Number: fi(0)},
					{Name: fs("PENDING"), Number: fi(1)},
					{Name: fs("SHIPPED"), Number: fi(2)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: fs("LineItem"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("sku", 1, tString, optional),
					field("quantity", 2, tInt32, optional),
					field("price", 3, tDouble, optional),
				},
			},
			{
				Name: fs("Order"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("id", 1, tString, optional),
					ref("created_at", 2, tMessage, optional, ".google.protobuf.Timestamp"),
					ref("items", 3, tMessage, repeated, ".acme.shop.LineItem"),
					ref("status", 4, tEnum, optional, ".acme.shop.Status"),
					field("signature", 5, tBytes, optional),
					field("tags", 6, tString, repeated),
				},
			},
		},
	}
	return compileFile(t, fdp)
}

// metaFile exercises the JSON-leaf and wrapper paths: a Duration, a map, and a
// wrapper scalar.
func metaFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    fs("acme/meta/meta.proto"),
		Package: fs("acme.meta"),
		Syntax:  fs("proto3"),
		Dependency: []string{
			"google/protobuf/duration.proto",
			"google/protobuf/wrappers.proto",
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: fs("Meta"),
				Field: []*descriptorpb.FieldDescriptorProto{
					ref("ttl", 1, tMessage, optional, ".google.protobuf.Duration"),
					ref("labels", 2, tMessage, repeated, ".acme.meta.Meta.LabelsEntry"),
					ref("retries", 3, tMessage, optional, ".google.protobuf.Int32Value"),
				},
				NestedType: []*descriptorpb.DescriptorProto{
					mapEntry("LabelsEntry", tString, tString, ""),
				},
			},
		},
	}
	return compileFile(t, fdp)
}

// userFile exercises single nested-message flattening to dotted attributes.
func userFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    fs("acme/user/user.proto"),
		Package: fs("acme.user"),
		Syntax:  fs("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: fs("Address"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("city", 1, tString, optional),
					field("zip", 2, tString, optional),
				},
			},
			{
				Name: fs("User"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("name", 1, tString, optional),
					ref("address", 2, tMessage, optional, ".acme.user.Address"),
				},
			},
		},
	}
	return compileFile(t, fdp)
}

// mapEntry builds the synthetic map-entry message descriptor for a map field.
func mapEntry(name string, keyT, valT descriptorpb.FieldDescriptorProto_Type, valTypeName string) *descriptorpb.DescriptorProto {
	val := field("value", 2, valT, optional)
	if valTypeName != "" {
		val.TypeName = fs(valTypeName)
	}
	yes := true
	return &descriptorpb.DescriptorProto{
		Name: fs(name),
		Field: []*descriptorpb.FieldDescriptorProto{
			field("key", 1, keyT, optional),
			val,
		},
		Options: &descriptorpb.MessageOptions{MapEntry: &yes},
	}
}

func messageByName(fd protoreflect.FileDescriptor, name string) protoreflect.MessageDescriptor {
	return fd.Messages().ByName(protoreflect.Name(name))
}
