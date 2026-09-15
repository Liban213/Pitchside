package predict

import (
	"testing"
	"time"

	"github.com/Liban213/pitchside/internal/models"
)

func TestBacktest_SkipsFirstEverMatch(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	report := Backtest([]models.Match{finishedMatch(day, 1, 2, 1, 0)})
	if report.Evaluated != 0 {
		t.Fatalf("expected the only match (no prior history) to be skipped, got Evaluated=%d", report.Evaluated)
	}
}

func TestBacktest_IgnoresUnfinishedMatches(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	scheduled := models.Match{
		HomeTeamID: 1,
		AwayTeamID: 2,
		Date:       day,
		Status:     models.StatusScheduled,
	}
	report := Backtest([]models.Match{scheduled})
	if report.Evaluated != 0 {
		t.Fatalf("expected no matches evaluated from unfinished-only history, got %d", report.Evaluated)
	}
}

func TestBacktest_EvaluatesEveryMatchAfterTheFirst(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const strong, weak = 1, 2

	var history []models.Match
	for i := 0; i < 8; i++ {
		date := day.AddDate(0, 0, i)
		if i%2 == 0 {
			history = append(history, finishedMatch(date, strong, weak, 3, 0))
		} else {
			history = append(history, finishedMatch(date, weak, strong, 0, 3))
		}
	}

	report := Backtest(history)
	if report.Evaluated != len(history)-1 {
		t.Fatalf("expected %d evaluated matches, got %d", len(history)-1, report.Evaluated)
	}
	if report.Correct > report.Evaluated {
		t.Fatalf("Correct (%d) exceeds Evaluated (%d)", report.Correct, report.Evaluated)
	}
	// A team that wins every single match by three goals should be an easy
	// pick once it has any track record at all.
	if acc := report.Accuracy(); acc < 0.7 {
		t.Fatalf("expected backtest accuracy >= 0.7 for a lopsided series, got %v", acc)
	}
}

func TestReport_Accuracy(t *testing.T) {
	tests := []struct {
		name   string
		report Report
		want   float64
	}{
		{"no matches evaluated", Report{}, 0},
		{"all correct", Report{Evaluated: 4, Correct: 4}, 1},
		{"half correct", Report{Evaluated: 4, Correct: 2}, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.report.Accuracy(); got != tt.want {
				t.Fatalf("Accuracy() = %v, want %v", got, tt.want)
			}
		})
	}
}
