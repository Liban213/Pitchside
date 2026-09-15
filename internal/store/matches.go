package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Liban213/pitchside/internal/models"
)

type MatchStore struct {
	pool *pgxpool.Pool
}

func NewMatchStore(pool *pgxpool.Pool) *MatchStore {
	return &MatchStore{pool: pool}
}

// UpsertMatches inserts matches that don't exist yet and updates the
// mutable fields (status/score) of ones that do, keyed on external_id.
func (s *MatchStore) UpsertMatches(ctx context.Context, matches []models.Match) error {
	for _, m := range matches {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO matches (external_id, home_team_id, away_team_id, match_date, status, home_score, away_score)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (external_id) DO UPDATE
				SET status = $5, home_score = $6, away_score = $7`,
			m.ExternalID, m.HomeTeamID, m.AwayTeamID, m.Date, string(m.Status), m.HomeScore, m.AwayScore)
		if err != nil {
			return fmt.Errorf("store: upsert match %d: %w", m.ExternalID, err)
		}
	}
	return nil
}

func (s *MatchStore) GetMatch(ctx context.Context, id int) (models.Match, error) {
	m, err := scanMatch(s.pool.QueryRow(ctx, `
		SELECT id, external_id, home_team_id, away_team_id, match_date, status, home_score, away_score
		FROM matches WHERE id = $1`, id))
	if err != nil {
		return models.Match{}, fmt.Errorf("store: get match %d: %w", id, err)
	}
	return m, nil
}

func (s *MatchStore) ListByDateRange(ctx context.Context, from, to time.Time) ([]models.Match, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, external_id, home_team_id, away_team_id, match_date, status, home_score, away_score
		FROM matches WHERE match_date >= $1 AND match_date <= $2 ORDER BY match_date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("store: list matches by date range: %w", err)
	}
	defer rows.Close()
	return collectMatches(rows)
}

// FinishedByTeam returns every finished match involving teamID, oldest
// first — the raw history the prediction model computes attack/defense
// strength from.
func (s *MatchStore) FinishedByTeam(ctx context.Context, teamID int) ([]models.Match, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, external_id, home_team_id, away_team_id, match_date, status, home_score, away_score
		FROM matches
		WHERE status = $1 AND (home_team_id = $2 OR away_team_id = $2)
		ORDER BY match_date`, string(models.StatusFinished), teamID)
	if err != nil {
		return nil, fmt.Errorf("store: list finished matches for team %d: %w", teamID, err)
	}
	defer rows.Close()
	return collectMatches(rows)
}

func (s *MatchStore) ListFinished(ctx context.Context) ([]models.Match, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, external_id, home_team_id, away_team_id, match_date, status, home_score, away_score
		FROM matches WHERE status = $1 ORDER BY match_date`, string(models.StatusFinished))
	if err != nil {
		return nil, fmt.Errorf("store: list finished matches: %w", err)
	}
	defer rows.Close()
	return collectMatches(rows)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMatch(row rowScanner) (models.Match, error) {
	var m models.Match
	var status string
	err := row.Scan(&m.ID, &m.ExternalID, &m.HomeTeamID, &m.AwayTeamID, &m.Date, &status, &m.HomeScore, &m.AwayScore)
	m.Status = models.MatchStatus(status)
	return m, err
}

type rowsScanner interface {
	rowScanner
	Next() bool
	Err() error
}

func collectMatches(rows rowsScanner) ([]models.Match, error) {
	var matches []models.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan match: %w", err)
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}
