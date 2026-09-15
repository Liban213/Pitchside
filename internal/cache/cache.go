// Package cache wraps a footballdata.Client with a Redis cache so many
// concurrent callers share one upstream quota. football-data.org's free
// tier allows roughly 10 requests/minute; without this layer, every
// connected client hitting an endpoint would translate into a fresh
// upstream call and the app would exhaust its quota in seconds.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Liban213/pitchside/internal/footballdata"
	"github.com/Liban213/pitchside/internal/models"
)

const (
	// teamsTTL is long because team identities barely change mid-season.
	teamsTTL = 24 * time.Hour
	// liveFixturesTTL covers date ranges that include today or the future,
	// where scores/status can change at any moment.
	liveFixturesTTL = 5 * time.Minute
	// finishedFixturesTTL covers date ranges entirely in the past, whose
	// results are final and will never change.
	finishedFixturesTTL = 30 * 24 * time.Hour
)

const teamsCacheKey = "teams:PL"

// CachingClient decorates a footballdata.Client with a Redis-backed cache.
// It implements footballdata.Client, so it's a drop-in replacement for the
// real HTTP client anywhere one is used.
type CachingClient struct {
	upstream footballdata.Client
	rdb      *redis.Client
	now      func() time.Time
}

func NewCachingClient(upstream footballdata.Client, rdb *redis.Client) *CachingClient {
	return &CachingClient{upstream: upstream, rdb: rdb, now: time.Now}
}

func (c *CachingClient) FetchTeams(ctx context.Context) ([]models.Team, error) {
	var teams []models.Team
	if hit, err := getCached(ctx, c.rdb, teamsCacheKey, &teams); err != nil {
		return nil, err
	} else if hit {
		return teams, nil
	}

	teams, err := c.upstream.FetchTeams(ctx)
	if err != nil {
		return nil, err
	}
	if err := setCached(ctx, c.rdb, teamsCacheKey, teams, teamsTTL); err != nil {
		return nil, err
	}
	return teams, nil
}

func (c *CachingClient) FetchMatches(ctx context.Context, dateFrom, dateTo time.Time) ([]models.Match, error) {
	key := fixturesCacheKey(dateFrom, dateTo)

	var matches []models.Match
	if hit, err := getCached(ctx, c.rdb, key, &matches); err != nil {
		return nil, err
	} else if hit {
		return matches, nil
	}

	matches, err := c.upstream.FetchMatches(ctx, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	if err := setCached(ctx, c.rdb, key, matches, c.fixturesTTL(dateTo)); err != nil {
		return nil, err
	}
	return matches, nil
}

func fixturesCacheKey(dateFrom, dateTo time.Time) string {
	return fmt.Sprintf("fixtures:%s:%s", dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"))
}

// fixturesTTL picks a short TTL when the range could still change (includes
// today or the future) and a long one when it's entirely historical.
func (c *CachingClient) fixturesTTL(dateTo time.Time) time.Duration {
	today := c.now().Truncate(24 * time.Hour)
	if dateTo.Truncate(24 * time.Hour).Before(today) {
		return finishedFixturesTTL
	}
	return liveFixturesTTL
}

func getCached(ctx context.Context, rdb *redis.Client, key string, out any) (bool, error) {
	raw, err := rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache: get %s: %w", key, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return false, fmt.Errorf("cache: unmarshal %s: %w", key, err)
	}
	return true, nil
}

func setCached(ctx context.Context, rdb *redis.Client, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal %s: %w", key, err)
	}
	if err := rdb.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %s: %w", key, err)
	}
	return nil
}
