package presentation

import (
	"net/http"

	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
)

type ChangePasswordRequest struct {
	UserId      string
	NewPassword string `json:"new_password"`
}

func NewChangePasswordHandler(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ChangePasswordRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		req.UserId = c.GetHeader("X-User-Id")
		if req.UserId == "" || req.NewPassword == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
		}

		err := svc.ChangePassword(req.UserId, req.NewPassword)
		if err != nil {
			handleError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "password changed successfully",
		})
	}
}
