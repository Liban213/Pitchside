package predict

import (
	"math"
	"testing"
	"time"

	"github.com/Liban213/pitchside/internal/models"
)

func finishedMatch(date time.Time, homeID, awayID, homeScore, awayScore int) models.Match {
	hs, as := homeScore, awayScore
	return models.Match{
		HomeTeamID: homeID,
		AwayTeamID: awayID,
		Date:       date,
		Status:     models.StatusFinished,
		HomeScore:  &hs,
		AwayScore:  &as,
	}
}

func TestPredict_NoHistoryReturnsError(t *testing.T) {
	_, err := Predict(nil, 1, 2)
	if err != ErrInsufficientHistory {
		t.Fatalf("expected ErrInsufficientHistory, got %v", err)
	}
}

func TestPredict_ProbabilitiesSumToOne(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	history := []models.Match{
		finishedMatch(day, 1, 2, 3, 0),
		finishedMatch(day.AddDate(0, 0, 1), 2, 1, 0, 2),
		finishedMatch(day.AddDate(0, 0, 2), 1, 3, 4, 1),
	}
	outcome, err := Predict(history, 1, 2)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	sum := outcome.HomeWin + outcome.Draw + outcome.AwayWin
	if math.Abs(sum-1) > 1e-4 {
		t.Fatalf("expected probabilities to sum to ~1, got %v (%+v)", sum, outcome)
	}
}

func TestPredict_DominantTeamFavoredRegardlessOfVenue(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const strong, weak = 1, 2

	var history []models.Match
	for i := 0; i < 6; i++ {
		date := day.AddDate(0, 0, i)
		if i%2 == 0 {
			history = append(history, finishedMatch(date, strong, weak, 3, 0))
		} else {
			history = append(history, finishedMatch(date, weak, strong, 0, 3))
		}
	}

	homeOutcome, err := Predict(history, strong, weak)
	if err != nil {
		t.Fatalf("Predict (strong at home): %v", err)
	}
	if homeOutcome.MostLikely() != ResultHomeWin {
		t.Fatalf("expected strong team favored at home, got %+v", homeOutcome)
	}

	awayOutcome, err := Predict(history, weak, strong)
	if err != nil {
		t.Fatalf("Predict (strong away): %v", err)
	}
	if awayOutcome.MostLikely() != ResultAwayWin {
		t.Fatalf("expected strong team favored away, got %+v", awayOutcome)
	}
}

func TestOutcome_MostLikely(t *testing.T) {
	tests := []struct {
		name    string
		outcome Outcome
		want    Result
	}{
		{"home win", Outcome{HomeWin: 0.6, Draw: 0.25, AwayWin: 0.15}, ResultHomeWin},
		{"draw", Outcome{HomeWin: 0.2, Draw: 0.5, AwayWin: 0.3}, ResultDraw},
		{"away win", Outcome{HomeWin: 0.1, Draw: 0.2, AwayWin: 0.7}, ResultAwayWin},
		{"tie favors home over draw", Outcome{HomeWin: 0.4, Draw: 0.4, AwayWin: 0.2}, ResultHomeWin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.outcome.MostLikely(); got != tt.want {
				t.Fatalf("MostLikely() = %v, want %v", got, tt.want)
			}
		})
	}
}
