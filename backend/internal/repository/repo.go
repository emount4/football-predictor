package repository

import (
	"database/sql"
	"errors"
	"football-predictor/internal/domain"
	"time"
)

// ==========================================
// USER REPOSITORY IMPLEMENTATION
// ==========================================
func (r *Repository) GetByTgID(tgID int64) (*domain.User, error) {
	query := `SELECT tg_id, username, display_name, photo_url, total_points FROM users WHERE tg_id = ?`
	var u domain.User
	err := r.db.QueryRow(query, tgID).Scan(&u.TgID, &u.Username, &u.DisplayName, &u.PhotoURL, &u.TotalPoints)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}
func (r *Repository) Create(u *domain.User) error {
	query := `INSERT INTO users (tg_id, username, display_name, photo_url, total_points) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, u.TgID, u.Username, u.DisplayName, u.PhotoURL, u.TotalPoints)
	return err
}

func (r *Repository) UpdateProfile(tgID int64, username, displayName, photoURL string) error {
	query := `UPDATE users SET username = ?, display_name = ?, photo_url = ? WHERE tg_id = ?`
	_, err := r.db.Exec(query, username, displayName, photoURL, tgID)
	return err
}

func (r *Repository) GetLeaderboard() ([]domain.User, error) {
	query := `SELECT tg_id, username, display_name, photo_url, total_points FROM users ORDER BY total_points DESC LIMIT 90`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.TgID, &u.Username, &u.DisplayName, &u.PhotoURL, &u.TotalPoints); err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, u)
	}
	return leaderboard, nil
}

func (r *Repository) AddUserPoints(tgID int64, points int) error {
	query := `UPDATE users SET total_points = total_points + ? WHERE tg_id = ?`
	_, err := r.db.Exec(query, points, tgID)
	return err
}

// ==========================================
// MATCH REPOSITORY IMPLEMENTATION
// ==========================================

func (r *Repository) SaveMatches(matches []domain.Match) error {
	query := `INSERT OR REPLACE INTO matches 
		(api_id, league_id, league_name, home_team, away_team, match_time, status, home_goals, away_goals, outcome) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, m := range matches {
		timeStr := m.MatchTime.Format(time.RFC3339) // SQLite любит строки для дат
		_, err := r.db.Exec(query, m.APIID, m.LeagueID, m.LeagueName, m.HomeTeam, m.AwayTeam, timeStr, m.Status, m.HomeGoals, m.AwayGoals, m.Outcome)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetActiveMatches() ([]domain.Match, error) {
	query := `SELECT api_id, league_id, league_name, home_team, away_team, match_time, status, home_goals, away_goals, outcome 
			  FROM matches WHERE status != 'FINISHED' ORDER BY match_time ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []domain.Match
	for rows.Next() {
		var m domain.Match
		var timeStr string
		err := rows.Scan(&m.APIID, &m.LeagueID, &m.LeagueName, &m.HomeTeam, &m.AwayTeam, &timeStr, &m.Status, &m.HomeGoals, &m.AwayGoals, &m.Outcome)
		if err != nil {
			return nil, err
		}
		m.MatchTime, _ = time.Parse(time.RFC3339, timeStr)
		matches = append(matches, m)
	}
	return matches, nil
}

func (r *Repository) GetMatchByID(apiID int64) (*domain.Match, error) {
	query := `SELECT api_id, league_id, league_name, home_team, away_team, match_time, status, home_goals, away_goals, outcome FROM matches WHERE api_id = ?`
	var m domain.Match
	var timeStr string
	err := r.db.QueryRow(query, apiID).Scan(&m.APIID, &m.LeagueID, &m.LeagueName, &m.HomeTeam, &m.AwayTeam, &timeStr, &m.Status, &m.HomeGoals, &m.AwayGoals, &m.Outcome)
	if err != nil {
		return nil, err
	}
	m.MatchTime, _ = time.Parse(time.RFC3339, timeStr)
	return &m, nil
}

func (r *Repository) UpdateMatchResults(matchID int64, homeGoals, awayGoals int, outcome string) error {
	query := `UPDATE matches SET home_goals = ?, away_goals = ?, outcome = ?, status = 'FINISHED' WHERE api_id = ?`
	_, err := r.db.Exec(query, homeGoals, awayGoals, outcome, matchID)
	return err
}

// ==========================================
// PREDICT REPOSITORY IMPLEMENTATION
// ==========================================

func (r *Repository) SavePrediction(p *domain.Prediction) error {
	// INSERT OR REPLACE позволяет пользователю менять свой прогноз до начала матча
	query := `INSERT OR REPLACE INTO predictions (user_id, match_id, user_choice) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, p.UserID, p.MatchID, p.UserChoice)
	return err
}

func (r *Repository) GetPredictionsByMatch(matchID int64) ([]domain.Prediction, error) {
	query := `SELECT id, user_id, match_id, user_choice, is_correct FROM predictions WHERE match_id = ?`
	rows, err := r.db.Query(query, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var preds []domain.Prediction
	for rows.Next() {
		var p domain.Prediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.MatchID, &p.UserChoice, &p.IsCorrect); err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, nil
}

func (r *Repository) UpdatePredictionStatus(predID int64, isCorrect bool) error {
	val := 0
	if isCorrect {
		val = 1
	}
	query := `UPDATE predictions SET is_correct = ? WHERE id = ?`
	_, err := r.db.Exec(query, val, predID)
	return err
}
