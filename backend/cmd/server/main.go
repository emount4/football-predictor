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
)

func main() {
	dbPath := getEnv("DB_PATH", filepath.Join("data", "football.db"))
	schemaPath := getEnv("SCHEMA_PATH", filepath.Join("internal", "repository", "schema.sql"))
	botToken := os.Getenv("BOT_TOKEN")
	apiKey := os.Getenv("API_SPORTS_KEY")
	port := getEnv("PORT", "8080")
	leagues := parseLeagues(getEnv("API_LEAGUES", "39"))

	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}
	if apiKey == "" {
		log.Fatal("API_SPORTS_KEY is required")
	}

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

	worker.NewWorker(matchService).Start()

	handler := httptransport.NewHandler(authService, matchService, predictService)
	router := handler.InitRoutes()

	log.Printf("HTTP server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func getEnv(name, fallback string) string {
	value := os.Getenv(name)
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
		if err != nil {
			continue
		}

		leagues = append(leagues, leagueID)
	}

	if len(leagues) == 0 {
		return []int{39}
	}

	return leagues
}
