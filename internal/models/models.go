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
