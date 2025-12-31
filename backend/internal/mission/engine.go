package mission

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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
	if err := e.cleanWorkspace(sess); err != nil {
		return "", fmt.Errorf("failed to clean workspace: %w", err)
	}
	// Re-create root if needed? MemFS handles it.
	// We use /project as the default directory to avoid "cannot init repo at root" errors
	_ = sess.Filesystem.MkdirAll("/project", 0755)
	sess.CurrentDir = "/project"

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

// cleanWorkspace removes all files and directories in the root of the session filesystem
func (e *Engine) cleanWorkspace(sess *state.Session) error {
	// Clear Repos map
	sess.Repos = make(map[string]*gogit.Repository)

	// List all files in root
	files, err := sess.Filesystem.ReadDir("/")
	if err != nil {
		// If root doesn't exist, that's fine (though unlikely for memfs)
		return nil
	}

	for _, file := range files {
		path := "/" + file.Name()
		if err := sess.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

// runCommand handles git commands and basic shell simulation (echo, mkdir, cd, redirection)
func (e *Engine) runCommand(ctx context.Context, session *state.Session, cmdStr string) error {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return nil
	}

	// 1. Handle "mkdir"
	if strings.HasPrefix(cmdStr, "mkdir ") {
		dirName := strings.TrimSpace(strings.TrimPrefix(cmdStr, "mkdir "))
		dirName = strings.Trim(dirName, "\"'")
		return session.Filesystem.MkdirAll(dirName, 0755)
	}

	// 2. Handle "cd"
	if strings.HasPrefix(cmdStr, "cd ") {
		dirName := strings.TrimSpace(strings.TrimPrefix(cmdStr, "cd "))
		dirName = strings.Trim(dirName, "\"'")
		// Update session's current directory
		if dirName == "/" {
			session.CurrentDir = "/"
		} else if strings.HasPrefix(dirName, "/") {
			session.CurrentDir = dirName
		} else {
			if session.CurrentDir == "/" {
				session.CurrentDir = "/" + dirName
			} else {
				session.CurrentDir = session.CurrentDir + "/" + dirName
			}
		}
		return nil
	}

	// 3. Handle "echo" with redirection
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

			// Resolve path relative to CurrentDir
			if !strings.HasPrefix(targetFile, "/") {
				if session.CurrentDir == "/" {
					targetFile = "/" + targetFile
				} else {
					targetFile = session.CurrentDir + "/" + targetFile
				}
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
			// Search log for commits. If MessagePattern is empty, just check if any commit exists.
			iter, iterErr := repo.Log(&gogit.LogOptions{})
			if iterErr == nil {
				_ = iter.ForEach(func(c *object.Commit) error {
					if check.MessagePattern == "" {
						// Any commit passes
						passed = true
					} else if strings.Contains(c.Message, check.MessagePattern) {
						passed = true
					}
					return nil
				})
			}

		case "file_content":
			// Resolve path relative to CurrentDir
			targetPath := check.Path
			if !strings.HasPrefix(targetPath, "/") {
				// Avoid double slash if CurrentDir is "/"
				if sess.CurrentDir == "/" {
					targetPath = "/" + targetPath
				} else {
					targetPath = sess.CurrentDir + "/" + targetPath
				}
			}

			// Check file content
			f, err := sess.Filesystem.Open(targetPath)
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

		case "file_tracked":
			// Check if a file is tracked by git (exists in HEAD commit)
			// A file is "tracked" if it's in the HEAD commit's tree
			headRef, hErr := repo.Head()
			if hErr == nil {
				commit, cErr := repo.CommitObject(headRef.Hash())
				if cErr == nil {
					tree, tErr := commit.Tree()
					if tErr == nil {
						_, fErr := tree.File(check.Path)
						passed = (fErr == nil)
					}
				}
			}

		case "clean_working_tree":
			// Check if working tree is clean (no unstaged or uncommitted changes)
			w, wErr := repo.Worktree()
			if wErr == nil {
				status, sErr := w.Status()
				if sErr == nil {
					passed = status.IsClean()
				}
			}

		case "branch_exists":
			// Check if a branch with the given name exists
			refs, rErr := repo.References()
			if rErr == nil {
				_ = refs.ForEach(func(ref *plumbing.Reference) error {
					if ref.Name().IsBranch() && ref.Name().Short() == check.Name {
						passed = true
					}
					return nil
				})
			}

		case "current_branch":
			// Check if current HEAD is on the specified branch
			headRef, hErr := repo.Head()
			if hErr == nil && headRef.Name().IsBranch() {
				passed = headRef.Name().Short() == check.Name
			}

		case "head_commit_message":
			// Check if HEAD commit message matches the pattern
			headRef, hErr := repo.Head()
			if hErr == nil {
				commit, cErr := repo.CommitObject(headRef.Hash())
				if cErr == nil {
					if check.MessagePattern == "" {
						passed = true
					} else {
						// Simple contains check, or exact match? 'pattern' usually implies contains/regex.
						// Using strings.Contains like commit_exists
						passed = strings.Contains(commit.Message, check.MessagePattern)
					}
				}
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
