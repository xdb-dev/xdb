package app

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xdb-dev/xdb/core"
)

func TestJSONFormatter(t *testing.T) {
	t.Run("formats record as JSON", func(t *testing.T) {
		// Arrange
		data := map[string]any{
			"name": "Alice",
			"age":  30,
		}

		formatter := &JSONFormatter{}

		// Act
		output, err := formatter.Format(data)

		// Assert
		assert.NoError(t, err)
		assert.Contains(t, string(output), "name")
		assert.Contains(t, string(output), "Alice")
	})

	t.Run("returns correct content type", func(t *testing.T) {
		formatter := &JSONFormatter{}
		assert.Equal(t, "application/json", formatter.ContentType())
	})
}

func TestTableFormatter(t *testing.T) {
	t.Run("formats single record as table", func(t *testing.T) {
		// Arrange
		record := core.NewRecord("com.example", "users", "123")
		record.Set("name", "Alice")
		record.Set("age", 30)

		formatter := &TableFormatter{}

		// Act
		output, err := formatter.Format(record)

		// Assert
		assert.NoError(t, err)
		assert.Contains(t, string(output), "ATTRIBUTE")
		assert.Contains(t, string(output), "VALUE")
		assert.Contains(t, string(output), "name")
		assert.Contains(t, string(output), "Alice")
	})

	t.Run("formats multiple records as table", func(t *testing.T) {
		// Arrange
		records := []*core.Record{
			core.NewRecord("com.example", "users", "1"),
			core.NewRecord("com.example", "users", "2"),
		}
		records[0].Set("name", "Alice")
		records[1].Set("name", "Bob")

		formatter := &TableFormatter{}

		// Act
		output, err := formatter.Format(records)

		// Assert
		assert.NoError(t, err)
		assert.Contains(t, string(output), "URI")
		assert.Contains(t, string(output), "Alice")
		assert.Contains(t, string(output), "Bob")
	})

	t.Run("handles empty records list", func(t *testing.T) {
		formatter := &TableFormatter{}

		output, err := formatter.Format([]*core.Record{})

		assert.NoError(t, err)
		assert.Contains(t, string(output), "No records found")
	})

	t.Run("returns correct content type", func(t *testing.T) {
		formatter := &TableFormatter{}
		assert.Equal(t, "text/plain", formatter.ContentType())
	})
}

func TestYAMLFormatter(t *testing.T) {
	t.Run("formats data as YAML", func(t *testing.T) {
		// Arrange
		data := map[string]any{
			"name": "Alice",
			"age":  30,
		}

		formatter := &YAMLFormatter{}

		// Act
		output, err := formatter.Format(data)

		// Assert
		assert.NoError(t, err)
		assert.Contains(t, string(output), "name: Alice")
		assert.Contains(t, string(output), "age: 30")
	})

	t.Run("returns correct content type", func(t *testing.T) {
		formatter := &YAMLFormatter{}
		assert.Equal(t, "application/yaml", formatter.ContentType())
	})
}

func TestSelectDefaultFormat(t *testing.T) {
	t.Run("returns a valid format", func(t *testing.T) {
		format := SelectDefaultFormat()
		assert.NotEmpty(t, format)
		// Valid formats: json, table
		assert.True(t, format == FormatJSON || format == FormatTable)
	})
}

func TestOutputWriter(t *testing.T) {
	t.Run("writes formatted output", func(t *testing.T) {
		// Arrange
		var buf bytes.Buffer
		writer := NewOutputWriter(&buf, FormatJSON)
		data := map[string]string{"key": "value"}

		// Act
		err := writer.Write(data)

		// Assert
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "key")
		assert.Contains(t, buf.String(), "value")
	})

	t.Run("quiet mode suppresses output", func(t *testing.T) {
		// Arrange
		var buf bytes.Buffer
		writer := NewOutputWriter(&buf, FormatJSON)
		writer.SetQuiet(true)

		// Act
		err := writer.Write(map[string]string{"key": "value"})

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, buf.String())
	})

	t.Run("writes newline after output", func(t *testing.T) {
		// Arrange
		var buf bytes.Buffer
		writer := NewOutputWriter(&buf, FormatJSON)
		data := map[string]string{"key": "value"}

		// Act
		err := writer.Write(data)

		// Assert
		assert.NoError(t, err)
		assert.True(t, len(buf.String()) > 0)
		assert.Equal(t, "\n", buf.String()[len(buf.String())-1:])
	})
}
