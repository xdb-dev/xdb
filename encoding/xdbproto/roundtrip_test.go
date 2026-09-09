package xdbproto

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/storetest"
)

// runProtoRoundTrip imports md, builds a message, and drives it through the
// shared harness. The corpus value is the message's canonical serialized bytes:
// proto messages have no stable DeepEqual, but deterministic wire bytes do, so
// this compares semantically while satisfying require.Equal on []byte.
func runProtoRoundTrip(
	t *testing.T,
	md protoreflect.MessageDescriptor,
	uri string,
	build func(m protoreflect.Message),
	opts ...Option,
) {
	t.Helper()

	def, err := ImportMessage(md, opts...)
	require.NoError(t, err)

	orig := dynamicpb.NewMessage(md)
	build(orig)

	marshalOpt := proto.MarshalOptions{Deterministic: true}
	origBytes, err := marshalOpt.Marshal(orig)
	require.NoError(t, err)

	storetest.RunRoundTrip(t, storetest.RoundTrip{
		Def:   def,
		Value: origBytes,
		Marshal: func(v any) (*core.Record, error) {
			m := dynamicpb.NewMessage(md)
			if err := proto.Unmarshal(v.([]byte), m); err != nil {
				return nil, err
			}
			return Marshal(uri, m, opts...)
		},
		Unmarshal: func(rec *core.Record, dst any) error {
			m := dynamicpb.NewMessage(md)
			if err := Unmarshal(rec, m, opts...); err != nil {
				return err
			}
			b, err := marshalOpt.Marshal(m)
			if err != nil {
				return err
			}
			*(dst.(*[]byte)) = b
			return nil
		},
	})
}

func TestRoundTrip_Order(t *testing.T) {
	md := messageByName(shopFile(t), "Order")

	runProtoRoundTrip(t, md, "xdb://acme.shop/Order/o1", func(m protoreflect.Message) {
		f := m.Descriptor().Fields()
		m.Set(f.ByName("id"), protoreflect.ValueOfString("o1"))

		ts := m.Mutable(f.ByName("created_at")).Message()
		tf := ts.Descriptor().Fields()
		ts.Set(tf.ByName("seconds"), protoreflect.ValueOfInt64(1600000000))
		ts.Set(tf.ByName("nanos"), protoreflect.ValueOfInt32(123456789))

		items := m.Mutable(f.ByName("items")).List()
		for _, it := range []struct {
			sku   string
			qty   int32
			price float64
		}{{"apple", 3, 1.50}, {"pear", 1, 2.25}} {
			e := items.NewElement()
			ef := e.Message().Descriptor().Fields()
			e.Message().Set(ef.ByName("sku"), protoreflect.ValueOfString(it.sku))
			e.Message().Set(ef.ByName("quantity"), protoreflect.ValueOfInt32(it.qty))
			e.Message().Set(ef.ByName("price"), protoreflect.ValueOfFloat64(it.price))
			items.Append(e)
		}

		m.Set(f.ByName("status"), protoreflect.ValueOfEnum(1)) // PENDING
		m.Set(f.ByName("signature"), protoreflect.ValueOfBytes([]byte{0x01, 0x02, 0x03}))

		tags := m.Mutable(f.ByName("tags")).List()
		tags.Append(protoreflect.ValueOfString("x"))
		tags.Append(protoreflect.ValueOfString("y"))
	})
}

func TestRoundTrip_UserNested(t *testing.T) {
	md := messageByName(userFile(t), "User")

	runProtoRoundTrip(t, md, "xdb://acme.user/User/u1", func(m protoreflect.Message) {
		f := m.Descriptor().Fields()
		m.Set(f.ByName("name"), protoreflect.ValueOfString("bob"))
		addr := m.Mutable(f.ByName("address")).Message()
		af := addr.Descriptor().Fields()
		addr.Set(af.ByName("city"), protoreflect.ValueOfString("NYC"))
		addr.Set(af.ByName("zip"), protoreflect.ValueOfString("10001"))
	})
}

func TestRoundTrip_MetaJSONAndWrapper(t *testing.T) {
	md := messageByName(metaFile(t), "Meta")

	runProtoRoundTrip(t, md, "xdb://acme.meta/Meta/m1", func(m protoreflect.Message) {
		f := m.Descriptor().Fields()

		ttl := m.Mutable(f.ByName("ttl")).Message()
		ttl.Set(ttl.Descriptor().Fields().ByName("seconds"), protoreflect.ValueOfInt64(3600))

		labels := m.Mutable(f.ByName("labels")).Map()
		labels.Set(protoreflect.ValueOfString("env").MapKey(), protoreflect.ValueOfString("prod"))
		labels.Set(protoreflect.ValueOfString("tier").MapKey(), protoreflect.ValueOfString("gold"))

		retries := m.Mutable(f.ByName("retries")).Message()
		retries.Set(retries.Descriptor().Fields().ByName("value"), protoreflect.ValueOfInt32(5))
	})
}
