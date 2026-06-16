package http

import (
	"football-predictor/internal/domain"
	"football-predictor/internal/service"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService    *service.AuthService
	matchService   *service.MatchService
	predictService domain.PredictService
}

func NewHandler(
	authService *service.AuthService,
	ms *service.MatchService,
	predictService domain.PredictService,
) *Handler {
	return &Handler{
		authService:    authService,
		matchService:   ms,
		predictService: predictService,
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	r := gin.Default()

	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	protected := api.Group("", h.telegramAuthMiddlewareWrapper())

	protected.GET("/auth", h.handleAuth)
	protected.POST("/auth", h.handleAuth)

	protected.GET("/me", h.getMe)
	protected.GET("/rules", h.getRules)
	protected.GET("/leaderboard", h.getLeaderboard)
	protected.GET("/leagues", h.getLeagues)

	protected.GET("/matches", h.getMatches)
	protected.GET("/matches/:id", h.getMatch)
	protected.GET("/matches/:id/prediction-stats", h.getPredictionStats)

	protected.POST("/predictions", h.submitPrediction)
	protected.GET("/predictions/me", h.getMyPredictions)

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(os.Getenv("CORS_ORIGIN"))
		if origin == "" {
			origin = "*"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (h *Handler) telegramAuthMiddlewareWrapper() gin.HandlerFunc {
	return TelegramAuthMiddleware(h.authService)
}

func (h *Handler) handleAuth(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) getMe(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	me, err := h.matchService.GetMe(user.TgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить профиль"})
		return
	}
	if me == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	c.JSON(http.StatusOK, me)
}

func (h *Handler) getRules(c *gin.Context) {
	c.JSON(http.StatusOK, h.matchService.GetRules())
}

func (h *Handler) getLeaderboard(c *gin.Context) {
	limit := queryInt(c, "limit", 100)

	leaderboard, err := h.matchService.GetLeaderboard(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить рейтинг"})
		return
	}

	c.JSON(http.StatusOK, leaderboard)
}

func (h *Handler) getLeagues(c *gin.Context) {
	leagues, err := h.matchService.GetLeagues()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить лиги"})
		return
	}

	c.JSON(http.StatusOK, leagues)
}

func (h *Handler) getMatches(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	status := normalizeStatus(c.Query("status"))

	matches, err := h.matchService.GetMatchesForUser(user.TgID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить матчи"})
		return
	}

	c.JSON(http.StatusOK, matches)
}

func (h *Handler) getMatch(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	matchID, ok := pathInt64(c, "id")
	if !ok {
		return
	}

	match, err := h.matchService.GetMatchForUser(user.TgID, matchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить матч"})
		return
	}
	if match == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Матч не найден"})
		return
	}

	stats, err := h.matchService.GetPredictionStats(matchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить статистику прогнозов"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"match":            match,
		"prediction_stats": stats,
	})
}

func (h *Handler) getPredictionStats(c *gin.Context) {
	matchID, ok := pathInt64(c, "id")
	if !ok {
		return
	}

	stats, err := h.matchService.GetPredictionStats(matchID)
	if err != nil {
		if err.Error() == "match not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Матч не найден"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить статистику прогнозов"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) submitPrediction(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	var input struct {
		MatchID    int64  `json:"match_id" binding:"required"`
		UserChoice string `json:"user_choice" binding:"required,oneof=1 X 2"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса или невалидный выбор исхода"})
		return
	}

	if err := h.predictService.MakePrediction(user.TgID, input.MatchID, input.UserChoice); err != nil {
		status := http.StatusInternalServerError

		switch err.Error() {
		case "match not found", "invalid choice: must be '1', 'X' or '2'":
			status = http.StatusBadRequest
		case "predictions are locked: match has already started":
			status = http.StatusConflict
		}

		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Прогноз успешно сохранен",
	})
}

func (h *Handler) getMyPredictions(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	status := normalizeStatus(c.Query("status"))

	items, err := h.matchService.GetPredictionHistory(user.TgID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить прогнозы"})
		return
	}

	c.JSON(http.StatusOK, items)
}

func pathInt64(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор"})
		return 0, false
	}

	return value, true
}

func queryInt(c *gin.Context, name string, fallback int) int {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func normalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))

	switch status {
	case "", "active":
		return "active"
	case "upcoming", "scheduled":
		return "scheduled"
	case "live":
		return "live"
	case "finished":
		return "finished"
	case "all":
		return "all"
	default:
		return "active"
	}
}
