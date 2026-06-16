package service

import (
	"encoding/json"
	"fmt"
	"football-predictor/internal/domain"
	"football-predictor/internal/repository"
	"net/http"
	"time"
)

type MatchService struct {
	repo    *repository.Repository
	apiKey  string
	leagues []int
	baseURL string
}

func NewMatchService(r *repository.Repository, apiKey string, leagues []int) *MatchService {
	return &MatchService{
		repo:    r,
		apiKey:  apiKey,
		leagues: leagues,
		baseURL: "https://v3.football.api-sports.io",
	}
}

type APIFootballResponse struct {
	Response []struct {
		Fixture struct {
			ID     int64  `json:"id"`
			Date   string `json:"date"`
			Status struct {
				Short string `json:"short"` // "NS" (Not Started), "FT" (Finished), "1H", "2H"
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
		url := fmt.Sprintf("%s/fixtures?date=%s&league=%d&season=2025", s.baseURL, today, leagueID)

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("x-apisports-key", s.apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var apiResp APIFootballResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return err
		}

		var matchesToSave []domain.Match
		for _, item := range apiResp.Response {
			matchTime, _ := time.Parse(time.RFC3339, item.Fixture.Date)

			// Переводим статус в наш внутренний формат
			status := "SCHEDULED"
			if item.Fixture.Status.Short == "FT" {
				status = "FINISHED"
			} else if item.Fixture.Status.Short != "NS" {
				status = "LIVE"
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
			})
		}

		if len(matchesToSave) > 0 {
			if err := s.repo.SaveMatches(matchesToSave); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *MatchService) GetMatchesForUser(tgID int64) ([]domain.Match, error) {
	return s.repo.GetActiveMatches()
}

func (s *MatchService) ProcessFinishedMatches() error {
	activeMatches, err := s.repo.GetActiveMatches()
	if err != nil {
		return err
	}

	for _, m := range activeMatches {
		url := fmt.Sprintf("%s/fixtures?id=%d", s.baseURL, m.APIID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("x-apisports-key", s.apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		var apiResp APIFootballResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if len(apiResp.Response) == 0 {
			continue
		}

		item := apiResp.Response[0]

		if item.Fixture.Status.Short == "FT" && item.Goals.Home != nil && item.Goals.Away != nil {
			homeG := *item.Goals.Home
			awayG := *item.Goals.Away

			outcome := "X"
			if homeG > awayG {
				outcome = "1"
			} else if homeG < awayG {
				outcome = "2"
			}

			err = s.repo.UpdateMatchResults(m.APIID, homeG, awayG, outcome)
			if err != nil {
				continue
			}

			preds, err := s.repo.GetPredictionsByMatch(m.APIID)
			if err != nil {
				continue
			}

			for _, p := range preds {
				isCorrect := p.UserChoice == outcome
				_ = s.repo.UpdatePredictionStatus(p.ID, isCorrect)

				if isCorrect {
					_ = s.repo.AddUserPoints(p.UserID, 1)
				}
			}
		}
	}
	return nil
}
