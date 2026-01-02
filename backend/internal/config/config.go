// Package config provides centralized configuration for the GitGym backend.
package config

import (
	"os"
	"path/filepath"
)

// Config holds application-wide configuration.
type Config struct {
	// DataRoot is the base directory for persistent data (cloned remotes, etc.)
	DataRoot string
}

// DefaultConfig returns the default configuration, reading from environment variables.
func DefaultConfig() *Config {
	dataRoot := os.Getenv("GITGYM_DATA_ROOT")
	if dataRoot == "" {
		dataRoot = ".gitgym-data"
	}
	return &Config{
		DataRoot: dataRoot,
	}
}

// RemotesDir returns the path for storing remote repositories.
func (c *Config) RemotesDir() string {
	return filepath.Join(c.DataRoot, "remotes")
}

// Global is the application-wide configuration instance.
var Global = DefaultConfig()
