package server

import (
	"net/http"

	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/kurobon/gitgym/backend/internal/mission"
)

type Server struct {
	SessionManager *git.SessionManager
	MissionEngine  *mission.Engine
	Mux            *http.ServeMux
}

func NewServer(sm *git.SessionManager, me *mission.Engine) *Server {
	s := &Server{
		SessionManager: sm,
		MissionEngine:  me,
		Mux:            http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.Mux.HandleFunc("/ping", s.handlePing)
	s.Mux.HandleFunc("/api/session/init", s.handleInitSession)
	s.Mux.HandleFunc("/api/command", s.handleExecCommand)
	s.Mux.HandleFunc("/api/state", s.handleGetGraphState)
	s.Mux.HandleFunc("/api/remote/state", s.handleGetRemoteState)
	s.Mux.HandleFunc("/api/strategies", s.handleGetStrategies)

	// Remote / Simulation
	s.Mux.HandleFunc("/api/remote/ingest", s.handleIngestRemote)
	s.Mux.HandleFunc("/api/remote/simulate-commit", s.handleSimulateRemoteCommit)
	s.Mux.HandleFunc("/api/remote/pull-requests", s.handleGetPullRequests)
	s.Mux.HandleFunc("/api/remote/pull-requests/create", s.handleCreatePullRequest)
	s.Mux.HandleFunc("/api/remote/pull-requests/merge", s.handleMergePullRequest)
	s.Mux.HandleFunc("/api/remote/pull-requests/delete", s.handleDeletePullRequest)
	s.Mux.HandleFunc("/api/remote/reset", s.handleResetRemote)
	s.Mux.HandleFunc("/api/remote/info", s.handleGetRemoteInfo)
	s.Mux.HandleFunc("/api/remote/create", s.handleCreateRemote)

	// Mission
	s.Mux.HandleFunc("/api/mission/list", s.handleListMissions)
	s.Mux.HandleFunc("/api/mission/start", s.handleStartMission)
	s.Mux.HandleFunc("/api/mission/verify", s.handleVerifyMission)

	// Workspace
	s.Mux.HandleFunc("/api/workspace/tree", s.handleGetWorkspaceTree)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply global middleware: CORS -> Logger -> Recoverer -> Mux
	handler := Chain(s.Mux, CORS, Logger, Recoverer)
	handler.ServeHTTP(w, r)
}
