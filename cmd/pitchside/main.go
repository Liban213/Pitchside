package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Liban213/pitchside/internal/cache"
	"github.com/Liban213/pitchside/internal/config"
	"github.com/Liban213/pitchside/internal/footballdata"
	"github.com/Liban213/pitchside/internal/store"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

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

	if err := syncOnce(ctx, client, teamStore, matchStore); err != nil {
		log.Fatalf("sync: %v", err)
	}
	log.Println("sync complete")
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
