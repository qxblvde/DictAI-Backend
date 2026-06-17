package presentation

import (
	"net/http"

	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
)

type RefreshRequest struct {
	UserId string
}

type RefreshResponse struct {
	Token string `json:"token"`
}

func NewRefreshHandler(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		req.UserId = c.GetHeader("X-User-Id")
		if req.UserId == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
		}

		token, err := svc.RefreshToken(req.UserId)
		if err != nil {
			handleError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	}
}
