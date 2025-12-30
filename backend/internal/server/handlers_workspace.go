package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
)

// DirectoryNode represents a node in the workspace tree
type DirectoryNode struct {
	Path     string           `json:"path"`
	Name     string           `json:"name"`
	IsDir    bool             `json:"isDir"`
	IsRepo   bool             `json:"isRepo"`
	Branch   string           `json:"branch,omitempty"`
	Children []*DirectoryNode `json:"children,omitempty"`
}

// handleGetWorkspaceTree returns the directory tree structure with repo indicators
func (s *Server) handleGetWorkspaceTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.SessionManager.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.RLock()
	defer session.RUnlock()

	// Build tree from filesystem and repos
	tree := s.buildDirectoryTree(session)

	// Detect current repository
	currentRepo := s.detectCurrentRepo(session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tree":        tree,
		"currentPath": session.CurrentDir,
		"currentRepo": currentRepo,
	})
}

// buildDirectoryTree constructs the directory tree from session filesystem
func (s *Server) buildDirectoryTree(session *git.Session) []*DirectoryNode {
	// Create root node
	root := &DirectoryNode{
		Path:     "/",
		Name:     "/",
		IsDir:    true,
		Children: []*DirectoryNode{},
	}

	// Read root directory
	entries, err := session.Filesystem.ReadDir(".")
	if err != nil {
		return []*DirectoryNode{root}
	}

	// Build children recursively
	for _, entry := range entries {
		if entry.IsDir() {
			child := s.buildNodeRecursive(session, entry.Name(), "/"+entry.Name())
			root.Children = append(root.Children, child)
		}
	}

	// Sort children
	sortNodes(root.Children)

	return []*DirectoryNode{root}
}

// buildNodeRecursive builds a directory node and its children
func (s *Server) buildNodeRecursive(session *git.Session, fsPath string, displayPath string) *DirectoryNode {
	node := &DirectoryNode{
		Path:     displayPath,
		Name:     getBaseName(displayPath),
		IsDir:    true,
		Children: []*DirectoryNode{},
	}

	// Check if this path is a repository
	if repo, ok := session.Repos[fsPath]; ok {
		node.IsRepo = true
		node.Branch = getBranchFromRepo(repo)
	}

	// Read subdirectories
	entries, err := session.Filesystem.ReadDir(fsPath)
	if err != nil {
		return node
	}

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != ".git" {
			childFsPath := fsPath + "/" + entry.Name()
			childDisplayPath := displayPath + "/" + entry.Name()
			child := s.buildNodeRecursive(session, childFsPath, childDisplayPath)
			node.Children = append(node.Children, child)
		}
	}

	sortNodes(node.Children)
	return node
}

// detectCurrentRepo finds the repository that contains the current directory
func (s *Server) detectCurrentRepo(session *git.Session) string {
	currentDir := session.CurrentDir
	if currentDir == "/" {
		return ""
	}

	// Strip leading slash
	currentPath := strings.TrimPrefix(currentDir, "/")

	// Check if current path is a repo or inside a repo
	for repoPath := range session.Repos {
		repoPath = strings.TrimPrefix(repoPath, "/")
		if currentPath == repoPath || strings.HasPrefix(currentPath, repoPath+"/") {
			return "/" + repoPath
		}
	}

	return ""
}

// getBranchFromRepo extracts the current branch name from a repository
func getBranchFromRepo(repo *gogit.Repository) string {
	head, err := repo.Head()
	if err != nil {
		return "main" // Default for empty repos
	}
	return head.Name().Short()
}

// getBaseName returns the last component of a path
func getBaseName(path string) string {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return path
}

// sortNodes sorts directory nodes alphabetically
func sortNodes(nodes []*DirectoryNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
}
