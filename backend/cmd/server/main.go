package main

import (
	"context"
	"log"
	"net/http"

	"github.com/kurobon/gitgym/backend/internal/git"
	_ "github.com/kurobon/gitgym/backend/internal/git/commands" // Register commands
	"github.com/kurobon/gitgym/backend/internal/server"
)

// DefaultRemoteURL is the pre-configured remote repository available for cloning
const DefaultRemoteURL = "https://github.com/git-fixtures/basic.git"

func main() {
	// Initialize Core Dependencies
	sessionManager := git.NewSessionManager()

	// Pre-ingest default remote repository so users can immediately clone
	log.Printf("Initializing default remote: %s", DefaultRemoteURL)
	if err := sessionManager.IngestRemote(context.Background(), "origin", DefaultRemoteURL); err != nil {
		log.Printf("Warning: Failed to ingest default remote: %v", err)
		log.Println("Users will need to configure a remote manually via /api/remote/ingest")
	} else {
		log.Println("Default remote 'origin' ready for cloning")
	}

	// Initialize HTTP Server
	srv := server.NewServer(sessionManager)

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}
}
