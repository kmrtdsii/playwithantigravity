package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// RepoInfo contains metadata about a GitHub repository
type RepoInfo struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Size          int    `json:"size"` // Size in KB
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
}

// CloneEstimate contains estimated clone time information
type CloneEstimate struct {
	RepoInfo         *RepoInfo     `json:"repoInfo"`
	EstimatedSeconds int           `json:"estimatedSeconds"`
	SizeDisplay      string        `json:"sizeDisplay"`
	Message          string        `json:"message"`
}

// FetchRepoInfo retrieves repository information from GitHub API
func FetchRepoInfo(url string) (*RepoInfo, error) {
	// Parse GitHub URL to extract owner/repo
	owner, repo, err := parseGitHubURL(url)
	if err != nil {
		return nil, err
	}

	// Construct GitHub API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent (required by GitHub API)
	req.Header.Set("User-Agent", "GitGym/1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repo info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("repository not found: %s/%s", owner, repo)
	}

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded. Please wait and try again.")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	var info RepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return &info, nil
}

// EstimateCloneTime estimates the time needed to clone a repository
func EstimateCloneTime(sizeKB int) time.Duration {
	// With Depth: 50 and Tags: NoTags, we estimate fetching ~30% of full repo
	effectiveSize := float64(sizeKB) * 0.3

	// Conservative estimate: 1 MB ≈ 1 second on typical broadband
	// Minimum 2 seconds, maximum 5 minutes
	seconds := effectiveSize / 1024.0
	if seconds < 2 {
		seconds = 2
	}
	if seconds > 300 {
		seconds = 300
	}

	return time.Duration(seconds) * time.Second
}

// GetCloneEstimate fetches repo info and calculates clone time estimate
func GetCloneEstimate(url string) (*CloneEstimate, error) {
	info, err := FetchRepoInfo(url)
	if err != nil {
		return nil, err
	}

	estimated := EstimateCloneTime(info.Size)

	// Format size for display
	sizeDisplay := formatSize(info.Size)

	// Create message based on size
	var message string
	if estimated > 60*time.Second {
		message = fmt.Sprintf("Large repository (%s). This may take a while.", sizeDisplay)
	} else if estimated > 30*time.Second {
		message = fmt.Sprintf("Medium repository (%s).", sizeDisplay)
	} else {
		message = fmt.Sprintf("Small repository (%s). Quick clone expected.", sizeDisplay)
	}

	return &CloneEstimate{
		RepoInfo:         info,
		EstimatedSeconds: int(estimated.Seconds()),
		SizeDisplay:      sizeDisplay,
		Message:          message,
	}, nil
}

// parseGitHubURL extracts owner and repo name from a GitHub URL
func parseGitHubURL(url string) (string, string, error) {
	// Clean the URL
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, ".git")

	// Pattern: https://github.com/owner/repo or git@github.com:owner/repo
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`github\.com[/:]([^/]+)/([^/]+)$`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(url)
		if len(matches) == 3 {
			return matches[1], matches[2], nil
		}
	}

	return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
}

// formatSize converts KB to human-readable format
func formatSize(sizeKB int) string {
	if sizeKB < 1024 {
		return fmt.Sprintf("%d KB", sizeKB)
	}
	sizeMB := float64(sizeKB) / 1024.0
	if sizeMB < 1024 {
		return fmt.Sprintf("%.1f MB", sizeMB)
	}
	sizeGB := sizeMB / 1024.0
	return fmt.Sprintf("%.2f GB", sizeGB)
}
