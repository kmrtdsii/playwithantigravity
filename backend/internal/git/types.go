package git

// ReflogEntry represents an entry in the reflog
type ReflogEntry struct {
	Hash    string
	Message string
}

// GraphState represents the serialized state for the frontend
type GraphState struct {
	Commits        []Commit          `json:"commits"`
	Branches       map[string]string `json:"branches"`
	RemoteBranches map[string]string `json:"remoteBranches"`
	Tags           map[string]string `json:"tags"`
	References     map[string]string `json:"references"`
	HEAD           Head              `json:"HEAD"`
	Files          []string          `json:"files"`
	Staging        []string          `json:"staging"`
	Modified       []string          `json:"modified"`
	Untracked      []string          `json:"untracked"`
	FileStatuses   map[string]string `json:"fileStatuses"`
	CurrentPath    string            `json:"currentPath"`
	Projects       []string          `json:"projects"`
	Remotes        []Remote          `json:"remotes"`
}

type Remote struct {
	Name string   `json:"name"`
	URLs []string `json:"urls"`
}

type Commit struct {
	ID             string `json:"id"`
	Message        string `json:"message"`
	ParentID       string `json:"parentId"`
	SecondParentID string `json:"secondParentId"`
	Branch         string `json:"branch"` // Naive branch inference
	Timestamp      string `json:"timestamp"`
	TreeID         string `json:"treeId"`
}

type Head struct {
	Type string `json:"type"` // "branch" or "commit"
	Ref  string `json:"ref,omitempty"`
	ID   string `json:"id,omitempty"`
}

type BranchingStrategy struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	MainBranch  string   `json:"mainBranch"`
	FlowSteps   []string `json:"flowSteps"`
}
