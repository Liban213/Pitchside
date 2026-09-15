package predict

import "github.com/Liban213/pitchside/internal/models"

type teamRecord struct {
	goalsFor     float64
	goalsAgainst float64
	matches      int
}

// league holds the scoring rates a Predict call needs: each team's overall
// attack/defense strength relative to the league, plus the league-wide
// home/away scoring averages that encode home advantage.
type league struct {
	avgHomeGoals float64
	avgAwayGoals float64
	avgPerTeam   float64
	teams        map[int]*teamRecord
}

// newLeague aggregates finished matches into per-team scoring rates. Matches
// that aren't finished, or are missing a score, are ignored.
func newLeague(matches []models.Match) *league {
	l := &league{teams: make(map[int]*teamRecord)}

	var totalHomeGoals, totalAwayGoals float64
	var n int
	for _, m := range matches {
		if m.Status != models.StatusFinished || m.HomeScore == nil || m.AwayScore == nil {
			continue
		}
		hs, as := float64(*m.HomeScore), float64(*m.AwayScore)

		home := l.record(m.HomeTeamID)
		home.goalsFor += hs
		home.goalsAgainst += as
		home.matches++

		away := l.record(m.AwayTeamID)
		away.goalsFor += as
		away.goalsAgainst += hs
		away.matches++

		totalHomeGoals += hs
		totalAwayGoals += as
		n++
	}

	if n > 0 {
		l.avgHomeGoals = totalHomeGoals / float64(n)
		l.avgAwayGoals = totalAwayGoals / float64(n)
		l.avgPerTeam = (totalHomeGoals + totalAwayGoals) / float64(2*n)
	}
	return l
}

func (l *league) record(teamID int) *teamRecord {
	r, ok := l.teams[teamID]
	if !ok {
		r = &teamRecord{}
		l.teams[teamID] = r
	}
	return r
}

// attack returns how many times more (or fewer) goals teamID scores per
// match than the league-average team; 1.0 (neutral) if teamID has no
// finished matches yet.
func (l *league) attack(teamID int) float64 {
	r, ok := l.teams[teamID]
	if !ok || r.matches == 0 || l.avgPerTeam == 0 {
		return 1
	}
	return (r.goalsFor / float64(r.matches)) / l.avgPerTeam
}

// defense returns how many times more (or fewer) goals teamID concedes per
// match than the league-average team; 1.0 (neutral) if teamID has no
// finished matches yet.
func (l *league) defense(teamID int) float64 {
	r, ok := l.teams[teamID]
	if !ok || r.matches == 0 || l.avgPerTeam == 0 {
		return 1
	}
	return (r.goalsAgainst / float64(r.matches)) / l.avgPerTeam
}

// expectedGoals returns the Poisson expected goals for both sides of a
// home-vs-away fixture: the league's home/away scoring baseline, scaled by
// the attacking side's strength and the opponent's defensive weakness.
func (l *league) expectedGoals(homeTeamID, awayTeamID int) (homeXG, awayXG float64) {
	homeXG = l.avgHomeGoals * l.attack(homeTeamID) * l.defense(awayTeamID)
	awayXG = l.avgAwayGoals * l.attack(awayTeamID) * l.defense(homeTeamID)
	return homeXG, awayXG
}
