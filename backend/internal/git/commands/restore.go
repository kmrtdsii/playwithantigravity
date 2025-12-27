package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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

	if staged {
		// restore --staged: Unstage files (reset index to HEAD)
		headRef, err := repo.Head()
		if err != nil {
			// No HEAD (initial commit?), unstaging means removing from index
			// We can iterate files and remove them from index
			idx, idxErr := repo.Storer.Index()
			if idxErr != nil {
				return "", idxErr
			}
			for _, file := range files {
				// Remove file from index entries
				newEntries := make([]*index.Entry, 0, len(idx.Entries))
				for _, e := range idx.Entries {
					if e.Name != file {
						newEntries = append(newEntries, e)
					}
				}
				idx.Entries = newEntries
			}
			_ = repo.Storer.SetIndex(idx)
			return "Unstaged files (initial commit)", nil
		}

		// HEAD exists, copy HEAD entry to Index
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

		for _, file := range files {
			// 1. Check if file exists in HEAD
			entry, err := tree.File(file)
			if err != nil {
				// File not in HEAD (it was a new file added). Remove from Index.
				newEntries := make([]*index.Entry, 0, len(idx.Entries))
				for _, e := range idx.Entries {
					if e.Name != file {
						newEntries = append(newEntries, e)
					}
				}
				idx.Entries = newEntries
				continue
			}

			// 2. File exists in HEAD. Update Index to match HEAD.
			found := false
			for i, e := range idx.Entries {
				if e.Name == file {
					// Update
					e.Hash = entry.Hash
					e.Mode = entry.Mode
					// ModifiedAt, Size etc?
					idx.Entries[i] = e
					found = true
					break
				}
			}
			if !found {
				// If not in index but in HEAD, add it back
				idx.Entries = append(idx.Entries, &index.Entry{
					Name: file,
					Hash: entry.Hash,
					Mode: entry.Mode,
				})
			}
		}
		_ = repo.Storer.SetIndex(idx)
		return "Unstaged files", nil

	} else {
		// restore (worktree): Discard changes in worktree (restore from Index)
		// Use w.Checkout to restore files from Index if possible, else manual

		// Try standard Checkout first if supported (it writes to worktree from index)
		// Usually checkout with options.Files works for restoring from index.
		// If Hash is empty, it uses Index.
		// We must do it manually for specific files because w.Checkout(Force:true)
		// without a 'Files' filter would overwrite the entire worktree.

		// If we can't use Checkout with Files, we must do it manually for specific files.
		// w.Checkout without Files checks out everything! That is bad if we only want one file.
		// So we MUST implement manual restore.

		idx, err := repo.Storer.Index()
		if err != nil {
			return "", err
		}

		for _, file := range files {
			// Find entry in index
			var entry *index.Entry
			for _, e := range idx.Entries {
				if e.Name == file {
					entry = e
					break
				}
			}

			if entry == nil {
				continue
			}

			// Read blob from Object Storage
			blob, err := repo.BlobObject(entry.Hash)
			if err != nil {
				return "", fmt.Errorf("failed to read blob %s: %w", entry.Hash, err)
			}
			reader, err := blob.Reader()
			if err != nil {
				return "", err
			}
			defer reader.Close()

			// Write to Worktree
			f, err := s.Filesystem.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return "", err
			}

			if _, err := io.Copy(f, reader); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
		}
		return "Restored files in worktree", nil
	}
}

func (c *RestoreCommand) Help() string {
	return `📘 GIT-RESTORE (1)                                      Git Manual

 💡 DESCRIPTION
    ・ファイルの変更を取り消して、以前の状態に戻す
    ・ステージングした変更を取り消す（--staged）
    オプションなしの場合は「インデックスの内容」でワーキングツリーを上書きします（変更の破棄）。
    ` + "`--staged`" + ` を付けると「HEADの内容」でインデックスを上書きします（ステージングの取り消し）。

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
`
}
