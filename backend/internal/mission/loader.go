package mission

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader handles loading missions from the filesystem.
type Loader struct {
	MissionDir string
}

func NewLoader(dir string) *Loader {
	return &Loader{MissionDir: dir}
}

// LoadMission loads a single mission by ID (filename without extension).
func (l *Loader) LoadMission(id string) (*Mission, error) {
	path := filepath.Join(l.MissionDir, id+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mission file: %w", err)
	}

	var m Mission
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse mission yaml: %w", err)
	}

	// Ensure ID matches filename if not set
	if m.ID == "" {
		m.ID = id
	}

	return &m, nil
}

// ListMissions returns all available missions.
func (l *Loader) ListMissions() ([]*Mission, error) {
	files, err := os.ReadDir(l.MissionDir)
	if err != nil {
		return nil, err
	}

	var missions []*Mission
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".yaml" {
			id := f.Name()[0 : len(f.Name())-len(".yaml")]
			m, err := l.LoadMission(id)
			if err != nil {
				// Log error but continue? For now, skip invalid files.
				continue
			}
			missions = append(missions, m)
		}
	}
	return missions, nil
}
