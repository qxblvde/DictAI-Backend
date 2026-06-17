package presentation

import (
	"net/http"

	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
)

func NewGetUserHandler(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		user, err := svc.GetUserByID(userID)
		if err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": user.ID, "email": user.Email})
	}
}
