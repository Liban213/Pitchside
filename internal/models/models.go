// Package models holds domain types shared across the ingestion, storage,
// caching, and prediction layers.
package models

import "time"

type Team struct {
	ID   int
	Name string
	Slug string
}

type MatchStatus string

const (
	StatusScheduled MatchStatus = "SCHEDULED"
	StatusLive      MatchStatus = "IN_PLAY"
	StatusFinished  MatchStatus = "FINISHED"
)

type Match struct {
	ID         int
	ExternalID int
	HomeTeamID int
	AwayTeamID int
	Date       time.Time
	Status     MatchStatus
	HomeScore  *int
	AwayScore  *int
}

// MatchEvent is a live status/score change for a match, broadcast to
// WebSocket clients as it happens.
type MatchEvent struct {
	MatchID    int         `json:"match_id"`
	ExternalID int         `json:"external_id"`
	HomeTeamID int         `json:"home_team_id"`
	AwayTeamID int         `json:"away_team_id"`
	Status     MatchStatus `json:"status"`
	HomeScore  *int        `json:"home_score"`
	AwayScore  *int        `json:"away_score"`
}
