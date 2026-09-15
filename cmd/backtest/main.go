// Command backtest walk-forward evaluates the prediction model against
// every finished match currently stored in Postgres and prints an accuracy
// report.
package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Liban213/pitchside/internal/config"
	"github.com/Liban213/pitchside/internal/predict"
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

	matches, err := store.NewMatchStore(pool).ListFinished(ctx)
	if err != nil {
		log.Fatalf("list finished matches: %v", err)
	}

	report := predict.Backtest(matches)
	log.Printf("backtest: %d finished matches, %d evaluated, %d correct (%.1f%% accuracy)",
		len(matches), report.Evaluated, report.Correct, report.Accuracy()*100)
}
