// Package predict computes win/draw/loss probabilities for a fixture from
// each team's attack/defense strength, derived from finished matches, fed
// into independent Poisson distributions over expected goals.
package predict

import (
	"errors"

	"github.com/Liban213/pitchside/internal/models"
)

// Result is a match outcome from the home team's perspective.
type Result string

const (
	ResultHomeWin Result = "HOME_WIN"
	ResultDraw    Result = "DRAW"
	ResultAwayWin Result = "AWAY_WIN"
)

// Outcome holds win/draw/loss probabilities for one fixture; they sum to ~1.
type Outcome struct {
	HomeWin float64
	Draw    float64
	AwayWin float64
}

// MostLikely returns the outcome with the highest probability, breaking ties
// in favor of the more decisive result over a draw.
func (o Outcome) MostLikely() Result {
	switch {
	case o.HomeWin >= o.Draw && o.HomeWin >= o.AwayWin:
		return ResultHomeWin
	case o.AwayWin >= o.Draw:
		return ResultAwayWin
	default:
		return ResultDraw
	}
}

// ErrInsufficientHistory is returned when history has no finished matches to
// derive league scoring rates from.
var ErrInsufficientHistory = errors.New("predict: no finished matches in history")

// maxGoals bounds the score grid summed over. Early-season strength ratios
// can push expected goals well above realistic single-match totals, so this
// is set generously rather than tuned to typical football scorelines.
const maxGoals = 20

// Predict estimates win/draw/loss probabilities for homeTeamID vs
// awayTeamID, using attack/defense strength computed from history.
func Predict(history []models.Match, homeTeamID, awayTeamID int) (Outcome, error) {
	l := newLeague(history)
	if l.avgPerTeam == 0 {
		return Outcome{}, ErrInsufficientHistory
	}
	homeXG, awayXG := l.expectedGoals(homeTeamID, awayTeamID)
	return outcomeFromExpectedGoals(homeXG, awayXG), nil
}

func outcomeFromExpectedGoals(homeXG, awayXG float64) Outcome {
	var o Outcome
	for i := 0; i <= maxGoals; i++ {
		ph := poissonPMF(homeXG, i)
		for j := 0; j <= maxGoals; j++ {
			p := ph * poissonPMF(awayXG, j)
			switch {
			case i > j:
				o.HomeWin += p
			case i == j:
				o.Draw += p
			default:
				o.AwayWin += p
			}
		}
	}
	return o
}
