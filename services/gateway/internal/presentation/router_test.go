package presentation

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"gateway/internal/config"
	"gateway/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func createTestToken(secret, userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
	})
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func TestRouterJWTMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:    "test-secret",
		RateLimitRPS: 1000,
		RateLimit:    1000,
	}

	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimit))
	r.Use(middleware.JWTMiddleware(cfg.JWTSecret, slog.Default()))

	r.GET("/test-user", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists)
		assert.Equal(t, "123", userID)
		c.Status(http.StatusOK)
	})

	token := createTestToken(cfg.JWTSecret, "123")

	req := httptest.NewRequest(http.MethodGet, "/test-user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
