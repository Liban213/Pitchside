package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Liban213/pitchside/internal/api"
	"github.com/Liban213/pitchside/internal/cache"
	"github.com/Liban213/pitchside/internal/config"
	"github.com/Liban213/pitchside/internal/footballdata"
	"github.com/Liban213/pitchside/internal/models"
	"github.com/Liban213/pitchside/internal/store"
	"github.com/Liban213/pitchside/internal/ws"
)

// pollInterval controls how often the server re-checks for live match
// changes, not how often it hits football-data.org — the CachingClient's
// short TTL for today/future ranges (internal/cache) is what actually
// throttles upstream calls.
const pollInterval = 30 * time.Second

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer pool.Close()

	if err := store.Migrate(ctx, pool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()

	upstream := footballdata.NewHTTPClient(cfg.FootballDataAPIKey)
	client := cache.NewCachingClient(upstream, rdb)

	teamStore := store.NewTeamStore(pool)
	matchStore := store.NewMatchStore(pool)
	hub := ws.NewHub()

	if err := syncOnce(ctx, client, teamStore, matchStore); err != nil {
		log.Fatalf("initial sync: %v", err)
	}

	go pollLive(ctx, client, matchStore, hub)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewServer(teamStore, matchStore, hub),
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}

// syncOnce pulls current teams and this week's fixtures through the cache
// and stores them in Postgres.
func syncOnce(ctx context.Context, client footballdata.Client, teamStore *store.TeamStore, matchStore *store.MatchStore) error {
	teams, err := client.FetchTeams(ctx)
	if err != nil {
		return err
	}
	if err := teamStore.UpsertTeams(ctx, teams); err != nil {
		return err
	}

	now := time.Now()
	matches, err := client.FetchMatches(ctx, now.AddDate(0, 0, -7), now.AddDate(0, 0, 7))
	if err != nil {
		return err
	}
	return matchStore.UpsertMatches(ctx, matches)
}

// pollLive periodically re-fetches matches in a narrow window around today
// and broadcasts a MatchEvent over hub for any status or score change, so
// WebSocket clients see live updates as they happen.
func pollLive(ctx context.Context, client footballdata.Client, matchStore *store.MatchStore, hub *ws.Hub) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := syncAndBroadcast(ctx, client, matchStore, hub); err != nil {
				log.Printf("poll: %v", err)
			}
		}
	}
}

func syncAndBroadcast(ctx context.Context, client footballdata.Client, matchStore *store.MatchStore, hub *ws.Hub) error {
	now := time.Now()
	from, to := now.AddDate(0, 0, -1), now.AddDate(0, 0, 1)

	before, err := matchStore.ListByDateRange(ctx, from, to)
	if err != nil {
		return err
	}
	beforeByExternalID := make(map[int]models.Match, len(before))
	for _, m := range before {
		beforeByExternalID[m.ExternalID] = m
	}

	matches, err := client.FetchMatches(ctx, from, to)
	if err != nil {
		return err
	}
	if err := matchStore.UpsertMatches(ctx, matches); err != nil {
		return err
	}

	for _, m := range matches {
		prev, existed := beforeByExternalID[m.ExternalID]
		if !existed || unchanged(prev, m) {
			continue
		}
		hub.Broadcast(models.MatchEvent{
			MatchID:    prev.ID,
			ExternalID: m.ExternalID,
			HomeTeamID: m.HomeTeamID,
			AwayTeamID: m.AwayTeamID,
			Status:     m.Status,
			HomeScore:  m.HomeScore,
			AwayScore:  m.AwayScore,
		})
	}
	return nil
}

func unchanged(prev, next models.Match) bool {
	return prev.Status == next.Status && scoreEqual(prev.HomeScore, next.HomeScore) && scoreEqual(prev.AwayScore, next.AwayScore)
}

func scoreEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
