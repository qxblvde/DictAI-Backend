package presentation

import (
	"net/http"

	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func NewLoginHandler(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		token, err := svc.LoginUser(req.Email, req.Password)
		if err != nil {
			handleError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	}
}
