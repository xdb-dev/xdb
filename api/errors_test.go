package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultErrorHandlerStatus(t *testing.T) {
	syntaxErr := &json.SyntaxError{Offset: 3}

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"syntax error", syntaxErr, http.StatusBadRequest},
		{"wrapped syntax error", fmt.Errorf("decode body: %w", syntaxErr), http.StatusBadRequest},
		{"unmarshal type error", &json.UnmarshalTypeError{Value: "string", Type: reflect.TypeFor[int]()}, http.StatusBadRequest},
		{"other error", fmt.Errorf("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			h := &defaultErrorHandler{}
			h.HandleError(rec, tt.err)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}
