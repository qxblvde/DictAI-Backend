package presentation

import (
	"net/http"

	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
)

type ChangeEmailRequest struct {
	UserId   string
	NewEmail string `json:"new_email"`
}

func NewChangeEmailHandler(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ChangeEmailRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		req.UserId = c.GetHeader("X-User-Id")
		if req.UserId == "" || req.NewEmail == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
		}

		err := svc.ChangeEmail(req.UserId, req.NewEmail)
		if err != nil {
			handleError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "email changed successfully",
		})
	}
}
