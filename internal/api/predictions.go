package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Liban213/pitchside/internal/models"
	"github.com/Liban213/pitchside/internal/predict"
)

type predictionResponse struct {
	MatchID   int     `json:"match_id"`
	HomeTeam  teamRef `json:"home_team"`
	AwayTeam  teamRef `json:"away_team"`
	HomeWin   float64 `json:"home_win"`
	Draw      float64 `json:"draw"`
	AwayWin   float64 `json:"away_win"`
	Predicted string  `json:"predicted"`
}

func (s *Server) handlePrediction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("matchId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match id")
		return
	}

	match, err := s.matches.GetMatch(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "match not found")
		return
	}

	history, err := s.matches.ListFinished(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load match history")
		return
	}
	history = excludeMatch(history, match.ID)

	outcome, err := predict.Predict(history, match.HomeTeamID, match.AwayTeamID)
	if err != nil {
		if errors.Is(err, predict.ErrInsufficientHistory) {
			writeError(w, http.StatusUnprocessableEntity, "not enough historical data to generate a prediction yet")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to generate prediction")
		return
	}

	home, err := s.teams.GetTeam(r.Context(), match.HomeTeamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load home team")
		return
	}
	away, err := s.teams.GetTeam(r.Context(), match.AwayTeamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load away team")
		return
	}

	writeJSON(w, http.StatusOK, predictionResponse{
		MatchID:   match.ID,
		HomeTeam:  teamRef{home.ID, home.Name},
		AwayTeam:  teamRef{away.ID, away.Name},
		HomeWin:   outcome.HomeWin,
		Draw:      outcome.Draw,
		AwayWin:   outcome.AwayWin,
		Predicted: string(outcome.MostLikely()),
	})
}

// excludeMatch drops matchID from history so a match never leaks into its
// own prediction (relevant when predicting an already-finished match).
func excludeMatch(history []models.Match, matchID int) []models.Match {
	filtered := make([]models.Match, 0, len(history))
	for _, m := range history {
		if m.ID != matchID {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
