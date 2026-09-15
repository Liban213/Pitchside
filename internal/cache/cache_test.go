package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/Liban213/pitchside/internal/footballdata"
	"github.com/Liban213/pitchside/internal/models"
)

func newTestClient(t *testing.T) (*CachingClient, *footballdata.Fake) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	fake := &footballdata.Fake{
		Teams: []models.Team{{ID: 1, Name: "Arsenal", Slug: "ARS"}},
	}
	return NewCachingClient(fake, rdb), fake
}

func TestFetchTeams_CacheHitDoesNotCallUpstream(t *testing.T) {
	client, fake := newTestClient(t)
	ctx := context.Background()

	first, err := client.FetchTeams(ctx)
	if err != nil {
		t.Fatalf("first FetchTeams: %v", err)
	}
	if fake.TeamsCalls != 1 {
		t.Fatalf("expected 1 upstream call after miss, got %d", fake.TeamsCalls)
	}

	second, err := client.FetchTeams(ctx)
	if err != nil {
		t.Fatalf("second FetchTeams: %v", err)
	}
	if fake.TeamsCalls != 1 {
		t.Fatalf("expected upstream call count to stay at 1 on cache hit, got %d", fake.TeamsCalls)
	}

	if len(second) != len(first) || second[0].Name != first[0].Name {
		t.Fatalf("cached result %v does not match original %v", second, first)
	}
}

func TestFetchMatches_HistoricalRangeCachedLong(t *testing.T) {
	client, fake := newTestClient(t)
	ctx := context.Background()

	past := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	client.now = func() time.Time { return time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC) }

	if _, err := client.FetchMatches(ctx, past, past); err != nil {
		t.Fatalf("FetchMatches: %v", err)
	}
	ttl := client.fixturesTTL(past)
	if ttl != finishedFixturesTTL {
		t.Fatalf("expected finishedFixturesTTL for a past date, got %v", ttl)
	}

	if _, err := client.FetchMatches(ctx, past, past); err != nil {
		t.Fatalf("FetchMatches (second call): %v", err)
	}
	if fake.MatchesCalls != 1 {
		t.Fatalf("expected upstream call count to stay at 1 on cache hit, got %d", fake.MatchesCalls)
	}
}

func TestFetchMatches_LiveRangeShortTTL(t *testing.T) {
	client, _ := newTestClient(t)
	client.now = func() time.Time { return time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC) }

	future := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	if ttl := client.fixturesTTL(future); ttl != liveFixturesTTL {
		t.Fatalf("expected liveFixturesTTL for a future date, got %v", ttl)
	}
}
