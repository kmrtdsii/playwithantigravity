package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
	_ "github.com/kmrtdsii/playwithantigravity/backend/pkg/git/commands" // Register commands
)

var sessionManager *git.SessionManager

func main() {
	sessionManager = git.NewSessionManager()
	
	mux := http.NewServeMux()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "pong",
			"system":  "GitForge Backend (Refactored)",
		})
	})

	// Git Engine API
	mux.HandleFunc("/api/session/init", initSession)
	mux.HandleFunc("/api/command", execCommand)
	mux.HandleFunc("/api/state", getGraphState)

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func initSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Generate simple ID
	sessionID := "user-session-1" // Fixed for now

	if _, err := sessionManager.CreateSession(sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "session created",
		"sessionId": sessionID,
	})
}

type CommandRequest struct {
	SessionID string `json:"sessionId"`
	Command   string `json:"command"`
}

func execCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = "user-session-1" // Default for testing
	}

	// Naive split of command string
	parts := strings.Fields(req.Command)
	
	// Handle Shortcuts
	if len(parts) > 0 {
		switch parts[0] {
		case "reset": // git reset
			newParts := []string{"git", "reset"}
			parts = append(newParts, parts[1:]...)
		case "add": // git add
			newParts := []string{"git", "add"}
			parts = append(newParts, parts[1:]...)
		case "commit": // git commit
			newParts := []string{"git", "commit", "-m"}
			parts = append(newParts, parts[1:]...)
		case "merge": // git merge
			newParts := []string{"git", "merge"}
			parts = append(newParts, parts[1:]...)
		case "tag": // git tag
			newParts := []string{"git", "tag"}
			parts = append(newParts, parts[1:]...)
		case "rebase": // git rebase
			newParts := []string{"git", "rebase"}
			parts = append(newParts, parts[1:]...)
		case "checkout": // git checkout
			newParts := []string{"git", "checkout"}
			parts = append(newParts, parts[1:]...)
		case "branch": // git branch
			newParts := []string{"git", "branch"}
			parts = append(newParts, parts[1:]...)
		case "switch": // git switch
			newParts := []string{"git", "switch"}
			parts = append(newParts, parts[1:]...)
		}
	}

	log.Printf("Command received: user=%s cmd=%s parts=%v", req.SessionID, req.Command, parts)
	
	if len(parts) > 0 {
		if parts[0] == "git" {
			session, err := sessionManager.GetSession(req.SessionID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			
			cmdName := ""
			args := []string{}
			if len(parts) > 1 {
				cmdName = parts[1]
				args = parts[1:] // Note: args usually include the command name or not? 
				// git.Dispatch: Execute(ctx, session, args)
				// Arg parsing in commands usually expects args[0] to be the command ("add", "commit") 
				// or maybe "init". 
				// My implementations checked args[0] in some cases?
				// add.go: file := args[1] (implies args[0] is "add")
				// init.go: checks nothing.
				// branch.go: args[1]
				// So yes, args should include the command name at index 0.
			}
			
			// parts = ["git", "add", "."] -> args = ["add", "."]
			
			output, err := git.Dispatch(r.Context(), session, cmdName, args)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"output": output})
			return

		} else if parts[0] == "touch" {
			if len(parts) < 2 {
				http.Error(w, "Filename required", http.StatusBadRequest)
				return
			}
			err := sessionManager.TouchFile(req.SessionID, parts[1])
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"output": "File updated"})
			return

		} else if parts[0] == "ls" {
			output, err := sessionManager.ListFiles(req.SessionID)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"output": output})
			return
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"error": "Only git commands supported right now"})
}

func getGraphState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		sessionID = "user-session-1" // Default
	}
	
	showAll := r.URL.Query().Get("showAll") == "true"

	state, err := sessionManager.GetGraphState(sessionID, showAll)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}
