package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("blame", func() git.Command { return &BlameCommand{} })
}

type BlameCommand struct{}

// Ensure BlameCommand implements git.Command
var _ git.Command = (*BlameCommand)(nil)

func (c *BlameCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.RLock()
	defer s.RUnlock()

	if len(args) < 2 {
		return "", fmt.Errorf("usage: git blame <file>")
	}
	filePath := args[1]

	// Normalize path
	// If path starts with /, treat it relative to repo root (which is CurrentDir if at root)
	// Complicated relative path logic might be needed if cwd is not root.
	// For now, assume simple relative path handling relative to repo root or use s.GetRepo() logic.

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("not a git repository (or any of the parent directories)")
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("HEAD not found: %v", err)
	}

	commit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", fmt.Errorf("invalid HEAD commit: %v", err)
	}

	// go-git's Blame takes a path relative to the worktree root.
	// We need to resolve filePath relative to the repo root.
	// For simplicity in this simulation, assuming cwd is repo root or fairly flat structure.
	// If s.CurrentDir is "/" (root of memfs), and repo is there.
	// We just strip leading slash if present.
	cleanPath := strings.TrimPrefix(filePath, "/")

	// Check/Read file first to ensure it exists and we have content
	// We read content ourselves to ensure we can display lines even if BlameResult.Text is weird
	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}
	file, err := tree.File(cleanPath)
	if err != nil {
		return "", fmt.Errorf("no such path '%s' in HEAD", cleanPath)
	}

	// Calculate Blame
	blameResult, err := gogit.Blame(commit, cleanPath)
	if err != nil {
		return "", fmt.Errorf("blame failed: %v", err)
	}

	// Read file content lines to ensure 1:1 mapping if needed
	// But let's try using `blameResult.Lines[i].Text` first.
	// If `Text` is empty, we fall back? No, let's use the explicit file content split.
	content, err := file.Contents()
	if err != nil {
		return "", err
	}
	fileLines := strings.Split(content, "\n")
	// Removing trailing empty string from split if file ends with newline?
	// strings.Split("abc\n", "\n") -> ["abc", ""]
	if len(fileLines) > 0 && fileLines[len(fileLines)-1] == "" {
		fileLines = fileLines[:len(fileLines)-1]
	}

	var sb strings.Builder
	for i, line := range blameResult.Lines {
		if i >= len(fileLines) {
			break
		}

		hashStr := line.Hash.String()
		if len(hashStr) > 8 {
			hashStr = hashStr[:8]
		}

		author := line.Author
		dateStr := line.Date.Format("2006-01-02 15:04:05")

		// Use actual file content for the text
		lineText := fileLines[i]

		sb.WriteString(fmt.Sprintf("%s (%-20s %s %4d) %s\n",
			hashStr,
			truncateString(author, 20),
			dateStr,
			i+1,
			lineText))
	}

	return sb.String(), nil
}

func (c *BlameCommand) Help() string {
	return `📘 BLAME (1)                                          Git Manual

 💡 DESCRIPTION
    ファイルの各行が「誰によって」「いつ」「どのコミットで」変更されたかを表示します。
    バグの原因調査や、コードの意図を確認する際に非常に便利です。

 📋 SYNOPSIS
    git blame <file>

 🛠  EXAMPLES
    1. README.md の履歴を見る
       $ git blame README.md

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-blame
`
}

func truncateString(s string, l int) string {
	if len(s) > l {
		return s[:l-1] + "."
	}
	return s
}
