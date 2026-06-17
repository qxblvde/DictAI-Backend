package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(secret string, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("missing authorization header", "request_id", reqID, "path", c.Request.URL.Path)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))

		if err != nil || !token.Valid {
			log.Warn("invalid or expired token", "request_id", reqID, "path", c.Request.URL.Path, "error", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Warn("failed to parse token claims", "request_id", reqID)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"]
		if !ok {
			log.Warn("user_id not found in token claims", "request_id", reqID)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		log.Debug("jwt validated", "request_id", reqID, "user_id", userID)
		c.Set("user_id", userID)
		c.Next()
	}
}
