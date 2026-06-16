package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"football-predictor/internal/repository"
	"football-predictor/internal/service"
	httptransport "football-predictor/internal/transport/http"
	"football-predictor/internal/worker"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env", "backend/.env")

	dbPath := getEnv("DB_PATH", filepath.Join("data", "football.db"))
	schemaPath := getEnv("SCHEMA_PATH", filepath.Join("internal", "repository", "schema.sql"))

	botToken := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	footballDataToken := strings.TrimSpace(os.Getenv("FOOTBALL_DATA_TOKEN"))
	if footballDataToken == "" {
		log.Fatal("FOOTBALL_DATA_TOKEN is required")
	}

	port := getEnv("PORT", "8080")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("failed to create db directory: %v", err)
	}

	repo, err := repository.NewRepository(dbPath, schemaPath)
	if err != nil {
		log.Fatalf("failed to initialize repository: %v", err)
	}

	authService := service.NewAuthService(repo, botToken)
	matchService := service.NewMatchService(repo, footballDataToken)
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
