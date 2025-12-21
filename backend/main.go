package main

import (
	"log"
	"net/http"

	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
	_ "github.com/kmrtdsii/playwithantigravity/backend/pkg/git/commands" // Register commands
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/server"
)

func main() {
	// Initialize Core Dependencies
	sessionManager := git.NewSessionManager()
	
	// Initialize HTTP Server
	srv := server.NewServer(sessionManager)

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}
}
