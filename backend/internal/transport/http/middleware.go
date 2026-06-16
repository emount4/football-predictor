package http

import (
	"net/http"

	"football-predictor/internal/domain"
	"football-predictor/internal/service"

	"github.com/gin-gonic/gin"
)

const authenticatedUserKey = "authenticated_user"

// TelegramAuthMiddleware проверяет подпись WebApp InitData и кладет пользователя в контекст.
func TelegramAuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		initData := c.GetHeader("Authorization")
		if initData == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Отсутствует заголовок Authorization"})
			return
		}

		user, err := authService.ValidateAndGetUser(initData)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверные данные Telegram"})
			return
		}

		c.Set(authenticatedUserKey, user)
		c.Next()
	}
}

func currentUserFromContext(c *gin.Context) (*domain.User, bool) {
	userValue, exists := c.Get(authenticatedUserKey)
	if !exists {
		return nil, false
	}

	user, ok := userValue.(*domain.User)
	return user, ok
}
