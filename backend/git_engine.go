package main

import (
	"fmt"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Session holds the state of a user's simulated git repo
type Session struct {
	ID         string
	Filesystem billy.Filesystem
	Repo       *git.Repository
	CreatedAt  time.Time
}

var sessions = make(map[string]*Session)

// InitSession creates a new in-memory git repository
func InitSession(id string) error {
	fs := memfs.New()
	st := memory.NewStorage()

	repo, err := git.Init(st, fs)
	if err != nil {
		return err
	}

	// Create initial files
	f, _ := fs.Create("README.md")
	f.Write([]byte("# My Project\n"))
	f.Close()

	sessions[id] = &Session{
		ID:         id,
		Filesystem: fs,
		Repo:       repo,
		CreatedAt:  time.Time{}, // Mock timestamp or use real
	}
	return nil
}

// ExecuteGitCommand parses a simple command string and executes it on the repo
// This is a naive implementation. In a real app, we'd parse args properly.
func ExecuteGitCommand(sessionID string, args []string) (string, error) {
	session, ok := sessions[sessionID]
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	if len(args) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	cmd := args[0]
	w, err := session.Repo.Worktree()
	if err != nil {
		return "", err
	}

	switch cmd {
	case "status":
		status, err := w.Status()
		if err != nil {
			return "", err
		}
		return status.String(), nil

	case "add":
		if len(args) < 2 {
			return "", fmt.Errorf("usage: git add <file>")
		}
		file := args[1]
		if file == "." {
			_, err = w.Add(".")
		} else {
			_, err = w.Add(file)
		}
		if err != nil {
			return "", err
		}
		return "Added " + file, nil

	case "commit":
		// simple parsing
		msg := "Default commit message"
		if len(args) >= 3 && args[1] == "-m" {
			msg = args[2] // This is very naive split
		}

		commit, err := w.Commit(msg, &git.CommitOptions{
			Author: &object.Signature{
				Name:  "User",
				Email: "user@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Commit created: %s", commit.String()), nil

	case "log":
		// ... implementation
		return "Log not implemented yet", nil

	default:
		return "", fmt.Errorf("command not supported: %s", cmd)
	}
}

// GraphState represents the serialized state for the frontend
type GraphState struct {
	Commits  []Commit          `json:"commits"`
	Branches map[string]string `json:"branches"`
	HEAD     Head              `json:"HEAD"`
	Files    []string          `json:"files"`
	Staging  []string          `json:"staging"`
	Modified []string          `json:"modified"`
}

type Commit struct {
	ID             string `json:"id"`
	Message        string `json:"message"`
	ParentID       string `json:"parentId"`
	SecondParentID string `json:"secondParentId"`
	Branch         string `json:"branch"` // Naive branch inference
	Timestamp      string `json:"timestamp"`
}

type Head struct {
	Type string `json:"type"` // "branch" or "commit"
	Ref  string `json:"ref,omitempty"`
	ID   string `json:"id,omitempty"`
}

func GetGraphState(sessionID string) (*GraphState, error) {
	session, ok := sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	state := &GraphState{
		Commits:  []Commit{},
		Branches: make(map[string]string),
	}

	// 1. Get HEAD
	ref, err := session.Repo.Head()
	if err != nil {
		// If empty repo (no commits yet)
		if err.Error() == "reference not found" {
			state.HEAD = Head{Type: "branch", Ref: "main"} // Default
			return state, nil
		}
		return nil, err
	}

	if ref.Name().IsBranch() {
		state.HEAD = Head{Type: "branch", Ref: ref.Name().Short()}
	} else {
		state.HEAD = Head{Type: "commit", ID: ref.Hash().String()}
	}

	// 2. Get Branches
	iter, err := session.Repo.Branches()
	if err != nil {
		return nil, err
	}
	iter.ForEach(func(r *plumbing.Reference) error {
		state.Branches[r.Name().Short()] = r.Hash().String()
		return nil
	})

	// 3. Walk Commits to build graph
	// This is a simplified walk. For a full graph with complex merges and disjoint branches,
	// we should iterate all refs and walk down.
	cIter, err := session.Repo.Log(&git.LogOptions{All: true})
	if err != nil {
		// Maybe no commits yet
		return state, nil
	}

	cIter.ForEach(func(c *object.Commit) error {
		// Naive logic:
		// In the real app, we need to map commits to lanes/branches for coloring.
		// For now, we just list them.

		parentID := ""
		if len(c.ParentHashes) > 0 {
			parentID = c.ParentHashes[0].String()
		}
		secondParentID := ""
		if len(c.ParentHashes) > 1 {
			secondParentID = c.ParentHashes[1].String()
		}

		state.Commits = append(state.Commits, Commit{
			ID:             c.Hash.String(),
			Message:        c.Message,
			ParentID:       parentID,
			SecondParentID: secondParentID,
			Timestamp:      c.Committer.When.Format(time.RFC3339),
		})
		return nil
	})

	// 4. Get Status (Files, Staging, Modified)
	w, _ := session.Repo.Worktree()
	status, _ := w.Status()

	// status is a map[string]*FileStatus
	for file, s := range status {
		if s.Staging != git.Unmodified {
			state.Staging = append(state.Staging, file)
		}
		if s.Worktree != git.Unmodified {
			state.Modified = append(state.Modified, file)
		}
		state.Files = append(state.Files, file)
	}

	return state, nil
}
