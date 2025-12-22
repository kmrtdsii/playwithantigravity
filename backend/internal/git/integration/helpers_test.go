package integration_test

import (
	"context"
	"fmt"

	"github.com/kurobon/gitgym/backend/internal/git"
	_ "github.com/kurobon/gitgym/backend/internal/git/commands"
)

var testSessionManager *git.SessionManager

func init() {
	testSessionManager = git.NewSessionManager()
}

func InitSession(id string) error {
	_, err := testSessionManager.CreateSession(id)
	return err
}

func ExecuteGitCommand(sessionID string, args []string) (string, error) {
	session, err := testSessionManager.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmdName := args[0]
	cmdArgs := args

	return git.Dispatch(context.Background(), session, cmdName, cmdArgs)
}

func GetGraphState(sessionID string, showAll bool) (*git.GraphState, error) {
	return testSessionManager.GetGraphState(sessionID, showAll)
}

func TouchFile(sessionID, filename string) error {
	return testSessionManager.TouchFile(sessionID, filename)
}

func ListFiles(sessionID string) (string, error) {
	return testSessionManager.ListFiles(sessionID)
}

func GetSession(id string) (*git.Session, error) {
	return testSessionManager.GetSession(id)
}
