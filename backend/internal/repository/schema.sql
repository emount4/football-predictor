-- Включаем поддержку внешних ключей (в SQLite по умолчанию отключена)
PRAGMA foreign_keys = ON;

-- 1. Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    tg_id INTEGER PRIMARY KEY,           -- Telegram ID (уникальный для каждого)
    username TEXT NOT NULL DEFAULT '',   -- @username из Телеграма
    display_name TEXT NOT NULL,          -- Имя + Фамилия (First/Last name)
    photo_url TEXT NOT NULL DEFAULT '',
    total_points INTEGER NOT NULL DEFAULT 0
);

-- Index для быстрого вывода топ-игроков
CREATE INDEX IF NOT EXISTS idx_users_points ON users(total_points DESC);

-- 2. Таблица футбольных матчей
CREATE TABLE IF NOT EXISTS matches (
    api_id INTEGER PRIMARY KEY,          -- ID матча, который отдает внешнее спортивное API
    league_id INTEGER NOT NULL,          -- ID лиги из внешнего API (например, 39 для АПЛ)
    league_name TEXT NOT NULL,           -- Название лиги ("Premier League", "Champions League")
    home_team TEXT NOT NULL,             -- Название первой команды
    away_team TEXT NOT NULL,             -- Название второй команды
    match_time TEXT NOT NULL,            -- Время начала в формате ISO8601 (строка)
    status TEXT NOT NULL DEFAULT 'SCHEDULED', -- 'SCHEDULED', 'LIVE', 'FINISHED'
    home_goals INTEGER DEFAULT NULL,     -- Заполняется после матча
    away_goals INTEGER DEFAULT NULL,     -- Заполняется после матча
    outcome TEXT DEFAULT NULL            -- Результат: '1', 'X', '2' (заполняется после матча)
);

-- 3. Таблица прогнозов пользователей
CREATE TABLE IF NOT EXISTS predictions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    match_id INTEGER NOT NULL,
    user_choice TEXT NOT NULL,           -- На что ставил: '1', 'X' или '2'
    is_correct INTEGER DEFAULT NULL,     -- NULL (матч идет), 1 (угадал), 0 (не угадал)
    
    FOREIGN KEY (user_id) REFERENCES users(tg_id) ON DELETE CASCADE,
    FOREIGN KEY (match_id) REFERENCES matches(api_id) ON DELETE CASCADE,
    -- Ограничение: один пользователь может сделать только один прогноз на один матч
    UNIQUE(user_id, match_id)
);