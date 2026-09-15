package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Liban213/pitchside/internal/models"
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

	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `TRUNCATE matches, teams RESTART IDENTITY CASCADE`)
	})

	return pool
}

func TestTeamStore_UpsertAndRetrieve(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	teams := NewTeamStore(pool)

	want := models.Team{ID: 57, Name: "Arsenal FC", Slug: "ARS"}
	if err := teams.UpsertTeams(ctx, []models.Team{want}); err != nil {
		t.Fatalf("UpsertTeams: %v", err)
	}

	got, err := teams.GetTeam(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	updated := models.Team{ID: 57, Name: "Arsenal Football Club", Slug: "ARS"}
	if err := teams.UpsertTeams(ctx, []models.Team{updated}); err != nil {
		t.Fatalf("UpsertTeams (update): %v", err)
	}
	got, err = teams.GetTeam(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetTeam after update: %v", err)
	}
	if got.Name != updated.Name {
		t.Fatalf("expected name to update to %q, got %q", updated.Name, got.Name)
	}
}

func TestMatchStore_UpsertAndRetrieve(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	teams := NewTeamStore(pool)
	matches := NewMatchStore(pool)

	home := models.Team{ID: 57, Name: "Arsenal FC", Slug: "ARS"}
	away := models.Team{ID: 61, Name: "Chelsea FC", Slug: "CHE"}
	if err := teams.UpsertTeams(ctx, []models.Team{home, away}); err != nil {
		t.Fatalf("UpsertTeams: %v", err)
	}

	matchDate := time.Date(2026, 8, 30, 15, 0, 0, 0, time.UTC)
	homeScore, awayScore := 2, 1
	match := models.Match{
		ExternalID: 12345,
		HomeTeamID: home.ID,
		AwayTeamID: away.ID,
		Date:       matchDate,
		Status:     models.StatusFinished,
		HomeScore:  &homeScore,
		AwayScore:  &awayScore,
	}
	if err := matches.UpsertMatches(ctx, []models.Match{match}); err != nil {
		t.Fatalf("UpsertMatches: %v", err)
	}

	results, err := matches.ListByDateRange(ctx, matchDate.AddDate(0, 0, -1), matchDate.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("ListByDateRange: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 match in range, got %d", len(results))
	}
	got := results[0]
	if got.ExternalID != match.ExternalID || got.Status != models.StatusFinished ||
		*got.HomeScore != homeScore || *got.AwayScore != awayScore {
		t.Fatalf("retrieved match %+v does not match stored %+v", got, match)
	}

	finished, err := matches.FinishedByTeam(ctx, home.ID)
	if err != nil {
		t.Fatalf("FinishedByTeam: %v", err)
	}
	if len(finished) != 1 {
		t.Fatalf("expected 1 finished match for home team, got %d", len(finished))
	}
}
