// Package checkout provides strategy implementations for the git checkout command.
package checkout

import (
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

// Options represents parsed checkout command options.
type Options struct {
	NewBranch      string
	ForceNewBranch string
	OrphanBranch   string
	Force          bool
	Detach         bool
	Target         string
	Files          []string // For "git checkout -- <file>"
}

// Mode represents the checkout operation mode.
type Mode int

const (
	ModeInvalid Mode = iota
	ModeFiles
	ModeOrphan
	ModeNewBranch
	ModeRefOrPath
)

// Context holds resolved state for checkout execution.
type Context struct {
	Mode           Mode
	Worktree       *gogit.Worktree
	Repo           *gogit.Repository
	Files          []string
	OrphanBranch   string
	NewBranch      string
	ForceCreate    bool
	StartPointHash *plumbing.Hash
	TargetRef      plumbing.ReferenceName
	TargetHash     *plumbing.Hash
	IsDetached     bool
}

// Strategy defines the interface for checkout strategies.
type Strategy interface {
	Execute(s *git.Session, ctx *Context, opts *Options) (string, error)
}
