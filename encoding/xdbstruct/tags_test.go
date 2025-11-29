package xdbstruct

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		want     field
		wantName string
		wantSkip bool
		wantPK   bool
	}{
		{
			name:     "empty tag",
			tag:      "",
			wantName: "",
			wantSkip: false,
			wantPK:   false,
		},
		{
			name:     "simple field name",
			tag:      "email",
			wantName: "email",
			wantSkip: false,
			wantPK:   false,
		},
		{
			name:     "field name with spaces",
			tag:      " email ",
			wantName: "email",
			wantSkip: false,
			wantPK:   false,
		},
		{
			name:     "skip field",
			tag:      "-",
			wantName: "-",
			wantSkip: true,
			wantPK:   false,
		},
		{
			name:     "primary key",
			tag:      "id,primary_key",
			wantName: "id",
			wantSkip: false,
			wantPK:   true,
		},
		{
			name:     "primary key with spaces",
			tag:      "id , primary_key ",
			wantName: "id",
			wantSkip: false,
			wantPK:   true,
		},
		{
			name:     "multiple options",
			tag:      "id,primary_key,other",
			wantName: "id",
			wantSkip: false,
			wantPK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTag(tt.tag)
			assert.Equal(t, tt.wantName, got.name, "field name mismatch")
			assert.Equal(t, tt.wantSkip, got.skip, "skip flag mismatch")
			assert.Equal(t, tt.wantPK, got.primaryKey, "primary_key flag mismatch")
		})
	}
}
