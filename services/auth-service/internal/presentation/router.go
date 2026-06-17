package presentation

import (
	"log/slog"

	"github.com/Microservices/services/auth-service/internal/presentation/handlers"
	"github.com/Microservices/services/auth-service/internal/presentation/middleware"
	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(svc *service.AuthService, log *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(middleware.Recovery(log))
	router.Use(cors.Default())
	router.Use(middleware.Logging(log))

	auth := router.Group("/auth")

	auth.POST("/register", presentation.NewRegisterHandler(svc))
	auth.POST("/login", presentation.NewLoginHandler(svc))
	auth.POST("/refresh", presentation.NewRefreshHandler(svc))
	auth.POST("/change-password", presentation.NewChangePasswordHandler(svc))
	auth.POST("/change-email", presentation.NewChangeEmailHandler(svc))
	auth.GET("/users/:id", presentation.NewGetUserHandler(svc))

	return router
}
