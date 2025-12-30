package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/mission"
)

type StartMissionRequest struct {
	MissionID string `json:"missionId"`
}

type StartMissionResponse struct {
	SessionID string `json:"sessionId"`
	MissionID string `json:"missionId"`
}

type VerifyMissionRequest struct {
	SessionID string `json:"sessionId"`
	MissionID string `json:"missionId"`
}

func (s *Server) handleListMissions(w http.ResponseWriter, r *http.Request) {
	missions, err := s.MissionEngine.Loader.ListMissions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Localization Logic
	lang := "en" // default
	acceptLang := r.Header.Get("Accept-Language")
	// Simple detection for Japanese. For production, consider using x/text/language.
	if strings.Contains(strings.ToLower(acceptLang), "ja") {
		lang = "ja"
	}

	if lang != "en" {
		localizedMissions := make([]*mission.Mission, len(missions))
		for i, m := range missions {
			// Copy the mission struct by dereferencing
			val := *m
			localized := &val

			if trans, ok := m.Translations[lang]; ok {
				if trans.Title != "" {
					localized.Title = trans.Title
				}
				if trans.Description != "" {
					localized.Description = trans.Description
				}
				if len(trans.Hints) > 0 {
					localized.Hints = trans.Hints
				}
			}
			localizedMissions[i] = localized
		}
		missions = localizedMissions
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(missions)
}

func (s *Server) handleStartMission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartMissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sessionID, err := s.MissionEngine.StartMission(r.Context(), req.MissionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StartMissionResponse{
		SessionID: sessionID,
		MissionID: req.MissionID,
	})
}

func (s *Server) handleVerifyMission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyMissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := s.MissionEngine.VerifyMission(req.SessionID, req.MissionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
