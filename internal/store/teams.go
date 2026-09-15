package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Liban213/pitchside/internal/models"
)

type TeamStore struct {
	pool *pgxpool.Pool
}

func NewTeamStore(pool *pgxpool.Pool) *TeamStore {
	return &TeamStore{pool: pool}
}

// UpsertTeams inserts teams that don't exist yet and updates the name/slug
// of ones that do, keyed on the external football-data.org id.
func (s *TeamStore) UpsertTeams(ctx context.Context, teams []models.Team) error {
	for _, t := range teams {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO teams (id, name, slug)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET name = $2, slug = $3`,
			t.ID, t.Name, t.Slug)
		if err != nil {
			return fmt.Errorf("store: upsert team %d: %w", t.ID, err)
		}
	}
	return nil
}

func (s *TeamStore) GetTeam(ctx context.Context, id int) (models.Team, error) {
	var t models.Team
	err := s.pool.QueryRow(ctx, `SELECT id, name, slug FROM teams WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Slug)
	if err != nil {
		return models.Team{}, fmt.Errorf("store: get team %d: %w", id, err)
	}
	return t, nil
}

func (s *TeamStore) ListTeams(ctx context.Context) ([]models.Team, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, slug FROM teams ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("store: list teams: %w", err)
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, fmt.Errorf("store: scan team: %w", err)
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}
