// Package footballdata talks to the football-data.org API for Premier
// League teams and matches. It is defined behind the Client interface so
// callers (the caching layer, tests) never depend on the concrete HTTP
// implementation.
package footballdata

import (
	"context"
	"time"

	"github.com/Liban213/pitchside/internal/models"
)

// Client fetches Premier League data from the upstream API.
type Client interface {
	// FetchTeams returns all Premier League teams for the current season.
	FetchTeams(ctx context.Context) ([]models.Team, error)
	// FetchMatches returns Premier League matches in [dateFrom, dateTo].
	FetchMatches(ctx context.Context, dateFrom, dateTo time.Time) ([]models.Match, error)
}
