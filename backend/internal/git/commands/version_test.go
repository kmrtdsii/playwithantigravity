package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	cmd := &VersionCommand{}
	output, err := cmd.Execute(context.TODO(), &git.Session{}, []string{"version"})
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(output, "git version"), "Output should start with 'git version'")
	assert.Contains(t, output, "2.51.2")
}
