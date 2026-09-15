CREATE TABLE teams (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL
);

CREATE TABLE matches (
    id           SERIAL PRIMARY KEY,
    external_id  INTEGER NOT NULL UNIQUE,
    home_team_id INTEGER NOT NULL REFERENCES teams(id),
    away_team_id INTEGER NOT NULL REFERENCES teams(id),
    match_date   TIMESTAMPTZ NOT NULL,
    status       TEXT NOT NULL,
    home_score   INTEGER,
    away_score   INTEGER
);

CREATE INDEX idx_matches_date ON matches (match_date);
CREATE INDEX idx_matches_home_team ON matches (home_team_id);
CREATE INDEX idx_matches_away_team ON matches (away_team_id);
