package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/kurobon/gitgym/backend/internal/mission"
)

func TestHandleCreateRemote(t *testing.T) {
	// Setup Dependencies
	tmpDir := t.TempDir()
	t.Setenv("GITGYM_DATA_ROOT", tmpDir)

	sm := git.NewSessionManager()
	ml := mission.NewLoader(tmpDir)
	me := mission.NewEngine(ml, sm)
	s := NewServer(sm, me)

	// Create a dummy session
	sessionID := "test-session"
	_, err := sm.CreateSession(sessionID)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		repoName := "valid-repo"
		body := map[string]string{"name": repoName}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/api/remote/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("X-Session-ID", sessionID)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]string
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)

		assert.Equal(t, repoName, resp["name"])
		assert.Equal(t, fmt.Sprintf("remote://gitgym/%s.git", repoName), resp["remoteUrl"])

		// Verify repo created in SM
		sm.RLock()
		_, exists := sm.SharedRemotes[repoName]
		sm.RUnlock()
		assert.True(t, exists)
	})

	t.Run("Validation Error", func(t *testing.T) {
		body := map[string]string{"name": "Invalid Name!"}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/api/remote/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("X-Session-ID", sessionID)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid repository name")
	})

	t.Run("Missing Session ID", func(t *testing.T) {
		body := map[string]string{"name": "valid-repo-2"}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/api/remote/create", bytes.NewBuffer(jsonBody))
		// No Header
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Session ID required")
	})
}
