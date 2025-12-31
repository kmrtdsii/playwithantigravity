package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("log", func() git.Command { return &LogCommand{} })
}

type LogCommand struct{}

// Ensure LogCommand implements git.Command
var _ git.Command = (*LogCommand)(nil)

type LogOptions struct {
	Oneline bool
	Graph   bool
	Args    []string // Revisions or paths
}

func (c *LogCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	return c.executeLog(s, repo, opts)
}

func (c *LogCommand) parseArgs(args []string) (*LogOptions, error) {
	opts := &LogOptions{}
	cmdArgs := args[1:]
	for _, arg := range cmdArgs {
		switch arg {
		case "--oneline":
			opts.Oneline = true
		case "--graph":
			opts.Graph = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			opts.Args = append(opts.Args, arg)
		}
	}
	return opts, nil
}

func (c *LogCommand) executeLog(_ *git.Session, repo *gogit.Repository, opts *LogOptions) (string, error) {
	// executeLog performs the log operation with optional graph rendering.
	// This implementation attempts a simplified ASCII graph.
	cIter, err := repo.Log(&gogit.LogOptions{All: false}) // HEAD only traversal by default
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	// Graph state
	// columns tracks the commit hashes currently "active" in vertical lines
	var columns []string

	err = cIter.ForEach(func(c *object.Commit) error {
		var graphLine string
		hash := c.Hash.String()

		if opts.Graph {
			// 1. Determine column for current commit
			colIndex := -1

			// Check if this commit connects to an existing column (is a parent of a previous commit)
			for i, h := range columns {
				if h == hash {
					colIndex = i
					break
				}
			}

			// If not found in columns, it's the start of the log traverse (HEAD), so it goes in a new column (or 0)
			if colIndex == -1 {
				// For the very first commit (HEAD), let's put it in column 0
				if len(columns) == 0 {
					colIndex = 0
					columns = append(columns, hash)
				} else {
					// This case implies a commit appeared that was not a parent of anyone seen so far?
					// In standard git log traversal starting from HEAD, this shouldn't happen for the first commit,
					// but might happen for disjoint histories or strict topo order oddities.
					// We'll append it.
					colIndex = len(columns)
					columns = append(columns, hash)
				}
			}

			// 2. Prepare graph characters
			// Basic logic:
			// '*' at colIndex
			// '|' for other columns

			// We need to construct the line for THIS commit row
			// And then update columns for the NEXT row (parents)

			lineChars := make([]string, len(columns))
			for i := range columns {
				if i == colIndex {
					lineChars[i] = "*"
				} else {
					lineChars[i] = "|"
				}
			}

			// Colorize asterisk (if we supported color, but simple for now)
			graphLine = strings.Join(lineChars, " ")

			// Indent message to align past the graph
			// Standard git aligns nicely. simpler here: just space.
			// indent = "" // handled by graphLine placement

			// 3. Update columns for next row (Process Parents)
			// Replace current commit hash in columns with its parents.
			// If 1 parent: swap.
			// If 2 parents: swap first, append second? (roughly)
			// If 0 parents: remove column.

			parents := c.ParentHashes
			if len(parents) == 0 {
				// Root commit: remove this column
				// We strip it from the slice.
				// To keep alignment valid for others, we usually want empty space or collapse?
				// "git log" collapses.
				columns = append(columns[:colIndex], columns[colIndex+1:]...)
			} else {
				// Replace current with Parent 0
				columns[colIndex] = parents[0].String()

				// If Merge (multiple parents), add others
				if len(parents) > 1 {
					for _, p := range parents[1:] {
						// Insert or append?
						// Git graph usually inserts next to it to show fork.
						// Simplified: append to end.
						found := false
						pStr := p.String()
						for _, existing := range columns {
							if existing == pStr {
								found = true
								break
							}
						}
						if !found {
							columns = append(columns, pStr)
						}
					}
				}
			}
		}

		// Rendering
		msgFirstCheck := strings.Split(c.Message, "\n")[0]

		if opts.Oneline {
			prefix := ""
			if opts.Graph {
				prefix = fmt.Sprintf("%-10s", graphLine) // min width padding?
				// Better: just space
				prefix = graphLine + " "
			}
			sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, hash[:7], msgFirstCheck))
		} else {
			// Multiline graph is hard to render correctly without line-by-line tracking.
			// Fallback: Just show graph on first line, indent others.
			prefix := ""
			indentStr := ""
			if opts.Graph {
				prefix = graphLine + " "
				indentStr = strings.Repeat("| ", len(columns)) // approximation
				if len(columns) == 0 {
					indentStr = "  "
				}
			}

			sb.WriteString(fmt.Sprintf("%scommit %s\n%sAuthor: %s <%s>\n%sDate:   %s\n\n%s    %s\n\n",
				prefix,
				hash,
				indentStr,
				c.Author.Name,
				c.Author.Email,
				indentStr,
				c.Author.When.Format(time.RFC3339),
				indentStr,
				strings.TrimSpace(c.Message),
			))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (c *LogCommand) Help() string {
	return `📘 GIT-LOG (1)                                          Git Manual

 💡 DESCRIPTION
    ・これまでのコミット履歴（いつ、誰が、何をしたか）を表示する
    ・プロジェクトの歴史を遡って確認する

 📋 SYNOPSIS
    git log [--oneline] [--graph]

 ⚙️  COMMON OPTIONS
    --oneline
        各コミットを1行（ハッシュの一部とメッセージのみ）で表示します。
        履歴の概観をつかむのに便利です。

    --graph
        履歴をグラフ（ASCIIアート）として表示します。
        ブランチやマージの流れを視覚的に確認できます。

 🛠  EXAMPLES
    1. 詳細なログを表示
       $ git log

    2. 簡潔なログを表示
       $ git log --oneline

    3. グラフ付きで表示
       $ git log --oneline --graph

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-log
`
}
