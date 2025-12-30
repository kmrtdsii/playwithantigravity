package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kurobon/gitgym/backend/internal/git"
	_ "github.com/kurobon/gitgym/backend/internal/git/commands" // Register commands
	"github.com/kurobon/gitgym/backend/internal/mission"
	"github.com/kurobon/gitgym/backend/internal/server"
)

// DefaultRemoteURL is the pre-configured remote repository available for cloning
const DefaultRemoteURL = "https://github.com/octocat/Spoon-Knife.git"

// DefaultDataDir is the default directory for storing persistent data
const DefaultDataDir = ".gitgym-data"

// getDataDir returns the data directory path, configurable via GITGYM_DATA_ROOT env var
func getDataDir() string {
	if dir := os.Getenv("GITGYM_DATA_ROOT"); dir != "" {
		return dir
	}
	return DefaultDataDir
}

func main() {
	dataDir := getDataDir()
	// Check if CLEAR_REMOTES_ON_START is set to clear the remotes directory
	if os.Getenv("CLEAR_REMOTES_ON_START") == "true" {
		remotesDir := dataDir + "/remotes"
		log.Printf("CLEAR_REMOTES_ON_START is set, clearing %s", remotesDir)
		if err := os.RemoveAll(remotesDir); err != nil {
			log.Printf("Warning: Failed to clear remotes directory: %v", err)
		}
	}

	// Initialize Core Dependencies
	sessionManager := git.NewSessionManager()

	// Initialize Mission Engine
	// We put missions in "missions" directory relative to binary? Or distinct dir.
	// Assume "missions" dir in CWD (backend root).
	missionLoader := mission.NewLoader("missions")
	missionEngine := mission.NewEngine(missionLoader, sessionManager)

	// Pre-ingest default remote repository asynchronously
	// Default remote initialization removed at user request
	// go func() { ... }()

	// Initialize HTTP Server
	srv := server.NewServer(sessionManager, missionEngine)

	// Security: Use http.Server with timeouts (G114)
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      srv,
		ReadTimeout:  300 * time.Second, // Increased for large repo operations
		WriteTimeout: 300 * time.Second, // Increased for large repo operations
		IdleTimeout:  300 * time.Second,
	}

	log.Println("Server listening on :8080")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
