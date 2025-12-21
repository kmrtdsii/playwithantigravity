package git

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/util"
)

// ListFiles returns a list of files in the worktree
func (sm *SessionManager) ListFiles(sessionID string) (string, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return "", err
	}

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

// TouchFile updates the modification time and appends content to a file to ensure it's treated as modified
func (sm *SessionManager) TouchFile(sessionID, filename string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if file exists
	_, err = session.Filesystem.Stat(filename)
	if err != nil {
		// File likely doesn't exist, create it (empty)
		f, err := session.Filesystem.Create(filename)
		if err != nil {
			return err
		}
		f.Close()
		return nil
	}

	// File exists, append to it to update hash/modification
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
