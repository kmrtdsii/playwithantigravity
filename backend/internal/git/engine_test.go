package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input        string
		expectedName string
		expectedArgs []string
	}{
		{"git status", "status", []string{"status"}},
		{"git commit -m 'msg'", "commit", []string{"commit", "-m", "msg"}},
		{"git --version", "version", []string{"version"}},
		{"git -v", "version", []string{"version"}},
		{"git --help", "help", []string{"help"}},
		{"git -h", "help", []string{"help"}},
		{"--version", "version", []string{"version"}},
		{"git", "help", []string{"help"}},
		{"git ls", "ls", []string{"ls"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, args := ParseCommand(tt.input)
			assert.Equal(t, tt.expectedName, name)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}
