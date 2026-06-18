package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"football-predictor/internal/domain"
	"strings"
	"time"
)

// ==========================================
// USER REPOSITORY IMPLEMENTATION
// ==========================================

func (r *Repository) GetByTgID(tgID int64) (*domain.User, error) {
	query := `
		SELECT tg_id, username, display_name, photo_url, total_points
		FROM users
		WHERE tg_id = ?
	`

	var u domain.User
	err := r.db.QueryRow(query, tgID).Scan(&u.TgID, &u.Username, &u.DisplayName, &u.PhotoURL, &u.TotalPoints)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (r *Repository) Create(u *domain.User) error {
	query := `
		INSERT INTO users (tg_id, username, display_name, photo_url, total_points)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, u.TgID, u.Username, u.DisplayName, u.PhotoURL, u.TotalPoints)
	return err
}

func (r *Repository) UpdateProfile(tgID int64, username, displayName, photoURL string) error {
	query := `
		UPDATE users
		SET username = ?, display_name = ?, photo_url = ?
		WHERE tg_id = ?
	`
	_, err := r.db.Exec(query, username, displayName, photoURL, tgID)
	return err
}

func (r *Repository) GetLeaderboard() ([]domain.User, error) {
	query := `
		SELECT tg_id, username, display_name, photo_url, total_points
		FROM users
		ORDER BY total_points DESC, tg_id ASC
		LIMIT 90
	`

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
	return leaderboard, rows.Err()
}

func (r *Repository) GetLeaderboardWithRanks(limit int) ([]domain.LeaderboardItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	query := `
		SELECT username, display_name, photo_url, total_points
		FROM users
		ORDER BY total_points DESC, tg_id ASC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.LeaderboardItem, 0, limit)
	rank := 1
	for rows.Next() {
		var item domain.LeaderboardItem
		if err := rows.Scan(&item.Username, &item.DisplayName, &item.PhotoURL, &item.TotalPoints); err != nil {
			return nil, err
		}
		item.Rank = rank
		items = append(items, item)
		rank++
	}
	return items, rows.Err()
}

