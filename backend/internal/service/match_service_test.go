package service

import (
	"fmt"
	"football-predictor/internal/domain"
	"football-predictor/internal/repository"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMatchService_FullFlow(t *testing.T) {
	// 1. Создаем схему
	schemaSQL := `
	CREATE TABLE IF NOT EXISTS users (tg_id INTEGER PRIMARY KEY, username TEXT, display_name TEXT, photo_url TEXT DEFAULT '', total_points INTEGER DEFAULT 0);
	CREATE TABLE IF NOT EXISTS matches (api_id INTEGER PRIMARY KEY, league_id INTEGER, league_name TEXT, home_team TEXT, away_team TEXT, match_time TEXT, status TEXT, home_goals INTEGER, away_goals INTEGER, outcome TEXT);
	CREATE TABLE IF NOT EXISTS predictions (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, match_id INTEGER, user_choice TEXT, is_correct INTEGER, UNIQUE(user_id, match_id));
	`
	tmpSchema, err := os.CreateTemp("", "schema_*.sql")
	if err != nil {
		t.Fatalf("Не удалось создать временную схему: %v", err)
	}
	defer os.Remove(tmpSchema.Name())

	// На Windows важно закрыть файл после записи, чтобы репозиторий мог его прочесть
	_, _ = tmpSchema.Write([]byte(schemaSQL))
	tmpSchema.Close()

	// 2. Инициализируем тестовую БД
	tmpDB, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Не удалось создать тестовую БД: %v", err)
	}
	// Закрываем дескриптор сразу, чтобы SQLite мог монопольно открыть файл
	tmpDB.Close()
	defer os.Remove(tmpDB.Name())

	repo, err := repository.NewRepository(tmpDB.Name(), tmpSchema.Name())
	if err != nil {
		t.Fatalf("Ошибка инициализации репозитория: %v", err)
	}

	// 3. Создаем тестовые данные
	testUser := &domain.User{TgID: 12345, Username: "test_gamer", DisplayName: "Иван Тестов", TotalPoints: 0}
	if err := repo.Create(testUser); err != nil {
		t.Fatalf("Не удалось создать юзера: %v", err)
	}

	pred := &domain.Prediction{UserID: 12345, MatchID: 999, UserChoice: "1"}
	if err := repo.SavePrediction(pred); err != nil {
		t.Fatalf("Не удалось создать прогноз: %v", err)
	}

	// 4. Запускаем фейковый сервер
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		if requestCount == 1 {
			fmt.Fprint(w, `{
				"response": [{
					"fixture": {"id": 999, "date": "2026-06-16T21:00:00+00:00", "status": {"short": "1H"}},
					"league": {"id": 39, "name": "Premier League"},
					"teams": {"home": {"name": "Arsenal"}, "away": {"name": "Chelsea"}},
					"goals": {"home": 1, "away": 0}
				}]
			}`)
		} else {
			fmt.Fprint(w, `{
				"response": [{
					"fixture": {"id": 999, "date": "2026-06-16T21:00:00+00:00", "status": {"short": "FT"}},
					"league": {"id": 39, "name": "Premier League"},
					"teams": {"home": {"name": "Arsenal"}, "away": {"name": "Chelsea"}},
					"goals": {"home": 2, "away": 1}
				}]
			}`)
		}
	}))
	defer server.Close()

	// 5. Настраиваем сервис
	service := NewMatchService(repo, "fake_api_key", []int{39})
	service.baseURL = server.URL // Перенаправляем на локальный сервер

	// Запуск стягивания
	err = service.FetchDailyMatches()
	if err != nil {
		t.Fatalf("FetchDailyMatches вернул ошибку: %v", err)
	}

	// ПРОВЕРКА: Разделяем ошибку получения и пустоту в базе
	match, err := repo.GetMatchByID(999)
	if err != nil {
		t.Fatalf("Ошибка при обращении к БД за матчем: %v", err)
	}
	if match == nil {
		t.Fatalf("Матч вообще не найден в БД (FetchDailyMatches записал 0 матчей)")
	}

	if match.Status != "LIVE" {
		t.Errorf("Ожидался статус LIVE, получили %s", match.Status)
	}

	// Обработка результатов
	err = service.ProcessFinishedMatches()
	if err != nil {
		t.Fatalf("ProcessFinishedMatches вернул ошибку: %v", err)
	}

	updatedMatch, _ := repo.GetMatchByID(999)
	if updatedMatch.Status != "FINISHED" || updatedMatch.Outcome != "1" {
		t.Errorf("Матч не обновился корректно: статус %s, исход %s", updatedMatch.Status, updatedMatch.Outcome)
	}

	updatedUser, _ := repo.GetByTgID(12345)
	if updatedUser.TotalPoints != 1 {
		t.Errorf("Очки не начислились! У юзера: %d", updatedUser.TotalPoints)
	} else {
		fmt.Println("🎉 Успех! Интеграционный тест пройден, очки начислены!")
	}
}
