package presentation

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
)

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidPassword):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid password"})
	case errors.Is(err, service.ErrWrongPassword):
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
	case errors.Is(err, service.ErrInvalidEmail):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
	case errors.Is(err, service.ErrEmailAlreadyExists):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "email already exists"})
	case errors.Is(err, service.ErrUserNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
	default:
		slog.Error("unhandled error in auth handler",
			"error", err,
			"path", c.Request.URL.Path,
			"request_id", c.GetHeader("X-Request-ID"),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