func (r *Repository) GetMeStats(tgID int64) (*domain.MeResponse, error) {
	user, err := r.GetByTgID(tgID)
	if err != nil || user == nil {
		return nil, err
	}

	var rank int
	if err := r.db.QueryRow(`SELECT COUNT(*) + 1 FROM users WHERE total_points > ?`, user.TotalPoints).Scan(&rank); err != nil {
		return nil, err
	}

	var predictionsCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM predictions WHERE user_id = ?`, tgID).Scan(&predictionsCount); err != nil {
		return nil, err
	}

	var correctPredictions int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM predictions WHERE user_id = ? AND is_correct = 1`, tgID).Scan(&correctPredictions); err != nil {
		return nil, err
	}

	return &domain.MeResponse{
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		PhotoURL:           user.PhotoURL,
		TotalPoints:        user.TotalPoints,
		Rank:               rank,
		PredictionsCount:   predictionsCount,
		CorrectPredictions: correctPredictions,
	}, nil
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
	query := `
		INSERT INTO matches (
			api_id, league_id, league_name, home_team, away_team, match_time,
			status, home_goals, away_goals, outcome
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(api_id) DO UPDATE SET
			league_id = excluded.league_id,
			league_name = excluded.league_name,
			home_team = excluded.home_team,
			away_team = excluded.away_team,
			match_time = excluded.match_time,
			status = excluded.status,
			home_goals = excluded.home_goals,
			away_goals = excluded.away_goals,
			outcome = excluded.outcome
	`

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range matches {
		_, err := stmt.Exec(
			m.APIID,
			m.LeagueID,
			m.LeagueName,
			m.HomeTeam,
			m.AwayTeam,
			m.MatchTime.Format(time.RFC3339),
			m.Status,
			m.HomeGoals,
			m.AwayGoals,
			nullableOutcome(m.Outcome),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetActiveMatches() ([]domain.Match, error) {
	return r.GetMatches("active")
}

func (r *Repository) GetMatches(status string) ([]domain.Match, error) {
	where, args := buildMatchStatusFilter(status, "")
	query := fmt.Sprintf(`
		SELECT api_id, league_id, league_name, home_team, away_team, match_time,
		       status, home_goals, away_goals, outcome
		FROM matches
		%s
		ORDER BY match_time ASC
	`, where)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]domain.Match, 0)
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *Repository) GetMatchesForUser(tgID int64, status string, page, limit int) (*domain.MatchesPage, error) {
	where, args := buildMatchStatusFilter(status, "m")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM matches m %s`, where)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	queryArgs := append([]any{tgID}, args...)
	queryArgs = append(queryArgs, limit, offset)

	query := fmt.Sprintf(`
		SELECT m.api_id, m.league_id, m.league_name, m.home_team, m.away_team, m.match_time,
		       m.status, m.home_goals, m.away_goals, m.outcome,
		       p.user_choice
		FROM matches m
		LEFT JOIN predictions p ON p.match_id = m.api_id AND p.user_id = ?
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, where, buildMatchOrderBy(status, "m"))

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.MatchForUser, 0)
	for rows.Next() {
		item, err := scanMatchForUser(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &domain.MatchesPage{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *Repository) GetMatchByID(apiID int64) (*domain.Match, error) {
	query := `
		SELECT api_id, league_id, league_name, home_team, away_team, match_time,
		       status, home_goals, away_goals, outcome
		FROM matches
		WHERE api_id = ?
	`

	row := r.db.QueryRow(query, apiID)
	m, err := scanMatch(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) GetMatchForUser(apiID int64, tgID int64) (*domain.MatchForUser, error) {
	query := `
		SELECT m.api_id, m.league_id, m.league_name, m.home_team, m.away_team, m.match_time,
		       m.status, m.home_goals, m.away_goals, m.outcome,
		       p.user_choice
		FROM matches m
		LEFT JOIN predictions p ON p.match_id = m.api_id AND p.user_id = ?
		WHERE m.api_id = ?
	`

	row := r.db.QueryRow(query, tgID, apiID)
	item, err := scanMatchForUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpdateMatchResults(matchID int64, homeGoals, awayGoals int, outcome string) error {
	query := `
		UPDATE matches
		SET home_goals = ?, away_goals = ?, outcome = ?, status = 'FINISHED'
		WHERE api_id = ?
	`
	_, err := r.db.Exec(query, homeGoals, awayGoals, outcome, matchID)
	return err
}

func (r *Repository) GetLeagues() ([]domain.LeagueInfo, error) {
	query := `
		SELECT league_id, league_name
		FROM matches
		GROUP BY league_id, league_name
		ORDER BY league_name ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.LeagueInfo, 0)
	for rows.Next() {
		var item domain.LeagueInfo
		if err := rows.Scan(&item.LeagueID, &item.LeagueName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetAdminStats() (*domain.AdminStats, error) {
	stats := &domain.AdminStats{}

	queries := []struct {
		target *int
		query  string
	}{
		{&stats.UsersCount, `SELECT COUNT(*) FROM users`},
		{&stats.MatchesCount, `SELECT COUNT(*) FROM matches`},
		{&stats.ActiveMatches, `SELECT COUNT(*) FROM matches WHERE status != 'FINISHED'`},
		{&stats.FinishedMatches, `SELECT COUNT(*) FROM matches WHERE status = 'FINISHED'`},
		{&stats.PredictionsCount, `SELECT COUNT(*) FROM predictions`},
	}

	for _, q := range queries {
		if err := r.db.QueryRow(q.query).Scan(q.target); err != nil {
			return nil, err
		}
	}

	return stats, nil
}

// ==========================================
// PREDICT REPOSITORY IMPLEMENTATION
// ==========================================

func (r *Repository) SavePrediction(p *domain.Prediction) error {
	query := `
		INSERT INTO predictions (user_id, match_id, user_choice, is_correct)
		VALUES (?, ?, ?, NULL)
		ON CONFLICT(user_id, match_id) DO UPDATE SET
			user_choice = excluded.user_choice,
			is_correct = NULL
	`
	_, err := r.db.Exec(query, p.UserID, p.MatchID, p.UserChoice)
	return err
}

func (r *Repository) GetPredictionsByMatch(matchID int64) ([]domain.Prediction, error) {
	query := `
		SELECT id, user_id, match_id, user_choice, is_correct
		FROM predictions
		WHERE match_id = ?
	`

	rows, err := r.db.Query(query, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	preds := make([]domain.Prediction, 0)
	for rows.Next() {
		p, err := scanPrediction(rows)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

func (r *Repository) GetPredictionStatsByMatch(matchID int64) (*domain.PredictionStats, error) {
	stats := &domain.PredictionStats{
		MatchID: matchID,
		Choices: map[string]int{"1": 0, "X": 0, "2": 0},
		Percent: map[string]float64{"1": 0, "X": 0, "2": 0},
	}

	query := `
		SELECT user_choice, COUNT(*)
		FROM predictions
		WHERE match_id = ?
		GROUP BY user_choice
	`

	rows, err := r.db.Query(query, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var choice string
		var count int
		if err := rows.Scan(&choice, &count); err != nil {
			return nil, err
		}
		stats.Choices[choice] = count
		stats.Total += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if stats.Total > 0 {
		for choice, count := range stats.Choices {
			stats.Percent[choice] = float64(count) * 100 / float64(stats.Total)
		}
	}

	voters, err := r.GetPredictionVotersByMatch(matchID)
	if err != nil {
		return nil, err
	}
	stats.Voters = voters

	return stats, nil
}

func (r *Repository) GetPredictionVotersByMatch(matchID int64) ([]domain.PredictionVoter, error) {
	query := `
		SELECT u.username, u.display_name, u.photo_url, p.user_choice
		FROM predictions p
		JOIN users u ON u.tg_id = p.user_id
		WHERE p.match_id = ?
		ORDER BY p.user_choice, COALESCE(NULLIF(u.display_name, ''), u.username)
	`

	rows, err := r.db.Query(query, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	voters := make([]domain.PredictionVoter, 0)
	for rows.Next() {
		var v domain.PredictionVoter
		if err := rows.Scan(&v.Username, &v.DisplayName, &v.PhotoURL, &v.UserChoice); err != nil {
			return nil, err
		}
		voters = append(voters, v)
	}

	return voters, rows.Err()
}

func (r *Repository) GetUserPredictionHistory(tgID int64, status string) ([]domain.PredictionHistoryItem, error) {
	where, args := buildMatchStatusFilter(status, "m")
	args = append([]any{tgID}, args...)

	query := fmt.Sprintf(`
		SELECT m.api_id, m.home_team, m.away_team, m.league_id, m.league_name,
		       m.match_time, m.status, m.home_goals, m.away_goals, m.outcome,
		       p.user_choice, p.is_correct
		FROM predictions p
		JOIN matches m ON m.api_id = p.match_id
		%s
		%s
		ORDER BY m.match_time DESC
	`, firstWhere(where, "p.user_id = ?"), andWithoutWhere(where))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.PredictionHistoryItem, 0)
	for rows.Next() {
		var item domain.PredictionHistoryItem
		var timeStr string
		var homeGoals, awayGoals sql.NullInt64
		var outcome sql.NullString
		var isCorrect sql.NullInt64

		if err := rows.Scan(
			&item.MatchID,
			&item.HomeTeam,
			&item.AwayTeam,
			&item.LeagueID,
			&item.LeagueName,
			&timeStr,
			&item.Status,
			&homeGoals,
			&awayGoals,
			&outcome,
			&item.UserChoice,
			&isCorrect,
		); err != nil {
			return nil, err
		}

		item.MatchTime, _ = time.Parse(time.RFC3339, timeStr)
		item.HomeGoals = nullableIntPtr(homeGoals)
		item.AwayGoals = nullableIntPtr(awayGoals)
		if outcome.Valid {
			item.Outcome = outcome.String
		}
		item.IsCorrect = nullableBoolPtr(isCorrect)
		if item.IsCorrect != nil && *item.IsCorrect {
			item.PointsAwarded = 1
		}

		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) UpdatePredictionStatus(predID int64, isCorrect bool) error {
	var userID int64
	if err := r.db.QueryRow(`SELECT user_id FROM predictions WHERE id = ?`, predID).Scan(&userID); err != nil {
		return err
	}
	return r.SetPredictionResult(predID, userID, isCorrect)
}

func (r *Repository) SetPredictionResult(predID int64, userID int64, isCorrect bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var oldValue sql.NullInt64
	if err := tx.QueryRow(`SELECT is_correct FROM predictions WHERE id = ? AND user_id = ?`, predID, userID).Scan(&oldValue); err != nil {
		return err
	}

	oldPoints := 0
	if oldValue.Valid && oldValue.Int64 == 1 {
		oldPoints = 1
	}

	newInt := 0
	newPoints := 0
	if isCorrect {
		newInt = 1
		newPoints = 1
	}

	if _, err := tx.Exec(`UPDATE predictions SET is_correct = ? WHERE id = ?`, newInt, predID); err != nil {
		return err
	}

	if delta := newPoints - oldPoints; delta != 0 {
		if _, err := tx.Exec(`UPDATE users SET total_points = total_points + ? WHERE tg_id = ?`, delta, userID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ==========================================
// HELPERS
// ==========================================

type scanner interface {
	Scan(dest ...any) error
}

func scanMatch(row scanner) (domain.Match, error) {
	var m domain.Match
	var timeStr string
	var homeGoals, awayGoals sql.NullInt64
	var outcome sql.NullString

	err := row.Scan(
		&m.APIID,
		&m.LeagueID,
		&m.LeagueName,
		&m.HomeTeam,
		&m.AwayTeam,
		&timeStr,
		&m.Status,
		&homeGoals,
		&awayGoals,
		&outcome,
	)
	if err != nil {
		return m, err
	}

	m.MatchTime, _ = time.Parse(time.RFC3339, timeStr)
	m.HomeGoals = nullableIntPtr(homeGoals)
	m.AwayGoals = nullableIntPtr(awayGoals)
	if outcome.Valid {
		m.Outcome = outcome.String
	}

	return m, nil
}

func scanMatchForUser(row scanner) (domain.MatchForUser, error) {
	var item domain.MatchForUser
	var timeStr string
	var homeGoals, awayGoals sql.NullInt64
	var outcome sql.NullString
	var myPrediction sql.NullString

	err := row.Scan(
		&item.APIID,
		&item.LeagueID,
		&item.LeagueName,
		&item.HomeTeam,
		&item.AwayTeam,
		&timeStr,
		&item.Status,
		&homeGoals,
		&awayGoals,
		&outcome,
		&myPrediction,
	)
	if err != nil {
		return item, err
	}

	item.MatchTime, _ = time.Parse(time.RFC3339, timeStr)
	item.HomeGoals = nullableIntPtr(homeGoals)
	item.AwayGoals = nullableIntPtr(awayGoals)
	if outcome.Valid {
		item.Outcome = outcome.String
	}
	if myPrediction.Valid {
		item.MyPrediction = &myPrediction.String
	}
	item.PredictionLocked = time.Now().After(item.MatchTime) || item.Status != "SCHEDULED"

	return item, nil
}

func scanPrediction(row scanner) (domain.Prediction, error) {
	var p domain.Prediction
	var isCorrect sql.NullInt64

	err := row.Scan(&p.ID, &p.UserID, &p.MatchID, &p.UserChoice, &isCorrect)
	if err != nil {
		return p, err
	}
	p.IsCorrect = nullableBoolPtr(isCorrect)
	return p, nil
}

func nullableIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}

func nullableBoolPtr(value sql.NullInt64) *bool {
	if !value.Valid {
		return nil
	}
	v := value.Int64 != 0
	return &v
}

func nullableOutcome(outcome string) any {
	if outcome == "" {
		return nil
	}
	return outcome
}

func buildMatchOrderBy(status string, alias string) string {
	column := "match_time"
	if alias != "" {
		column = alias + ".match_time"
	}

	if strings.ToLower(strings.TrimSpace(status)) == "finished" {
		return column + " DESC"
	}

	return column + " ASC"
}

func buildMatchStatusFilter(status string, alias string) (string, []any) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "active", "upcoming":
		return fmt.Sprintf("WHERE %sstatus != 'FINISHED'", prefix), nil
	case "finished":
		return fmt.Sprintf("WHERE %sstatus = 'FINISHED'", prefix), nil
	case "scheduled":
		return fmt.Sprintf("WHERE %sstatus = 'SCHEDULED'", prefix), nil
	case "live":
		return fmt.Sprintf("WHERE %sstatus = 'LIVE'", prefix), nil
	case "all":
		return "", nil
	default:
		return fmt.Sprintf("WHERE %sstatus != 'FINISHED'", prefix), nil
	}
}

func firstWhere(existingWhere string, condition string) string {
	if strings.TrimSpace(existingWhere) == "" {
		return "WHERE " + condition
	}
	return "WHERE " + condition
}

func andWithoutWhere(existingWhere string) string {
	existingWhere = strings.TrimSpace(existingWhere)
	if existingWhere == "" {
		return ""
	}
	return "AND " + strings.TrimPrefix(existingWhere, "WHERE ")
}
