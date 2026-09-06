// Package xdbproto imports protobuf message descriptors into XDB schemas.
// It also marshals proto messages to and from [core.Record] values.
//
// The package works only through protoreflect on descriptors. It needs no
// generated code and no protoc. [ImportFiles] and [ImportMessage] walk a
// [protoreflect.FileDescriptor] or a [protoreflect.MessageDescriptor] into a
// [schema.Def]. [Marshal] and [Unmarshal] move data between a proto message
// and a record through the protoreflect API of the message. The caller
// supplies the message on both sides.
//
// # Type mapping
//
//	proto construct              XDB
//	-------------------------    ----------------------------------------
//	proto package                namespace (override with WithNamespace)
//	message                      schema (Def), one per top-level message
//	nested message (singular)    flattened to dotted attributes
//	repeated message             ARRAY<JSON> object array (Field.Items)
//	scalar (int32, string, …)    matching core type; width in proto.type
//	enum                         STRING (value name); proto.enum annotation
//	repeated scalar/enum         ARRAY<scalar>
//	Timestamp                    TIME
//	Duration / Struct / Any      JSON (proto.type annotation)
//	wrappers (Int32Value, …)     the wrapped scalar
//	map<K,V>                     JSON (proto.map annotation)
//	oneof                        import error (union; non-goal)
//	recursive message            import error, unless WithAllowJSON
//
// Every field records its proto field number in Annotations["proto.number"].
// [CheckRename] uses these numbers to detect a rename on re-import.
//
// # Descriptors without protoc
//
// Callers can build descriptors in two ways. The first way is to construct
// [google.golang.org/protobuf/types/descriptorpb.FileDescriptorProto]
// literals and pass them through
// [google.golang.org/protobuf/reflect/protodesc.NewFile]. The second way is
// to reuse the compiled descriptors of the well-known types (timestamppb,
// durationpb, wrapperspb, structpb) that are linked into the protobuf module.
// The tests use both ways. Neither way invokes protoc.
package xdbproto
