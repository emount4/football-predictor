package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"football-predictor/internal/repository"
	"football-predictor/internal/service"
	httptransport "football-predictor/internal/transport/http"
	"football-predictor/internal/worker"

	"github.com/joho/godotenv"
)

func main() {
	// Работает и при запуске из backend:
	//   go run ./cmd/server
	// и при запуске из корня:
	//   go run ./backend/cmd/server
	_ = godotenv.Load(".env", "backend/.env")

	dbPath := getEnv("DB_PATH", filepath.Join("data", "football.db"))
	schemaPath := getEnv("SCHEMA_PATH", filepath.Join("internal", "repository", "schema.sql"))

	botToken := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	apiKey := strings.TrimSpace(os.Getenv("API_SPORTS_KEY"))
	if apiKey == "" {
		log.Println("warning: API_SPORTS_KEY is empty; match sync from API-Sports will fail")
	}

	port := getEnv("PORT", "8080")
	leagues := parseLeagues(getEnv("API_LEAGUES", "39"))

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("failed to create db directory: %v", err)
	}

	repo, err := repository.NewRepository(dbPath, schemaPath)
	if err != nil {
		log.Fatalf("failed to initialize repository: %v", err)
	}

	authService := service.NewAuthService(repo, botToken)
	matchService := service.NewMatchService(repo, apiKey, leagues)
	predictService := service.NewPredictService(repo, repo)

	if getEnv("ENABLE_WORKER", "true") == "true" {
		worker.NewWorker(matchService).Start()
	}

	handler := httptransport.NewHandler(authService, matchService, predictService)
	router := handler.InitRoutes()

	log.Printf("HTTP server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func getEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func parseLeagues(raw string) []int {
	parts := strings.Split(raw, ",")
	leagues := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		leagueID, err := strconv.Atoi(part)
		if err != nil || leagueID <= 0 {
			continue
		}

		leagues = append(leagues, leagueID)
	}

	if len(leagues) == 0 {
		return []int{39}
	}

	return leagues
}
