package git

import "time"

type PullRequestStatus string

const (
	PROpen   PullRequestStatus = "OPEN"
	PRMerged PullRequestStatus = "MERGED"
	PRClosed PullRequestStatus = "CLOSED"
)

type PullRequest struct {
	ID           int               `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	SourceBranch string            `json:"sourceBranch"`
	TargetBranch string            `json:"targetBranch"`
	Status       PullRequestStatus `json:"status"`
	Creator      string            `json:"creator"`
	CreatedAt    time.Time         `json:"createdAt"`
}
