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
	"time"
)

type MatchService struct {
	repo    *repository.Repository
	apiKey  string
	leagues []int
	baseURL string
	season  int
	client  *http.Client
}

func NewMatchService(r *repository.Repository, apiKey string, leagues []int) *MatchService {
	return &MatchService{
		repo:    r,
		apiKey:  apiKey,
		leagues: leagues,
		baseURL: "https://v3.football.api-sports.io",
		season:  getSeason(),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type APIFootballResponse struct {
	Response []struct {
		Fixture struct {
			ID     int64  `json:"id"`
			Date   string `json:"date"`
			Status struct {
				Short string `json:"short"`
			} `json:"status"`
		} `json:"fixture"`
		League struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"league"`
		Teams struct {
			Home struct {
				Name string `json:"name"`
			} `json:"home"`
			Away struct {
				Name string `json:"name"`
			} `json:"away"`
		} `json:"teams"`
		Goals struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"goals"`
	} `json:"response"`
}

func (s *MatchService) FetchDailyMatches() error {
	today := time.Now().Format("2006-01-02")

	for _, leagueID := range s.leagues {
		url := fmt.Sprintf("%s/fixtures?date=%s&league=%d&season=%d", s.baseURL, today, leagueID, s.season)
		apiResp, err := s.fetchFixtures(url)
		if err != nil {
			return err
		}

		type finishedResult struct {
			matchID int64
			home    int
			away    int
		}

		matchesToSave := make([]domain.Match, 0, len(apiResp.Response))
		finishedResults := make([]finishedResult, 0)
		for _, item := range apiResp.Response {
			matchTime, err := time.Parse(time.RFC3339, item.Fixture.Date)
			if err != nil {
				continue
			}

			status := mapAPIStatus(item.Fixture.Status.Short)
			outcome := ""
			if status == "FINISHED" && item.Goals.Home != nil && item.Goals.Away != nil {
				outcome = calculateOutcome(*item.Goals.Home, *item.Goals.Away)
				finishedResults = append(finishedResults, finishedResult{
					matchID: item.Fixture.ID,
					home:    *item.Goals.Home,
					away:    *item.Goals.Away,
				})
			}

			matchesToSave = append(matchesToSave, domain.Match{
				APIID:      item.Fixture.ID,
				LeagueID:   item.League.ID,
				LeagueName: item.League.Name,
				HomeTeam:   item.Teams.Home.Name,
				AwayTeam:   item.Teams.Away.Name,
				MatchTime:  matchTime,
				Status:     status,
				HomeGoals:  item.Goals.Home,
				AwayGoals:  item.Goals.Away,
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
	item, err := s.fetchFixtureByID(matchID)
	if err != nil {
		return err
	}

	status := mapAPIStatus(item.Fixture.Status.Short)
	if status != "FINISHED" || item.Goals.Home == nil || item.Goals.Away == nil {
		return errors.New("match is not finished yet")
	}

	return s.applyFinishedMatch(matchID, *item.Goals.Home, *item.Goals.Away)
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

func (s *MatchService) fetchFixtureByID(matchID int64) (struct {
	Fixture struct {
		ID     int64  `json:"id"`
		Date   string `json:"date"`
		Status struct {
			Short string `json:"short"`
		} `json:"status"`
	} `json:"fixture"`
	League struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"league"`
	Teams struct {
		Home struct {
			Name string `json:"name"`
		} `json:"home"`
		Away struct {
			Name string `json:"name"`
		} `json:"away"`
	} `json:"teams"`
	Goals struct {
		Home *int `json:"home"`
		Away *int `json:"away"`
	} `json:"goals"`
}, error) {
	var zero struct {
		Fixture struct {
			ID     int64  `json:"id"`
			Date   string `json:"date"`
			Status struct {
				Short string `json:"short"`
			} `json:"status"`
		} `json:"fixture"`
		League struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"league"`
		Teams struct {
			Home struct {
				Name string `json:"name"`
			} `json:"home"`
			Away struct {
				Name string `json:"name"`
			} `json:"away"`
		} `json:"teams"`
		Goals struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"goals"`
	}

	url := fmt.Sprintf("%s/fixtures?id=%d", s.baseURL, matchID)
	apiResp, err := s.fetchFixtures(url)
	if err != nil {
		return zero, err
	}
	if len(apiResp.Response) == 0 {
		return zero, errors.New("match not found in external api")
	}

	return apiResp.Response[0], nil
}

func (s *MatchService) fetchFixtures(url string) (*APIFootballResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("x-apisports-key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api-football returned status %d", resp.StatusCode)
	}

	var apiResp APIFootballResponse
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

func mapAPIStatus(short string) string {
	switch short {
	case "FT", "AET", "PEN":
		return "FINISHED"
	case "NS", "TBD":
		return "SCHEDULED"
	default:
		return "LIVE"
	}
}

func getSeason() int {
	seasonRaw := os.Getenv("API_SEASON")
	if seasonRaw == "" {
		return 2025
	}
	season, err := strconv.Atoi(seasonRaw)
	if err != nil || season <= 0 {
		return 2025
	}
	return season
}
