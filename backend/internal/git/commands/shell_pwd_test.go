package commands

import (
	"context"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestPwdCommand(t *testing.T) {
	cmd := &PwdCommand{}
	session := &git.Session{
		CurrentDir: "/my/path",
	}

	out, err := cmd.Execute(context.TODO(), session, []string{"pwd"})
	assert.NoError(t, err)
	assert.Equal(t, "/my/path", out)

	// Test default
	session.CurrentDir = ""
	out, err = cmd.Execute(context.TODO(), session, []string{"pwd"})
	assert.NoError(t, err)
	assert.Equal(t, "/", out)
}
