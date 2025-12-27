package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("restore", func() git.Command { return &RestoreCommand{} })
}

type RestoreCommand struct{}

func (c *RestoreCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	staged := false
	var files []string

	// Basic parsing
	for _, arg := range args {
		if arg == "restore" {
			continue
		}
		if arg == "--staged" {
			staged = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue // ignore other flags
		}
		files = append(files, arg)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("fatal: you must specify path(s) to restore")
	}

	// 1. Expand Pathspecs
	targets, err := c.expandPathspecs(repo, files)
	if err != nil {
		return "", err
	}
	if len(targets) == 0 {
		// If original files contained ".", it means we wanted everything but found nothing
		for _, f := range files {
			if f == "." {
				return "Nothing to restore (no tracked files found)", nil
			}
		}
	}

	// 2. Dispatch
	if staged {
		return c.restoreStaged(repo, targets, len(targets) > len(files)) // heuristics for "all" message
	} else {
		return c.restoreWorktree(repo, targets, len(targets) > len(files))
	}
}

// expandPathspecs resolves "." to all files in index, otherwise returns files as-is
func (c *RestoreCommand) expandPathspecs(repo *gogit.Repository, files []string) ([]string, error) {
	idx, err := repo.Storer.Index()
	if err != nil {
		return nil, err
	}

	containDot := false
	for _, f := range files {
		if f == "." {
			containDot = true
			break
		}
	}

	if containDot {
		var targets []string
		// In GitGym, operations are generally at repo root.
		// If we support subdirectories later, we need to calculate path relative to Repo Root.
		// For now, assume '.' implies everything in the repo (recursive).
		for _, e := range idx.Entries {
			targets = append(targets, e.Name)
		}
		return targets, nil
	}

	return files, nil
}

func (c *RestoreCommand) restoreStaged(repo *gogit.Repository, files []string, isMassOperation bool) (string, error) {
	headRef, err := repo.Head()
	if err != nil {
		// No HEAD (initial commit?), unstaging means removing from index
		idx, idxErr := repo.Storer.Index()
		if idxErr != nil {
			return "", idxErr
		}

		count := 0
		for _, file := range files {
			// Remove file from index entries
			// Note: This is O(N*M) naive implementation.
			newEntries := make([]*index.Entry, 0, len(idx.Entries))
			found := false
			for _, e := range idx.Entries {
				if e.Name != file {
					newEntries = append(newEntries, e)
				} else {
					found = true
				}
			}
			if found {
				idx.Entries = newEntries
				count++
			}
		}
		_ = repo.Storer.SetIndex(idx)
		return fmt.Sprintf("Unstaged files from initial commit (%d files)", count), nil
	}

	// HEAD exists
	commit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return "", err
	}

	successCount := 0
	for _, file := range files {
		// 1. Check if file exists in HEAD
		entry, err := tree.File(file)
		if err != nil {
			// File not in HEAD (new file). Remove from Index.
			newEntries := make([]*index.Entry, 0, len(idx.Entries))
			found := false
			for _, e := range idx.Entries {
				if e.Name != file {
					newEntries = append(newEntries, e)
				} else {
					found = true
				}
			}
			if found {
				idx.Entries = newEntries
				successCount++
			}
			continue
		}

		// 2. File exists in HEAD. Update Index.
		foundInIndex := false
		for i, e := range idx.Entries {
			if e.Name == file {
				e.Hash = entry.Hash
				e.Mode = entry.Mode
				idx.Entries[i] = e
				foundInIndex = true
				successCount++
				break
			}
		}
		if !foundInIndex {
			// Add back to index if missing
			idx.Entries = append(idx.Entries, &index.Entry{
				Name: file,
				Hash: entry.Hash,
				Mode: entry.Mode,
			})
			successCount++
		}
	}

	err = repo.Storer.SetIndex(idx)
	if err != nil {
		return "", err
	}

	if isMassOperation {
		return fmt.Sprintf("Unstaged all files in current directory (%d files)", successCount), nil
	}
	return "Unstaged files", nil
}

func (c *RestoreCommand) restoreWorktree(repo *gogit.Repository, files []string, isMassOperation bool) (string, error) {
	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return "", err
	}

	restoredCount := 0
	for _, file := range files {
		var entry *index.Entry
		for _, e := range idx.Entries {
			if e.Name == file {
				entry = e
				break
			}
		}

		if entry == nil {
			// If explicitly requested but not in index, error
			if !isMassOperation {
				return "", fmt.Errorf("pathspec '%s' did not match any file(s) known to git", file)
			}
			continue
		}

		blob, err := repo.BlobObject(entry.Hash)
		if err != nil {
			return "", fmt.Errorf("failed to read blob %s: %w", entry.Hash, err)
		}
		reader, err := blob.Reader()
		if err != nil {
			return "", err
		}
		defer reader.Close()

		f, err := w.Filesystem.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(f, reader); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
		restoredCount++
	}

	if isMassOperation {
		return fmt.Sprintf("Restored all tracked files in current directory (%d files)", restoredCount), nil
	}
	return "Restored files in worktree", nil
}

func (c *RestoreCommand) Help() string {
	return `📘 GIT-RESTORE (1)                                      Git Manual

 💡 DESCRIPTION
    ・ファイルの変更を破棄して、元の状態に戻す
    ・ステージングした変更を取り消す（--staged）
    
    「編集をやり直したい」時や「addを取り消したい」時に使います。

 📋 SYNOPSIS
    git restore [<options>] <pathspec>...

 ⚙️  COMMON OPTIONS
    --staged
        ワーキングツリーではなく、インデックス（ステージングエリア）を復元します。
        ` + "`git add`" + ` した内容を取り消す際によく使用します。

 🛠  EXAMPLES
    1. ワーキングツリーの変更を破棄する（元に戻す）
       $ git restore README.md

    2. ステージングした変更を取り消す（Unstage）
       $ git restore --staged README.md

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-restore

 💡 TIPS
    ` + "`" + `git restore .` + "`" + ` を実行すると、現在のディレクトリ以下の
    「まだaddしていない変更」をすべて破棄します（Untrackedなファイルは消えません）。
    「実験的にいろいろいじったけど、全部なかったことにしてスッキリしたい」時に便利です。
`
}
