package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"football-predictor/internal/domain"
	"football-predictor/internal/repository"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type MatchService struct {
	repo         *repository.Repository
	apiToken     string
	competitions []string
	baseURL      string
	daysAhead    int
	client       *http.Client
}

func NewMatchService(r *repository.Repository, apiToken string) *MatchService {
	return &MatchService{
		repo:         r,
		apiToken:     strings.TrimSpace(apiToken),
		competitions: parseCompetitions(getEnvString("FOOTBALL_DATA_COMPETITIONS", "CL,WC")),
		baseURL:      "https://api.football-data.org/v4",
		daysAhead:    getEnvInt("FOOTBALL_DATA_DAYS_AHEAD", 7),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type FootballDataMatchesResponse struct {
	Filters   map[string]any `json:"filters"`
	ResultSet struct {
		Count int `json:"count"`
	} `json:"resultSet"`
	Competition struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Code   string `json:"code"`
		Emblem string `json:"emblem"`
	} `json:"competition"`
	Matches []FootballDataMatch `json:"matches"`
}

type FootballDataMatch struct {
	ID          int64  `json:"id"`
	UTCDate     string `json:"utcDate"`
	Status      string `json:"status"`
	Matchday    int    `json:"matchday"`
	Stage       string `json:"stage"`
	LastUpdated string `json:"lastUpdated"`

	Competition struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Code   string `json:"code"`
		Emblem string `json:"emblem"`
	} `json:"competition"`

	HomeTeam struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		TLA       string `json:"tla"`
		Crest     string `json:"crest"`
	} `json:"homeTeam"`

	AwayTeam struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		TLA       string `json:"tla"`
		Crest     string `json:"crest"`
	} `json:"awayTeam"`

	Score struct {
		Winner   string `json:"winner"`
		Duration string `json:"duration"`
		FullTime struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"fullTime"`
	} `json:"score"`
}

func (s *MatchService) FetchDailyMatches() error {
	from := time.Now().UTC().Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, s.daysAhead).Format("2006-01-02")

	for _, competition := range s.competitions {
		url := fmt.Sprintf(
			"%s/competitions/%s/matches?dateFrom=%s&dateTo=%s",
			s.baseURL,
			competition,
			from,
			to,
		)

		fmt.Printf("football-data fetch: %s\n", url)

		apiResp, err := s.fetchFootballDataMatches(url)
		if err != nil {
			return err
		}

		fmt.Printf(
			"football-data loaded: competition=%s from=%s to=%s results=%d\n",
			competition,
			from,
			to,
			len(apiResp.Matches),
		)

		type finishedResult struct {
			matchID int64
			home    int
			away    int
		}

		matchesToSave := make([]domain.Match, 0, len(apiResp.Matches))
		finishedResults := make([]finishedResult, 0)

		for _, item := range apiResp.Matches {
			matchTime, err := time.Parse(time.RFC3339, item.UTCDate)
			if err != nil {
				continue
			}

			status := mapFootballDataStatus(item.Status)
			homeGoals := item.Score.FullTime.Home
			awayGoals := item.Score.FullTime.Away
			outcome := ""

			if status == "FINISHED" && homeGoals != nil && awayGoals != nil {
				outcome = calculateOutcome(*homeGoals, *awayGoals)
				finishedResults = append(finishedResults, finishedResult{
					matchID: item.ID,
					home:    *homeGoals,
					away:    *awayGoals,
				})
			}

			leagueID := item.Competition.ID
			leagueName := item.Competition.Name
			if leagueID == 0 {
				leagueID = apiResp.Competition.ID
			}
			if leagueName == "" {
				leagueName = apiResp.Competition.Name
			}

			matchesToSave = append(matchesToSave, domain.Match{
				APIID:      item.ID,
				LeagueID:   leagueID,
				LeagueName: leagueName,
				HomeTeam:   cleanTeamName(item.HomeTeam.Name, item.HomeTeam.ShortName),
				AwayTeam:   cleanTeamName(item.AwayTeam.Name, item.AwayTeam.ShortName),
				MatchTime:  matchTime,
				Status:     status,
				HomeGoals:  homeGoals,
				AwayGoals:  awayGoals,
				Outcome:    outcome,
			})
		}

		if len(matchesToSave) > 0 {
			if err := s.repo.SaveMatches(matchesToSave); err != nil {
				return err
			}
		}

		for _, result := range finishedResults {
			if err := s.applyFinishedMatch(result.matchID, result.home, result.away); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *MatchService) GetMatchesForUser(tgID int64, status string) ([]domain.MatchForUser, error) {
	return s.repo.GetMatchesForUser(tgID, status)
}

func (s *MatchService) GetMatchForUser(tgID int64, matchID int64) (*domain.MatchForUser, error) {
	return s.repo.GetMatchForUser(matchID, tgID)
}

func (s *MatchService) GetPredictionStats(matchID int64) (*domain.PredictionStats, error) {
	match, err := s.repo.GetMatchByID(matchID)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, errors.New("match not found")
	}
	return s.repo.GetPredictionStatsByMatch(matchID)
}

func (s *MatchService) GetPredictionHistory(tgID int64, status string) ([]domain.PredictionHistoryItem, error) {
	return s.repo.GetUserPredictionHistory(tgID, status)
}

func (s *MatchService) GetLeaderboard(limit int) ([]domain.LeaderboardItem, error) {
	return s.repo.GetLeaderboardWithRanks(limit)
}

func (s *MatchService) GetMe(tgID int64) (*domain.MeResponse, error) {
	return s.repo.GetMeStats(tgID)
}

func (s *MatchService) GetLeagues() ([]domain.LeagueInfo, error) {
	return s.repo.GetLeagues()
}

func (s *MatchService) GetRules() domain.RulesResponse {
	return domain.RulesResponse{
		PredictionChoices: []string{"1", "X", "2"},
		Points: map[string]int{
			"correct_outcome": 1,
			"wrong_outcome":   0,
		},
		LockRule: "Прогноз можно менять только до времени начала матча и только пока статус матча SCHEDULED.",
	}
}

func (s *MatchService) GetAdminStats() (*domain.AdminStats, error) {
	return s.repo.GetAdminStats()
}

func (s *MatchService) ProcessFinishedMatches() error {
	activeMatches, err := s.repo.GetActiveMatches()
	if err != nil {
		return err
	}

	for _, m := range activeMatches {
		if err := s.RecalculateMatch(m.APIID); err != nil {
			continue
		}
	}

	return nil
}

func (s *MatchService) RecalculateMatch(matchID int64) error {
	item, err := s.fetchFootballDataMatchByID(matchID)
	if err != nil {
		return err
	}

	status := mapFootballDataStatus(item.Status)
	if status != "FINISHED" ||
		item.Score.FullTime.Home == nil ||
		item.Score.FullTime.Away == nil {
		return errors.New("match is not finished yet")
	}

	return s.applyFinishedMatch(
		matchID,
		*item.Score.FullTime.Home,
		*item.Score.FullTime.Away,
	)
}

func (s *MatchService) applyFinishedMatch(matchID int64, homeGoals, awayGoals int) error {
	outcome := calculateOutcome(homeGoals, awayGoals)

	if err := s.repo.UpdateMatchResults(matchID, homeGoals, awayGoals, outcome); err != nil {
		return err
	}

	predictions, err := s.repo.GetPredictionsByMatch(matchID)
	if err != nil {
		return err
	}

	for _, p := range predictions {
		isCorrect := p.UserChoice == outcome
		if err := s.repo.SetPredictionResult(p.ID, p.UserID, isCorrect); err != nil {
			return err
		}
	}

	return nil
}

func (s *MatchService) fetchFootballDataMatchByID(matchID int64) (*FootballDataMatch, error) {
	url := fmt.Sprintf("%s/matches/%d", s.baseURL, matchID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", s.apiToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("football-data returned status %d: %v", resp.StatusCode, errBody)
	}

	var match FootballDataMatch
	if err := json.NewDecoder(resp.Body).Decode(&match); err != nil {
		return nil, err
	}

	return &match, nil
}

func (s *MatchService) fetchFootballDataMatches(url string) (*FootballDataMatchesResponse, error) {
	if strings.TrimSpace(s.apiToken) == "" {
		return nil, errors.New("FOOTBALL_DATA_TOKEN is empty")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", s.apiToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("football-data returned status %d: %v", resp.StatusCode, errBody)
	}

	var apiResp FootballDataMatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

func calculateOutcome(homeGoals, awayGoals int) string {
	if homeGoals > awayGoals {
		return "1"
	}
	if homeGoals < awayGoals {
		return "2"
	}
	return "X"
}

func mapFootballDataStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "FINISHED":
		return "FINISHED"
	case "IN_PLAY", "PAUSED":
		return "LIVE"
	case "SCHEDULED", "TIMED":
		return "SCHEDULED"
	case "POSTPONED", "SUSPENDED", "CANCELLED":
		return "CANCELLED"
	default:
		return "SCHEDULED"
	}
}

func cleanTeamName(name, shortName string) string {
	name = strings.TrimSpace(name)
	shortName = strings.TrimSpace(shortName)

	if shortName != "" {
		return shortName
	}

	return name
}

func parseCompetitions(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.ToUpper(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		result = append(result, part)
	}

	if len(result) == 0 {
		return []string{"CL", "WC"}
	}

	return result
}

func getEnvString(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
