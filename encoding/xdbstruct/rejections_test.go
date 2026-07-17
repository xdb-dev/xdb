package xdbstruct_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/encoding/xdbstruct"
)

type BadMap struct {
	Meta map[string]int `xdb:"meta"`
}

type BadIface struct {
	Payload any `xdb:"payload"`
}

type BadChan struct {
	Signal chan int `xdb:"signal"`
}

type BadFunc struct {
	Handler func() `xdb:"handler"`
}

// Node is directly recursive (Node → Node).
type Node struct {
	Label string `xdb:"label"`
	Next  *Node  `xdb:"next"`
}

// Manager and Employee form an indirect cycle (Employee → Manager → Employee).
type Manager struct {
	Reports []Employee `xdb:"reports"`
}

type Employee struct {
	Name string   `xdb:"name"`
	Boss *Manager `xdb:"boss"`
}

// JSONNode escapes the cycle via the json opt-in.
type JSONNode struct {
	Label string    `xdb:"label"`
	Next  *JSONNode `xdb:"next,json"`
}

func TestRejections(t *testing.T) {
	t.Run("map without json opt-in", func(t *testing.T) {
		_, err := xdbstruct.Def[BadMap]("xdb://com.example/bad")
		require.ErrorIs(t, err, xdbstruct.ErrUnsupported)
		assert.Contains(t, err.Error(), "meta")
		assert.Contains(t, err.Error(), "json")
	})

	t.Run("interface without json opt-in", func(t *testing.T) {
		_, err := xdbstruct.Def[BadIface]("xdb://com.example/bad")
		require.ErrorIs(t, err, xdbstruct.ErrUnsupported)
		assert.Contains(t, err.Error(), "payload")
		assert.Contains(t, err.Error(), "json")
	})

	t.Run("channel", func(t *testing.T) {
		_, err := xdbstruct.Def[BadChan]("xdb://com.example/bad")
		require.ErrorIs(t, err, xdbstruct.ErrUnsupported)
		assert.Contains(t, err.Error(), "signal")
	})

	t.Run("function", func(t *testing.T) {
		_, err := xdbstruct.Def[BadFunc]("xdb://com.example/bad")
		require.ErrorIs(t, err, xdbstruct.ErrUnsupported)
		assert.Contains(t, err.Error(), "handler")
	})

	t.Run("direct recursion", func(t *testing.T) {
		_, err := xdbstruct.Def[Node]("xdb://com.example/bad")
		require.ErrorIs(t, err, xdbstruct.ErrRecursive)
		assert.Contains(t, err.Error(), "next")
		assert.Contains(t, err.Error(), "Node → Node")
	})

	t.Run("indirect recursion", func(t *testing.T) {
		_, err := xdbstruct.Def[Employee]("xdb://com.example/bad")
		require.ErrorIs(t, err, xdbstruct.ErrRecursive)
		assert.Contains(t, err.Error(), "Employee → Manager → Employee")
	})

	t.Run("json opt-in escapes recursion", func(t *testing.T) {
		def, err := xdbstruct.Def[JSONNode]("xdb://com.example/ok")
		require.NoError(t, err)
		assert.Equal(t, "JSON", def.Fields["next"].Type.ID().String())
	})
}
