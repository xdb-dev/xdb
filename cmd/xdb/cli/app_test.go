package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckStdinConsumers(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		file    string
		args    []string
		wantErr bool
	}{
		{"no dashes", "xdb://x/y/z", "", []string{"xdb://x/y/z"}, false},
		{"single positional dash", "", "", []string{"xdb://x/y/z", "-"}, false},
		{"single --uri dash", "-", "", nil, false},
		{"single --file dash", "xdb://x/y/z", "-", []string{"xdb://x/y/z"}, false},
		{"two positional dashes", "", "", []string{"-", "-"}, true},
		{"--uri dash + positional dash", "-", "", []string{"-"}, true},
		{"--uri dash + --file dash", "-", "-", nil, true},
		{"--file dash + positional dash", "", "-", []string{"-"}, true},
		{"--uri + --file + positional (three)", "-", "-", []string{"-"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkStdinConsumers(tc.uri, tc.file, tc.args)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "at most one")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
