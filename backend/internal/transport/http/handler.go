package http

import (
	"football-predictor/internal/domain"
	"football-predictor/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService    *service.AuthService
	matchService   *service.MatchService
	predictService domain.PredictService
}

func NewHandler(authService *service.AuthService, ms *service.MatchService, predictService domain.PredictService) *Handler {
	return &Handler{
		authService:    authService,
		matchService:   ms,
		predictService: predictService,
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	r := gin.Default()

	// CORS мидлварь
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	protected := api.Group("", h.telegramAuthMiddlewareWrapper())
	protected.POST("/auth", h.handleAuth)
	protected.GET("/matches", h.getMatches)
	protected.POST("/predictions", h.submitPrediction)

	return r
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

func (h *Handler) getMatches(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	matches, err := h.matchService.GetMatchesForUser(user.TgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить матчи"})
		return
	}
	c.JSON(http.StatusOK, matches)
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

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Прогноз успешно сохранен!"})
}
