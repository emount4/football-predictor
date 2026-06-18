package domain

import "time"

type User struct {
	TgID        int64  `json:"tg_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	PhotoURL    string `json:"photo_url"`
	TotalPoints int    `json:"total_points"`
}

type Match struct {
	APIID      int64     `json:"api_id"`
	HomeTeam   string    `json:"home_team"`
	AwayTeam   string    `json:"away_team"`
	LeagueID   int       `json:"league_id"`
	LeagueName string    `json:"league_name"`
	MatchTime  time.Time `json:"match_time"`
	Status     string    `json:"status"` // SCHEDULED, LIVE, FINISHED
	HomeGoals  *int      `json:"home_goals"`
	AwayGoals  *int      `json:"away_goals"`
	Outcome    string    `json:"outcome"` // 1, X, 2 или пустая строка до завершения
}

type Prediction struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	MatchID    int64  `json:"match_id"`
	UserChoice string `json:"user_choice"` // 1, X, 2
	IsCorrect  *bool  `json:"is_correct"`  // nil до завершения матча
}

type MatchForUser struct {
	Match
	MyPrediction     *string `json:"my_prediction"`
	PredictionLocked bool    `json:"prediction_locked"`
}

type MatchesPage struct {
	Items []MatchForUser `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

type PredictionHistoryItem struct {
	MatchID       int64     `json:"match_id"`
	HomeTeam      string    `json:"home_team"`
	AwayTeam      string    `json:"away_team"`
	LeagueID      int       `json:"league_id"`
	LeagueName    string    `json:"league_name"`
	MatchTime     time.Time `json:"match_time"`
	Status        string    `json:"status"`
	HomeGoals     *int      `json:"home_goals"`
	AwayGoals     *int      `json:"away_goals"`
	Outcome       string    `json:"outcome"`
	UserChoice    string    `json:"user_choice"`
	IsCorrect     *bool     `json:"is_correct"`
	PointsAwarded int       `json:"points_awarded"`
}

type PredictionVoter struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	PhotoURL    string `json:"photo_url"`
	UserChoice  string `json:"user_choice"`
}

type PredictionStats struct {
	MatchID int64              `json:"match_id"`
	Total   int                `json:"total"`
	Choices map[string]int     `json:"choices"`
	Percent map[string]float64 `json:"percent"`
	Voters  []PredictionVoter  `json:"voters"`
}

type LeaderboardItem struct {
	Rank        int    `json:"rank"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	PhotoURL    string `json:"photo_url"`
	TotalPoints int    `json:"total_points"`
}

type MeResponse struct {
	Username           string `json:"username"`
	DisplayName        string `json:"display_name"`
	PhotoURL           string `json:"photo_url"`
	TotalPoints        int    `json:"total_points"`
	Rank               int    `json:"rank"`
	PredictionsCount   int    `json:"predictions_count"`
	CorrectPredictions int    `json:"correct_predictions"`
}

type LeagueInfo struct {
	LeagueID   int    `json:"league_id"`
	LeagueName string `json:"league_name"`
}

type RulesResponse struct {
	PredictionChoices []string       `json:"prediction_choices"`
	Points            map[string]int `json:"points"`
	LockRule          string         `json:"lock_rule"`
}

type AdminStats struct {
	UsersCount       int `json:"users_count"`
	MatchesCount     int `json:"matches_count"`
	ActiveMatches    int `json:"active_matches"`
	FinishedMatches  int `json:"finished_matches"`
	PredictionsCount int `json:"predictions_count"`
}

type UserRepository interface {
	GetByTgID(tgID int64) (*User, error)
	Create(user *User) error
	UpdateProfile(tgID int64, username, displayName, photoURL string) error
	GetLeaderboard() ([]User, error)
	GetLeaderboardWithRanks(limit int) ([]LeaderboardItem, error)
	GetMeStats(tgID int64) (*MeResponse, error)
	AddUserPoints(tgID int64, points int) error
}

type MatchRepository interface {
	SaveMatches(matches []Match) error
	GetActiveMatches() ([]Match, error)
	GetMatches(status string) ([]Match, error)
	GetMatchesForUser(tgID int64, status string, page, limit int) (*MatchesPage, error)
	GetMatchByID(apiID int64) (*Match, error)
	GetMatchForUser(apiID int64, tgID int64) (*MatchForUser, error)
	GetLeagues() ([]LeagueInfo, error)
	GetAdminStats() (*AdminStats, error)
	UpdateMatchResults(matchID int64, homeGoals, awayGoals int, outcome string) error
}

type PredictRepository interface {
	SavePrediction(pred *Prediction) error
	GetPredictionsByMatch(matchID int64) ([]Prediction, error)
	GetPredictionStatsByMatch(matchID int64) (*PredictionStats, error)
	GetPredictionVotersByMatch(matchID int64) ([]PredictionVoter, error)
	GetUserPredictionHistory(tgID int64, status string) ([]PredictionHistoryItem, error)
	UpdatePredictionStatus(predID int64, isCorrect bool) error
	SetPredictionResult(predID int64, userID int64, isCorrect bool) error
}

type MatchService interface {
	GetMatchesForUser(tgID int64, status string, page, limit int) (*MatchesPage, error)
	GetMatchForUser(tgID int64, matchID int64) (*MatchForUser, error)
	GetPredictionStats(matchID int64) (*PredictionStats, error)
	GetPredictionHistory(tgID int64, status string) ([]PredictionHistoryItem, error)
	GetLeaderboard(limit int) ([]LeaderboardItem, error)
	GetMe(tgID int64) (*MeResponse, error)
	GetLeagues() ([]LeagueInfo, error)
	GetRules() RulesResponse
	GetAdminStats() (*AdminStats, error)
	FetchDailyMatches() error
	ProcessFinishedMatches() error
	RecalculateMatch(matchID int64) error
}

type PredictService interface {
	MakePrediction(tgID int64, matchID int64, choice string) error
}
