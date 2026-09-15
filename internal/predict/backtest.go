package predict

import (
	"sort"

	"github.com/Liban213/pitchside/internal/models"
)

// Report summarizes a walk-forward backtest.
type Report struct {
	Evaluated int
	Correct   int
}

// Accuracy returns the fraction of evaluated matches whose most-likely
// predicted outcome matched the actual result. Zero if nothing was
// evaluated (e.g. every match was its team's first, with no prior history).
func (r Report) Accuracy() float64 {
	if r.Evaluated == 0 {
		return 0
	}
	return float64(r.Correct) / float64(r.Evaluated)
}

// Backtest walk-forward evaluates the model against every finished match in
// history: matches are sorted chronologically, and each one is predicted
// using only the matches strictly before it, so no future result leaks into
// its own prediction. Matches with no prior history at all are skipped.
func Backtest(history []models.Match) Report {
	finished := make([]models.Match, 0, len(history))
	for _, m := range history {
		if m.Status == models.StatusFinished && m.HomeScore != nil && m.AwayScore != nil {
			finished = append(finished, m)
		}
	}
	sort.Slice(finished, func(i, j int) bool { return finished[i].Date.Before(finished[j].Date) })

	var report Report
	for i, m := range finished {
		outcome, err := Predict(finished[:i], m.HomeTeamID, m.AwayTeamID)
		if err != nil {
			continue
		}
		report.Evaluated++
		if outcome.MostLikely() == actualResult(m) {
			report.Correct++
		}
	}
	return report
}

func actualResult(m models.Match) Result {
	switch {
	case *m.HomeScore > *m.AwayScore:
		return ResultHomeWin
	case *m.HomeScore < *m.AwayScore:
		return ResultAwayWin
	default:
		return ResultDraw
	}
}
