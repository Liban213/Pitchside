// Package api implements Pitchside's REST endpoints: fixtures, predictions,
// and team stats, all served over the stores built in internal/store.
package api

import (
	"net/http"

	"github.com/Liban213/pitchside/internal/store"
	"github.com/Liban213/pitchside/internal/ws"
)

type Server struct {
	teams   *store.TeamStore
	matches *store.MatchStore
	mux     *http.ServeMux
}

func NewServer(teams *store.TeamStore, matches *store.MatchStore, hub *ws.Hub) *Server {
	s := &Server{teams: teams, matches: matches, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/fixtures", s.handleFixtures)
	s.mux.HandleFunc("GET /api/predictions/{matchId}", s.handlePrediction)
	s.mux.HandleFunc("GET /api/teams/{id}/stats", s.handleTeamStats)
	s.mux.Handle("GET /ws", hub)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

type teamRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
