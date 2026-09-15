# Pitchside

A Go service that ingests Premier League match data from football-data.org,
stores it in Postgres, and caches upstream calls in Redis to stay within the
free API tier's rate limit.

## Status: Day 1

Data ingestion, storage, and caching foundation. Prediction model and live
WebSocket updates are not built yet.

## Stack

- Go
- PostgreSQL (via `pgx`)
- Redis (via `go-redis`)
- Docker Compose for local Postgres + Redis

## Layout

- `internal/models` — domain types (`Team`, `Match`)
- `internal/footballdata` — football-data.org client, behind a `Client`
  interface so tests never hit the real API
- `internal/cache` — Redis-backed decorator over `footballdata.Client`;
  short TTL for fixtures that can still change, long TTL for finished
  results and team lists
- `internal/store` — Postgres repositories for teams and matches, plus
  embedded SQL migrations
- `cmd/pitchside` — entrypoint that syncs teams and this week's fixtures

## Running locally

```
docker compose up -d
cp .env.example .env   # add your football-data.org API key
go run ./cmd/pitchside
```

## Testing

```
createdb pitchside_test_db   # once, if not using docker compose
go test ./...
```

The cache tests use an in-memory Redis fake (`miniredis`) and a fake
`footballdata.Client`, so they run without any external services. The
Postgres store tests need a real database and skip themselves if
`pitchside_test_db` isn't reachable.
