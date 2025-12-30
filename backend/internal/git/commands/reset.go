package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("reset", func() git.Command { return &ResetCommand{} })
}

type ResetCommand struct{}

// Ensure ResetCommand implements git.Command
var _ git.Command = (*ResetCommand)(nil)

type ResetOptions struct {
	Mode   gogit.ResetMode
	Target string
}

func (c *ResetCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// 1. Parse Args
	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// 2. Resolve Context
	targetHash, err := repo.ResolveRevision(plumbing.Revision(opts.Target))
	if err != nil {
		return "", err
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// 3. Execution
	return c.executeReset(s, w, targetHash, opts)
}

func (c *ResetCommand) parseArgs(args []string) (*ResetOptions, error) {
	opts := &ResetOptions{
		Mode:   gogit.MixedReset,
		Target: "HEAD",
	}
	cmdArgs := args[1:]

	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "--soft":
			opts.Mode = gogit.SoftReset
		case "--mixed":
			opts.Mode = gogit.MixedReset
		case "--hard":
			opts.Mode = gogit.HardReset
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			opts.Target = arg
		}
	}
	return opts, nil
}

func (c *ResetCommand) executeReset(s *git.Session, w *gogit.Worktree, targetHash *plumbing.Hash, opts *ResetOptions) (string, error) {
	// Update ORIG_HEAD before reset
	s.UpdateOrigHead()

	if err := w.Reset(&gogit.ResetOptions{
		Commit: *targetHash,
		Mode:   opts.Mode,
	}); err != nil {
		return "", err
	}
	s.RecordReflog(fmt.Sprintf("reset: moving to %s", opts.Target))

	return fmt.Sprintf("HEAD is now at %s", targetHash.String()[:7]), nil
}

func (c *ResetCommand) Help() string {
	return `📘 GIT-RESET (1)                                        Git Manual

 💡 DESCRIPTION
    ・コミットをなかったことにして、過去の状態に戻る（HEADを移動する）
    ・ステージングした変更を取り消す（Unstage）
    ・作業中の変更をすべて破棄して元に戻す（Hard Reset）
    オプションによって、インデックスやワーキングツリーの状態をどう扱うかが変わります。

 📋 SYNOPSIS
    git reset [--soft | --mixed | --hard] <commit>

 ⚙️  COMMON OPTIONS
    --soft
        HEADのみを移動します。インデックスとワーキングツリーは変更しません。
        （戻った分のコミット内容は「ステージング済み」として残ります）

    --mixed (default)
        HEADとインデックスを移動します。ワーキングツリーは変更しません。
        （戻った分のコミット内容は「未ステージ」として残ります）

    --hard
        HEAD、インデックス、ワーキングツリーすべてを強制的に移動します。
        未コミットの変更はすべて破棄されます。

 🛠  EXAMPLES
    1. 直前のコミットを取り消す（変更はそのまま残す）
       $ git reset HEAD~1

    2. 全てを強制的に以前の状態に戻す（危険）
       $ git reset --hard HEAD~1

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-reset
`
}
