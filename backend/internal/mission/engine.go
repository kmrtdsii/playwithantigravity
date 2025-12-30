package mission

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/kurobon/gitgym/backend/internal/state"
)

type Engine struct {
	Loader  *Loader
	Manager *state.SessionManager
}

func NewEngine(loader *Loader, manager *state.SessionManager) *Engine {
	return &Engine{
		Loader:  loader,
		Manager: manager,
	}
}

// StartMission initializes a temporary session for the mission.
func (e *Engine) StartMission(ctx context.Context, missionID string) (string, error) {
	m, err := e.Loader.LoadMission(missionID)
	if err != nil {
		return "", err
	}

	// Create a unique session ID for this mission instance
	// In a real app we might suffix with user ID, but here we just use missionID + timestamp or just missionID for simplicity if single user mode.
	// Actually, let's assume we return a generic "mission-<id>" session.
	// If concurrent users, we need Unique IDs.
	// For now, let's assume one active mission per browser session.
	// We'll generate a random suffix or use a fixed one if debugging.
	// Let's use "mission-<missionID>" for now.
	sessionID := fmt.Sprintf("mission-%s", missionID)
	// Force recreate
	// Note: SessionManager.CreateSession reuses if exists. We want a fresh state.
	// We should probably remove it first if it exists?
	// or assume the caller handles cleanup.
	// state.Session has no "Reset" but we can just manipulate the filesystem.
	// Let's implement a hard reset by creating a NEW session ID every time to be safe?
	// Or clear the existing one.

	// Better: SessionManager doesn't expose "DeleteSession".
	// We will manually clear the filesystem of the session if it exists.
	// But `CreateSession` locks.

	// Implementation Detail: We'll stick to one ID per mission type for now to save temp space,
	// but we need to Clean it.

	sess, err := e.Manager.CreateSession(sessionID)
	if err != nil {
		return "", err
	}

	// sess.Lock() caused deadlock with git.Dispatch
	// defer sess.Unlock()

	// 1. Clean Filesystem (Reset State)
	// We need a way to empty the MemFS.
	// Sess.RemoveAll("/") might work if implemented recursively.
	// state.Session has RemoveAll.
	_ = sess.RemoveAll("/")
	// Re-create root if needed? MemFS handles it.
	sess.CurrentDir = "/"
	sess.Repos = make(map[string]*gogit.Repository)
	// We can't easily wipe sess.Repos without raw access.
	// Let's just create a new Session object in the Manager if possible?
	// Manager.sessions is private.

	// Workaround: We'll assume RemoveAll("/") clears the files,
	// and we manually clear the Repos map (since we have access to the pointer).
	// But `Repos` is defined in state.Session.
	// We just need to clear it.
	for k := range sess.Repos {
		delete(sess.Repos, k)
	}

	// 2. Run Setup Commands
	for _, cmdStr := range m.Setup {
		if err := e.runCommand(ctx, sess, cmdStr); err != nil {
			return "", fmt.Errorf("setup failed at '%s': %w", cmdStr, err)
		}
	}

	// Reset Reflog for the user starting fresh
	sess.Reflog = nil

	return sessionID, nil
}

// runCommand handles git commands and basic shell simulation (echo, redirection)
func (e *Engine) runCommand(ctx context.Context, session *state.Session, cmdStr string) error {
	// 1. Handle "echo" with redirection
	if strings.HasPrefix(cmdStr, "echo ") {
		parts := strings.SplitN(cmdStr, ">", 2)
		if len(parts) == 2 {
			content := strings.Trim(parts[0], " \"'") // simplistic trim
			content = strings.TrimPrefix(content, "echo ")
			content = strings.Trim(content, "\"'")

			// Handle ">>" vs ">"
			appendMode := false
			targetFile := strings.TrimSpace(parts[1])
			if strings.HasPrefix(targetFile, ">") {
				appendMode = true
				targetFile = strings.TrimSpace(strings.TrimPrefix(targetFile, ">"))
			}

			// Write to file
			// We can't use os constants directly with billy sometimes depending on impl,
			// but usually they map.
			// Simpler: Use util helper or just WriteFile if overwrite.

			if !appendMode {
				// Overwrite
				f, err := session.Filesystem.Create(targetFile)
				if err != nil {
					return err
				}
				_, err = f.Write([]byte(content + "\n"))
				_ = f.Close()
				return err
			} else {
				// Append

				// Read existing content safely
				var builder strings.Builder
				f, err := session.Filesystem.OpenFile(targetFile, os.O_RDWR|os.O_CREATE, 0644)
				if err == nil {
					buf := make([]byte, 1024)
					for {
						n, readErr := f.Read(buf)
						if n > 0 {
							builder.Write(buf[:n])
						}
						if readErr == io.EOF {
							break
						}
						if readErr != nil {
							// Handle other errors or just break?
							break
						}
					}
					f.Close()
				}
				builder.WriteString(content + "\n")

				// Re-open/Create to overwrite with full content (MemFS might not support efficient append seek)
				f, err = session.Filesystem.Create(targetFile)
				if err != nil {
					return err
				}
				_, err = f.Write([]byte(builder.String()))
				_ = f.Close()
				return err
			}
		}
	}

	// 2. Handle Git Commands
	name, args := git.ParseCommand(cmdStr)
	if name == "" {
		return nil // empty line
	}

	// Use git.Dispatch
	_, err := git.Dispatch(ctx, (*git.Session)(session), name, args)
	return err
}

type VerificationResult struct {
	Success   bool          `json:"success"`
	MissionID string        `json:"missionId"`
	Progress  []CheckResult `json:"progress"`
}

type CheckResult struct {
	Description string `json:"description"`
	Passed      bool   `json:"passed"`
}

func (e *Engine) VerifyMission(sessionID string, missionID string) (*VerificationResult, error) {
	m, err := e.Loader.LoadMission(missionID)
	if err != nil {
		return nil, err
	}

	sess, ok := e.Manager.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	sess.RLock() // Read lock
	defer sess.RUnlock()

	repo := sess.GetRepo() // Assuming root repo
	if repo == nil {
		// If setup failed or no repo, fail all
		return &VerificationResult{Success: false, MissionID: missionID}, nil
	}

	var results []CheckResult
	allPassed := true

	for _, check := range m.Validation.Checks {
		passed := false
		switch check.Type {
		case "no_conflict":
			// Check status
			w, _ := repo.Worktree()
			status, _ := w.Status()
			passed = true
			for _, s := range status {
				if s.Staging == 'U' || s.Worktree == 'U' {
					passed = false
					break
				}
			}

		case "commit_exists":
			// Search log for message pattern
			iter, _ := repo.Log(&gogit.LogOptions{})
			_ = iter.ForEach(func(c *object.Commit) error {
				if strings.Contains(c.Message, check.MessagePattern) {
					passed = true
				}
				return nil
			})

		case "file_content":
			// Check file content
			f, err := sess.Filesystem.Open(check.Path)
			if err == nil {
				// Read content
				contentBytes, readErr := io.ReadAll(f)
				if readErr != nil {
					// Handle read error
				}
				content := string(contentBytes)
				f.Close()

				matchAll := true
				for _, substr := range check.Contains {
					if !strings.Contains(content, substr) {
						matchAll = false
						break
					}
				}
				passed = matchAll
			}
		}

		results = append(results, CheckResult{
			Description: check.Description,
			Passed:      passed,
		})
		if !passed {
			allPassed = false
		}
	}

	return &VerificationResult{
		Success:   allPassed,
		MissionID: missionID,
		Progress:  results,
	}, nil
}
