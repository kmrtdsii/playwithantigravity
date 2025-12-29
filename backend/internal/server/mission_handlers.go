package server

import (
	"encoding/json"
	"net/http"
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
