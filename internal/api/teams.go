package api

import (
	"net/http"
	"strconv"

	"github.com/Liban213/pitchside/internal/predict"
)

type teamStatsResponse struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	MatchesPlayed   int     `json:"matches_played"`
	Wins            int     `json:"wins"`
	Draws           int     `json:"draws"`
	Losses          int     `json:"losses"`
	GoalsFor        int     `json:"goals_for"`
	GoalsAgainst    int     `json:"goals_against"`
	GoalDifference  int     `json:"goal_difference"`
	AttackStrength  float64 `json:"attack_strength"`
	DefenseStrength float64 `json:"defense_strength"`
}

func (s *Server) handleTeamStats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}

	team, err := s.teams.GetTeam(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}

	finished, err := s.matches.FinishedByTeam(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load team history")
		return
	}

	stats := teamStatsResponse{ID: team.ID, Name: team.Name, AttackStrength: 1, DefenseStrength: 1}
	for _, m := range finished {
		stats.MatchesPlayed++
		goalsFor, goalsAgainst := *m.HomeScore, *m.AwayScore
		if m.AwayTeamID == id {
			goalsFor, goalsAgainst = *m.AwayScore, *m.HomeScore
		}
		stats.GoalsFor += goalsFor
		stats.GoalsAgainst += goalsAgainst
		switch {
		case goalsFor > goalsAgainst:
			stats.Wins++
		case goalsFor == goalsAgainst:
			stats.Draws++
		default:
			stats.Losses++
		}
	}
	stats.GoalDifference = stats.GoalsFor - stats.GoalsAgainst

	// League-wide strength needs every finished match, not just this
	// team's — attack/defense are always relative to the league average.
	league, err := s.matches.ListFinished(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load league history")
		return
	}
	if attack, defense, err := predict.Strength(league, id); err == nil {
		stats.AttackStrength = attack
		stats.DefenseStrength = defense
	}

	writeJSON(w, http.StatusOK, stats)
}
