package output_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
)

func TestErrorEnvelope_Error(t *testing.T) {
	t.Run("with URI", func(t *testing.T) {
		e := &output.ErrorEnvelope{
			Code:    "NOT_FOUND",
			Message: "record not found",
			URI:     "xdb://ns/s/id",
		}
		assert.Equal(t, "NOT_FOUND: record not found (xdb://ns/s/id)", e.Error())
	})

	t.Run("without URI", func(t *testing.T) {
		e := &output.ErrorEnvelope{
			Code:    "INVALID_ARGUMENT",
			Message: "URI required",
		}
		assert.Equal(t, "INVALID_ARGUMENT: URI required", e.Error())
	})
}

func TestFormatError_Envelope(t *testing.T) {
	env := &output.ErrorEnvelope{
		Code:     "NOT_FOUND",
		Message:  "record not found",
		Resource: "records",
		Action:   "get",
		URI:      "xdb://ns/s/id",
		Hint:     "try: xdb records list xdb://ns/s",
	}

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatJSON).FormatError(&buf, env))

		s := buf.String()
		assert.Contains(t, s, "\"code\": \"NOT_FOUND\"")
		assert.Contains(t, s, "\"resource\": \"records\"")
		assert.Contains(t, s, "\"action\": \"get\"")
		assert.Contains(t, s, "\"uri\": \"xdb://ns/s/id\"")
		assert.Contains(t, s, "\"hint\": \"try: xdb records list xdb://ns/s\"")
	})

	t.Run("ndjson", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatNDJSON).FormatError(&buf, env))

		s := buf.String()
		assert.Equal(t, 1, bytes.Count([]byte(s), []byte("\n")))
		assert.Contains(t, s, "\"code\":\"NOT_FOUND\"")
		assert.Contains(t, s, "\"resource\":\"records\"")
		assert.Contains(t, s, "\"action\":\"get\"")
	})

	t.Run("yaml", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatYAML).FormatError(&buf, env))

		s := buf.String()
		assert.Contains(t, s, "code: NOT_FOUND")
		assert.Contains(t, s, "resource: records")
		assert.Contains(t, s, "action: get")
	})

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatTable).FormatError(&buf, env))

		s := buf.String()
		assert.Contains(t, s, "Error")
		assert.Contains(t, s, "NOT_FOUND")
		assert.Contains(t, s, "Resource")
		assert.Contains(t, s, "records")
		assert.Contains(t, s, "Hint")
	})
}

func TestFormatError_OmitsEmptyOptionalFields(t *testing.T) {
	env := &output.ErrorEnvelope{
		Code:    "INVALID_ARGUMENT",
		Message: "URI required",
	}

	t.Run("json omits empty fields", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatJSON).FormatError(&buf, env))

		s := buf.String()
		assert.NotContains(t, s, "\"resource\"")
		assert.NotContains(t, s, "\"action\"")
		assert.NotContains(t, s, "\"uri\"")
		assert.NotContains(t, s, "\"hint\"")
	})

	t.Run("table omits empty rows", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatTable).FormatError(&buf, env))

		lines := bytes.Split(bytes.TrimRight(buf.Bytes(), "\n"), []byte("\n"))
		labels := make([]string, 0, len(lines))
		for _, line := range lines {
			fields := bytes.Fields(line)
			if len(fields) > 0 {
				labels = append(labels, string(fields[0]))
			}
		}
		assert.ElementsMatch(t, []string{"Error", "Message"}, labels)
	})
}
