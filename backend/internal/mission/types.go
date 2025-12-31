package mission

// Mission defines the structure of a practice mission loaded from YAML.
type Mission struct {
	ID           string                        `yaml:"id" json:"id"`
	Title        string                        `yaml:"title" json:"title"`
	Description  string                        `yaml:"description" json:"description"`
	Difficulty   Difficulty                    `yaml:"difficulty" json:"difficulty"`
	Skill        string                        `yaml:"skill" json:"skill"`
	Setup        []string                      `yaml:"setup" json:"-"`         // Commands to run for setup
	Validation   Validation                    `yaml:"validation" json:"-"`    // Validation rules
	Hints        []string                      `yaml:"hints" json:"hints"`     // Hints for the user
	Scoring      Scoring                       `yaml:"scoring" json:"scoring"` // Scoring rules
	Translations map[string]MissionTranslation `yaml:"translations" json:"-"`  // Localized content
}

type MissionTranslation struct {
	Title       string   `yaml:"title" json:"title"`
	Description string   `yaml:"description" json:"description"`
	Hints       []string `yaml:"hints" json:"hints"`
}

type Difficulty struct {
	Level string `yaml:"level" json:"level"` // basic, intermediate, advanced, etc.
	Stars int    `yaml:"stars" json:"stars"` // 1-5
}

type Validation struct {
	Checks []Check `yaml:"checks"`
}

type Check struct {
	Type           string   `yaml:"type"`            // no_conflict, commit_exists, file_content, file_tracked, clean_working_tree, branch_exists, current_branch
	Description    string   `yaml:"description"`     // User facing description
	MessagePattern string   `yaml:"message_pattern"` // For log checks
	Path           string   `yaml:"path"`            // For file checks
	Contains       []string `yaml:"contains"`        // For file content checks
	Name           string   `yaml:"name"`            // For branch checks (branch_exists, current_branch)
	Negate         bool     `yaml:"negate"`          // If true, inverts the pass condition
}

type Scoring struct {
	TimeBonus   bool `yaml:"time_bonus" json:"time_bonus"`
	HintPenalty int  `yaml:"hint_penalty" json:"hint_penalty"`
}

// MissionState tracks the user's progress in a specific mission session.
type MissionState struct {
	MissionID string `json:"missionId"`
	Status    string `json:"status"` // active, completed, failed
	Score     int    `json:"score"`
}
