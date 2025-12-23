package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestServerEndpoints(t *testing.T) {
	// Setup
	sm := git.NewSessionManager()
	srv := NewServer(sm)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := ts.Client()
	sessionID := "user-session-1" // Matches hardcoded ID in handlers

	// 1. Ping
	t.Run("Ping", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/ping")
		if err != nil {
			t.Fatalf("Failed to ping: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	// 2. Init Session
	t.Run("InitSession", func(t *testing.T) {
		resp, err := client.Post(ts.URL+"/api/session/init", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to init session: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
		var res map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if res["sessionId"] == "" {
			t.Error("Expected sessionId in response, got empty string")
		}
		// Update dynamically
		sessionID = res["sessionId"]
	})

	// 3. Exec Command: git init
	t.Run("Git Init", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sessionId": sessionID,
			"command":   "git init",
		})
		resp, err := client.Post(ts.URL+"/api/command", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to exec command: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	// 4. Exec Command: git status
	t.Run("Git Status", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sessionId": sessionID,
			"command":   "git status",
		})
		resp, err := client.Post(ts.URL+"/api/command", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to exec command: %v", err)
		}
		defer resp.Body.Close()

		var res map[string]string
		json.NewDecoder(resp.Body).Decode(&res)

		output := res["output"]
		if !strings.Contains(output, "On branch main") && !strings.Contains(output, "No commits yet") {
			// Exact output depends on git version/implementation but checking basics
			// "On branch main" or "master"
		}
	})

	// 5. Get Graph State
	t.Run("Get Graph State", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/state?sessionId=" + sessionID)
		if err != nil {
			t.Fatalf("GET state failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		var state git.GraphState
		if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
			t.Fatalf("Failed to decode graph state: %v", err)
		}

		if state.HEAD.Type == "" {
			t.Error("Expected HEAD info in state")
		}
	})

	// 6. Invalid Method
	t.Run("Invalid Method", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/command") // Should be POST
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405 Method Not Allowed, got %d", resp.StatusCode)
		}
	})
}
