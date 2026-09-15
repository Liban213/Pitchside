package footballdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Liban213/pitchside/internal/models"
)

const (
	defaultBaseURL   = "https://api.football-data.org/v4"
	premierLeagueID  = "PL"
)

// HTTPClient is the real football-data.org implementation of Client.
type HTTPClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(apiKey string) *HTTPClient {
	return &HTTPClient{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPClient) do(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("footballdata: build request: %w", err)
	}
	req.Header.Set("X-Auth-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("footballdata: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("footballdata: %s returned status %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("footballdata: decode %s: %w", path, err)
	}
	return nil
}

type teamsResponse struct {
	Teams []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"tla"`
	} `json:"teams"`
}

func (c *HTTPClient) FetchTeams(ctx context.Context) ([]models.Team, error) {
	var resp teamsResponse
	if err := c.do(ctx, fmt.Sprintf("/competitions/%s/teams", premierLeagueID), &resp); err != nil {
		return nil, err
	}

	teams := make([]models.Team, 0, len(resp.Teams))
	for _, t := range resp.Teams {
		teams = append(teams, models.Team{
			ID:   t.ID,
			Name: t.Name,
			Slug: t.ShortName,
		})
	}
	return teams, nil
}

type matchesResponse struct {
	Matches []struct {
		ID       int       `json:"id"`
		UTCDate  time.Time `json:"utcDate"`
		Status   string    `json:"status"`
		HomeTeam struct {
			ID int `json:"id"`
		} `json:"homeTeam"`
		AwayTeam struct {
			ID int `json:"id"`
		} `json:"awayTeam"`
		Score struct {
			FullTime struct {
				Home *int `json:"home"`
				Away *int `json:"away"`
			} `json:"fullTime"`
		} `json:"score"`
	} `json:"matches"`
}

func (c *HTTPClient) FetchMatches(ctx context.Context, dateFrom, dateTo time.Time) ([]models.Match, error) {
	path := fmt.Sprintf("/competitions/%s/matches?dateFrom=%s&dateTo=%s",
		premierLeagueID, dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"))

	var resp matchesResponse
	if err := c.do(ctx, path, &resp); err != nil {
		return nil, err
	}

	matches := make([]models.Match, 0, len(resp.Matches))
	for _, m := range resp.Matches {
		matches = append(matches, models.Match{
			ExternalID: m.ID,
			HomeTeamID: m.HomeTeam.ID,
			AwayTeamID: m.AwayTeam.ID,
			Date:       m.UTCDate,
			Status:     models.MatchStatus(m.Status),
			HomeScore:  m.Score.FullTime.Home,
			AwayScore:  m.Score.FullTime.Away,
		})
	}
	return matches, nil
}
