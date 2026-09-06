package xdbproto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestImportMessage_Order(t *testing.T) {
	fd := shopFile(t)
	def, err := ImportMessage(messageByName(fd, "Order"))
	require.NoError(t, err)

	assert.Equal(t, "xdb://acme.shop/Order", def.URI.String())
	assert.Equal(t, schema.ModeStrict, def.Mode)
	assert.Equal(t, "proto", def.Annotations["source"])
	assert.Equal(t, "acme.shop.Order", def.Annotations["proto.message"])

	id := def.Fields["id"]
	assert.Equal(t, core.TIDString, id.Type.ID())
	assert.Equal(t, "string", id.Annotations["proto.type"])
	assert.Equal(t, "1", id.Annotations["proto.number"])

	createdAt := def.Fields["created_at"]
	assert.Equal(t, core.TIDTime, createdAt.Type.ID())
	assert.Equal(t, "google.protobuf.Timestamp", createdAt.Annotations["proto.type"])

	items := def.Fields["items"]
	assert.Equal(t, core.TIDArray, items.Type.ID())
	assert.Equal(t, core.TIDJSON, items.Type.ElemTypeID())
	assert.Equal(t, "acme.shop.LineItem", items.Annotations["proto.type"])
	assert.Equal(t, core.TIDString, items.Items["sku"].Type.ID())
	assert.Equal(t, core.TIDInteger, items.Items["quantity"].Type.ID())
	assert.Equal(t, "int32", items.Items["quantity"].Annotations["proto.type"])
	assert.Equal(t, core.TIDFloat, items.Items["price"].Type.ID())

	status := def.Fields["status"]
	assert.Equal(t, core.TIDString, status.Type.ID())
	assert.Equal(t, "acme.shop.Status", status.Annotations["proto.enum"])
	assert.Equal(t, "enum", status.Annotations["proto.type"])

	assert.Equal(t, core.TIDBytes, def.Fields["signature"].Type.ID())

	tags := def.Fields["tags"]
	assert.Equal(t, core.TIDArray, tags.Type.ID())
	assert.Equal(t, core.TIDString, tags.Type.ElemTypeID())
}

func TestImportMessage_WKTAndMap(t *testing.T) {
	fd := metaFile(t)
	def, err := ImportMessage(messageByName(fd, "Meta"))
	require.NoError(t, err)

	ttl := def.Fields["ttl"]
	assert.Equal(t, core.TIDJSON, ttl.Type.ID())
	assert.Equal(t, "google.protobuf.Duration", ttl.Annotations["proto.type"])

	labels := def.Fields["labels"]
	assert.Equal(t, core.TIDJSON, labels.Type.ID())
	assert.Equal(t, "map", labels.Annotations["proto.type"])
	assert.Equal(t, "map<string, string>", labels.Annotations["proto.map"])

	retries := def.Fields["retries"]
	assert.Equal(t, core.TIDInteger, retries.Type.ID())
	assert.Equal(t, "google.protobuf.Int32Value", retries.Annotations["proto.type"])
}

func TestImportFiles_AllTopLevelMessages(t *testing.T) {
	defs, err := ImportFiles([]protoreflect.FileDescriptor{shopFile(t)})
	require.NoError(t, err)
	require.Len(t, defs, 2) // LineItem and Order
}

func TestImportMessage_NamespaceOverride(t *testing.T) {
	fd := shopFile(t)
	def, err := ImportMessage(messageByName(fd, "Order"), WithNamespace("custom.ns"))
	require.NoError(t, err)
	assert.Equal(t, "xdb://custom.ns/Order", def.URI.String())
}
