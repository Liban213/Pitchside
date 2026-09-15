package footballdata

import (
	"context"
	"time"

	"github.com/Liban213/pitchside/internal/models"
)

// Fake is a test double for Client that records how many times each method
// was called, so tests can assert the cache actually prevents upstream
// calls on a hit.
type Fake struct {
	Teams        []models.Team
	Matches      []models.Match
	TeamsCalls   int
	MatchesCalls int
	Err          error
}

func (f *Fake) FetchTeams(ctx context.Context) ([]models.Team, error) {
	f.TeamsCalls++
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Teams, nil
}

func (f *Fake) FetchMatches(ctx context.Context, dateFrom, dateTo time.Time) ([]models.Match, error) {
	f.MatchesCalls++
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Matches, nil
}
