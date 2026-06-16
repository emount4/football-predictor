PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    tg_id INTEGER PRIMARY KEY,
    username TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL,
    photo_url TEXT NOT NULL DEFAULT '',
    total_points INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_points ON users(total_points DESC);

CREATE TABLE IF NOT EXISTS matches (
    api_id INTEGER PRIMARY KEY,
    league_id INTEGER NOT NULL,
    league_name TEXT NOT NULL,
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    match_time TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'SCHEDULED',
    home_goals INTEGER DEFAULT NULL,
    away_goals INTEGER DEFAULT NULL,
    outcome TEXT DEFAULT NULL CHECK (outcome IS NULL OR outcome IN ('1', 'X', '2'))
);

CREATE INDEX IF NOT EXISTS idx_matches_status_time ON matches(status, match_time);
CREATE INDEX IF NOT EXISTS idx_matches_league_time ON matches(league_id, match_time);

CREATE TABLE IF NOT EXISTS predictions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    match_id INTEGER NOT NULL,
    user_choice TEXT NOT NULL CHECK (user_choice IN ('1', 'X', '2')),
    is_correct INTEGER DEFAULT NULL CHECK (is_correct IS NULL OR is_correct IN (0, 1)),
    FOREIGN KEY (user_id) REFERENCES users(tg_id) ON DELETE CASCADE,
    FOREIGN KEY (match_id) REFERENCES matches(api_id) ON DELETE CASCADE,
    UNIQUE(user_id, match_id)
);

CREATE INDEX IF NOT EXISTS idx_predictions_user ON predictions(user_id);
CREATE INDEX IF NOT EXISTS idx_predictions_match ON predictions(match_id);
