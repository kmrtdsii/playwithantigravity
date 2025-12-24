package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/util"
)

// ListFiles returns a list of files in the worktree
func (sm *SessionManager) ListFiles(sessionID string) (string, error) {
	session, ok := sm.GetSession(sessionID)
	if !ok {
		return "", fmt.Errorf("session not found")
	}
	// Original code returned err if session not found? GetSession returns bool.
	// We should probably return error if not found.
	// But let's check GetSession signature in session.go.
	// func (sm *SessionManager) GetSession(id string) (*Session, bool)

	session.mu.RLock()
	defer session.mu.RUnlock()

	var files []string
	util.Walk(session.Filesystem, "/", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			if path == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		// Clean path
		if path != "" && path[0] == '/' {
			path = path[1:]
		}
		files = append(files, path)
		return nil
	})

	if len(files) == 0 {
		return "", nil
	}
	return strings.Join(files, "\n"), nil
}

// TouchFile updates the modification time and appends content to a file
func (sm *SessionManager) TouchFile(sessionID, filename string) error {
	session, ok := sm.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if file exists
	_, err := session.Filesystem.Stat(filename)
	if err != nil {
		// File likely doesn't exist, create it (empty)
		f, createErr := session.Filesystem.Create(filename)
		if createErr != nil {
			return createErr
		}
		f.Close()
		return nil
	}

	// File exists, append to it
	f, err := session.Filesystem.OpenFile(filename, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write([]byte("\n// Update")); err != nil {
		return err
	}

	return nil
}
