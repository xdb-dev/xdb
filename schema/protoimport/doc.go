// Package protoimport imports protobuf message descriptors into XDB schemas and
// marshals proto messages to and from [core.Record] values.
//
// It works entirely through protoreflect on descriptors — no generated code and
// no protoc. [ImportFiles] and [ImportMessage] walk a [protoreflect.FileDescriptor]
// or [protoreflect.MessageDescriptor] into a [schema.Def]; [Marshal] and
// [Unmarshal] move data using dynamicpb on the read side.
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
// Every field records its proto field number in Annotations["proto.number"];
// [CheckRename] uses those numbers to detect a rename on re-import.
//
// # Obtaining descriptors without protoc
//
// Callers build descriptors either by constructing
// [google.golang.org/protobuf/types/descriptorpb.FileDescriptorProto] literals
// and passing them through
// [google.golang.org/protobuf/reflect/protodesc.NewFile], or by reusing the
// already-compiled descriptors of the well-known types (timestamppb, durationpb,
// wrapperspb, structpb) that are linked into the protobuf module. The tests use
// both approaches; neither invokes protoc.
package protoimport
