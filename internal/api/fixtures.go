package api

import (
	"net/http"
	"time"

	"github.com/Liban213/pitchside/internal/models"
)

type fixtureResponse struct {
	ID        int       `json:"id"`
	Date      time.Time `json:"date"`
	Status    string    `json:"status"`
	HomeTeam  teamRef   `json:"home_team"`
	AwayTeam  teamRef   `json:"away_team"`
	HomeScore *int      `json:"home_score"`
	AwayScore *int      `json:"away_score"`
}

// handleFixtures lists matches in a date range, defaulting to the last
// week through the next month. ?from= and ?to= override it (YYYY-MM-DD).
func (s *Server) handleFixtures(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, 0, -7)
	to := time.Now().AddDate(0, 0, 30)

	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid from date, expected YYYY-MM-DD")
			return
		}
		from = t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid to date, expected YYYY-MM-DD")
			return
		}
		to = t
	}

	matches, err := s.matches.ListByDateRange(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list fixtures")
		return
	}

	teams, err := s.teams.ListTeams(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load teams")
		return
	}
	byID := make(map[int]models.Team, len(teams))
	for _, t := range teams {
		byID[t.ID] = t
	}

	resp := make([]fixtureResponse, 0, len(matches))
	for _, m := range matches {
		resp = append(resp, fixtureResponse{
			ID:        m.ID,
			Date:      m.Date,
			Status:    string(m.Status),
			HomeTeam:  teamRef{byID[m.HomeTeamID].ID, byID[m.HomeTeamID].Name},
			AwayTeam:  teamRef{byID[m.AwayTeamID].ID, byID[m.AwayTeamID].Name},
			HomeScore: m.HomeScore,
			AwayScore: m.AwayScore,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}
