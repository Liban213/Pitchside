package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Liban213/pitchside/internal/models"
	"github.com/Liban213/pitchside/internal/store"
	"github.com/Liban213/pitchside/internal/ws"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://libanm@localhost:5432/pitchside_test_db"
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skipf("skipping: could not create pool for %s: %v", url, err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skipping: could not reach test database at %s: %v", url, err)
	}
	t.Cleanup(pool.Close)

	if err := store.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `TRUNCATE matches, teams RESTART IDENTITY CASCADE`)
	})

	return pool
}

func score(n int) *int { return &n }

func TestHandleFixtures(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	teams := store.NewTeamStore(pool)
	matches := store.NewMatchStore(pool)

	home := models.Team{ID: 1, Name: "Arsenal FC", Slug: "ARS"}
	away := models.Team{ID: 2, Name: "Chelsea FC", Slug: "CHE"}
	if err := teams.UpsertTeams(ctx, []models.Team{home, away}); err != nil {
		t.Fatalf("UpsertTeams: %v", err)
	}

	date := time.Date(2026, 9, 12, 15, 0, 0, 0, time.UTC)
	match := models.Match{ExternalID: 100, HomeTeamID: 1, AwayTeamID: 2, Date: date, Status: models.StatusFinished, HomeScore: score(2), AwayScore: score(1)}
	if err := matches.UpsertMatches(ctx, []models.Match{match}); err != nil {
		t.Fatalf("UpsertMatches: %v", err)
	}

	server := NewServer(teams, matches, ws.NewHub())
	req := httptest.NewRequest(http.MethodGet, "/api/fixtures?from=2026-09-01&to=2026-09-30", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got []fixtureResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 fixture, got %d", len(got))
	}
	if got[0].HomeTeam.Name != "Arsenal FC" || got[0].AwayTeam.Name != "Chelsea FC" {
		t.Fatalf("unexpected team names: %+v", got[0])
	}
	if *got[0].HomeScore != 2 || *got[0].AwayScore != 1 {
		t.Fatalf("unexpected score: %+v", got[0])
	}
}

func TestHandleFixtures_InvalidDate(t *testing.T) {
	pool := testPool(t)
	server := NewServer(store.NewTeamStore(pool), store.NewMatchStore(pool), ws.NewHub())

	req := httptest.NewRequest(http.MethodGet, "/api/fixtures?from=not-a-date", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePrediction(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	teams := store.NewTeamStore(pool)
	matches := store.NewMatchStore(pool)

	strong, weak := models.Team{ID: 1, Name: "Strong FC", Slug: "STR"}, models.Team{ID: 2, Name: "Weak FC", Slug: "WEA"}
	if err := teams.UpsertTeams(ctx, []models.Team{strong, weak}); err != nil {
		t.Fatalf("UpsertTeams: %v", err)
	}

	day := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)
	history := []models.Match{
		{ExternalID: 1, HomeTeamID: 1, AwayTeamID: 2, Date: day, Status: models.StatusFinished, HomeScore: score(3), AwayScore: score(0)},
		{ExternalID: 2, HomeTeamID: 2, AwayTeamID: 1, Date: day.AddDate(0, 0, 1), Status: models.StatusFinished, HomeScore: score(0), AwayScore: score(3)},
	}
	if err := matches.UpsertMatches(ctx, history); err != nil {
		t.Fatalf("UpsertMatches: %v", err)
	}
	upcoming := models.Match{ExternalID: 3, HomeTeamID: 1, AwayTeamID: 2, Date: day.AddDate(0, 0, 7), Status: models.StatusScheduled}
	if err := matches.UpsertMatches(ctx, []models.Match{upcoming}); err != nil {
		t.Fatalf("UpsertMatches (upcoming): %v", err)
	}

	fixtures, err := matches.ListByDateRange(ctx, day, day.AddDate(0, 0, 14))
	if err != nil {
		t.Fatalf("ListByDateRange: %v", err)
	}
	var matchID int
	for _, m := range fixtures {
		if m.Status == models.StatusScheduled {
			matchID = m.ID
		}
	}
	if matchID == 0 {
		t.Fatalf("could not find upcoming match id")
	}

	server := NewServer(teams, matches, ws.NewHub())
	req := httptest.NewRequest(http.MethodGet, "/api/predictions/"+strconv.Itoa(matchID), nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got predictionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	sum := got.HomeWin + got.Draw + got.AwayWin
	if sum < 0.99 || sum > 1.01 {
		t.Fatalf("expected probabilities to sum to ~1, got %v (%+v)", sum, got)
	}
}

func TestHandlePrediction_NotFound(t *testing.T) {
	pool := testPool(t)
	server := NewServer(store.NewTeamStore(pool), store.NewMatchStore(pool), ws.NewHub())

	req := httptest.NewRequest(http.MethodGet, "/api/predictions/999", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleTeamStats(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	teams := store.NewTeamStore(pool)
	matches := store.NewMatchStore(pool)

	home, away := models.Team{ID: 1, Name: "Arsenal FC", Slug: "ARS"}, models.Team{ID: 2, Name: "Chelsea FC", Slug: "CHE"}
	if err := teams.UpsertTeams(ctx, []models.Team{home, away}); err != nil {
		t.Fatalf("UpsertTeams: %v", err)
	}

	day := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)
	match := models.Match{ExternalID: 1, HomeTeamID: 1, AwayTeamID: 2, Date: day, Status: models.StatusFinished, HomeScore: score(3), AwayScore: score(1)}
	if err := matches.UpsertMatches(ctx, []models.Match{match}); err != nil {
		t.Fatalf("UpsertMatches: %v", err)
	}

	server := NewServer(teams, matches, ws.NewHub())
	req := httptest.NewRequest(http.MethodGet, "/api/teams/1/stats", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got teamStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.MatchesPlayed != 1 || got.Wins != 1 || got.GoalsFor != 3 || got.GoalsAgainst != 1 {
		t.Fatalf("unexpected stats: %+v", got)
	}
}

func TestHandleTeamStats_NotFound(t *testing.T) {
	pool := testPool(t)
	server := NewServer(store.NewTeamStore(pool), store.NewMatchStore(pool), ws.NewHub())

	req := httptest.NewRequest(http.MethodGet, "/api/teams/999/stats", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
