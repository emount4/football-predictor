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
	LeagueID   int       `json:"league_id"`   // Новое поле
	LeagueName string    `json:"league_name"` // Новое поле
	MatchTime  time.Time `json:"match_time"`
	Status     string    `json:"status"`     // "SCHEDULED", "LIVE", "FINISHED"
	HomeGoals  *int      `json:"home_goals"` // указатели, так как до матча голов нет (NULL в БД)
	AwayGoals  *int      `json:"away_goals"`
	Outcome    string    `json:"outcome"` // "1", "X", "2" или ""
}

type Prediction struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	MatchID    int64  `json:"match_id"`
	UserChoice string `json:"user_choice"` // "1", "X", "2"
	IsCorrect  *bool  `json:"is_correct"`  // NULL пока матч идет, затем true/false
}

// UserRepository описывает, что должен уметь делать слой данных с юзерами
type UserRepository interface {
	GetByTgID(tgID int64) (*User, error)
	Create(user *User) error
	GetLeaderboard() ([]User, error)
}

// MatchRepository работает с матчами в SQLite
type MatchRepository interface {
	SaveMatches(matches []Match) error
	GetActiveMatches() ([]Match, error)
	GetMatchByID(apiID int64) (*Match, error)
	UpdateMatchResults(matchID int64, homeGoals, awayGoals int, outcome string) error
}

// PredictRepository работает со ставками
type PredictRepository interface {
	SavePrediction(pred *Prediction) error
	GetPredictionsByMatch(matchID int64) ([]Prediction, error)
	UpdatePredictionStatus(predID int64, isCorrect bool) error
}

// MatchService описывает бизнес-логику (вызывается из Gin и Воркеров)
type MatchService interface {
	GetMatchesForUser(tgID int64) ([]Match, error) // Получить матчи + статус, делал ли юзер ставку
	FetchDailyMatches() error                      // Метод для воркера — стянуть матчи из внешнего API
	ProcessFinishedMatches() error                 // Метод для воркера — проверить результаты и начислить очки
}

// PredictService логика отправки прогнозов
type PredictService interface {
	MakePrediction(tgID int64, matchID int64, choice string) error // Тут проверяем, не начался ли матч
}
