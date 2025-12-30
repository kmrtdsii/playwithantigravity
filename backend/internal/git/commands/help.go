package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("help", func() git.Command { return &HelpCommand{} })
}

type HelpCommand struct{}

// Ensure HelpCommand implements git.Command
var _ git.Command = (*HelpCommand)(nil)

// Command metadata for help display
type cmdMeta struct {
	Category string
	Desc     string
}

// Categories
const (
	CatStart    = "Start a working area"
	CatWork     = "Work on the current change"
	CatHistory  = "Examine the history and state"
	CatGrow     = "Grow, mark and tweak your common history"
	CatCollab   = "Collaborate"
	CatShell    = "Shell & Utilities"
	CatInternal = "Internal" // Hidden
)

var commandMetadata = map[string]cmdMeta{
	// Start
	"clone":    {CatStart, "Clone a repository into a new directory"},
	"init":     {CatStart, "Create an empty Git repository (not supported checking out new projects yet)"},
	"worktree": {CatStart, "Manage multiple working trees (not supported in current UI)"},

	// Work
	"add":     {CatWork, "Add file contents to the index"},
	"clean":   {CatWork, "Remove untracked files from the working tree"},
	"restore": {CatWork, "Restore working tree files"},
	"rm":      {CatWork, "Remove files from the working tree and from the index"},

	// History
	"blame":  {CatHistory, "Show what revision and author last modified each line of a file"},
	"diff":   {CatHistory, "Show changes between commits, commit and working tree, etc"},
	"log":    {CatHistory, "Show commit logs"},
	"reflog": {CatHistory, "Manage reflog information"},
	"show":   {CatHistory, "Show various types of objects"},
	"status": {CatHistory, "Show the working tree status"},

	// Grow
	"branch":      {CatGrow, "List, create, or delete branches"},
	"checkout":    {CatGrow, "Switch branches or restore working tree files"},
	"cherry-pick": {CatGrow, "Apply the changes introduced by some existing commits"},
	"commit":      {CatGrow, "Record changes to the repository"},
	"merge":       {CatGrow, "Join two or more development histories together"},
	"rebase":      {CatGrow, "Reapply commits on top of another base tip"},
	"reset":       {CatGrow, "Reset current HEAD to the specified state"},
	"revert":      {CatGrow, "Revert some existing commits"},
	"stash":       {CatGrow, "Stash the changes in a dirty working directory away"},
	"switch":      {CatGrow, "Switch branches"},
	"tag":         {CatGrow, "Create, list, delete or verify a tag object"},

	// Collab
	"fetch":  {CatCollab, "Download objects and refs from another repository"},
	"pull":   {CatCollab, "Fetch from and integrate with another repository or a local branch"},
	"push":   {CatCollab, "Update remote refs along with associated objects (simulated)"},
	"remote": {CatCollab, "Manage set of tracked repositories"},

	// Shell
	"cd":      {CatShell, "Change the current directory"},
	"ls":      {CatShell, "List directory contents"},
	"pwd":     {CatShell, "Print name of current/working directory"},
	"touch":   {CatShell, "Change file access and modification times"},
	"help":    {CatShell, "Display help information"},
	"version": {CatShell, "Show version info"},

	// Internal / Hidden (Marked but filtered later)
	"simulate-commit": {CatInternal, "Simulate a commit"},
	"merge-pr":        {CatInternal, "Merge a pull request"},
}

// Order of categories for display
var categoryOrder = []string{
	CatStart,
	CatWork,
	CatHistory,
	CatGrow,
	CatCollab,
	CatShell,
}

func (c *HelpCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	if len(args) > 1 {
		subcmd := args[1]
		helpStr, err := git.GetCommandHelp(subcmd)
		if err != nil {
			// Fallback if not found in metadata or registry
			if meta, ok := commandMetadata[subcmd]; ok {
				return fmt.Sprintf("%s: %s\n", subcmd, meta.Desc), nil
			}
			return fmt.Sprintf("git help: unknown command '%s'", subcmd), nil
		}
		return helpStr, nil
	}

	// 1. Group commands by category
	cmds := git.GetSupportedCommands()
	grouped := make(map[string][]string)

	// Max length for padding
	maxLen := 0

	for _, cmd := range cmds {
		meta, ok := commandMetadata[cmd]
		if !ok || meta.Category == CatInternal {
			continue // Skip hidden or unknown
		}
		grouped[meta.Category] = append(grouped[meta.Category], cmd)
		if len(cmd) > maxLen {
			maxLen = len(cmd)
		}
	}

	// 2. Build Output
	var sb strings.Builder
	sb.WriteString("usage: git [--version] [--help] <command> [<args>]\n\n")
	sb.WriteString("These are common Git commands used in various situations:\n")

	for _, cat := range categoryOrder {
		list, ok := grouped[cat]
		if !ok || len(list) == 0 {
			continue
		}
		sort.Strings(list)

		sb.WriteString(fmt.Sprintf("\n%s:\n", cat))
		for _, cmd := range list {
			meta := commandMetadata[cmd]
			padding := strings.Repeat(" ", maxLen-len(cmd)+3)
			sb.WriteString(fmt.Sprintf("   %s%s%s\n", cmd, padding, meta.Desc))
		}
	}

	sb.WriteString("\nType 'git help <command>' for more information about a specific command.")
	return sb.String(), nil
}

func (c *HelpCommand) Help() string {
	return `📘 GIT-HELP (1)                                         Git Manual

 💡 DESCRIPTION
    Gitコマンドの使い方やオプションを確認します。
    困った時はまずこれを使ってみてください。
    引数なしで実行すると、利用可能な主要コマンド一覧を表示します。

 📋 SYNOPSIS
    git help [-a] [<command>]

 ⚙️  COMMON OPTIONS
    -a, --all
        全てのコマンドを表示します。

 🛠  EXAMPLES
    1. コマンドの使い方を調べる
       $ git help commit
`
}
