package presentation

import (
	"log/slog"

	"github.com/DictAI/Microservices/services/workspace-service/internal/presentation/handlers"
	"github.com/DictAI/Microservices/services/workspace-service/internal/presentation/middleware"
	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(svc *service.WorkspaceService, log *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(middleware.Recovery(log))
	router.Use(cors.Default())
	router.Use(middleware.Logging(log))

	workspaces := router.Group("/workspaces")

	workspaces.GET("/", handlers.NewGetUserWorkspacesHandler(svc))
	workspaces.POST("/", handlers.NewCreateWorkspaceHandler(svc))
	workspaces.GET("/:workspace_id", handlers.NewGetWorkspaceHandler(svc))
	workspaces.PUT("/:workspace_id", handlers.NewUpdateWorkspaceHandler(svc))
	workspaces.DELETE("/:workspace_id", handlers.NewDeleteWorkspaceHandler(svc))
	workspaces.POST("/:workspace_id/participants", handlers.NewAddParticipantHandler(svc))
	workspaces.GET("/:workspace_id/participants", handlers.NewGetParticipantHandler(svc))
	workspaces.PUT("/:workspace_id/participants/:participant_id", handlers.NewUpdateParticipantHandler(svc))
	workspaces.DELETE("/:workspace_id/participants/:participant_id", handlers.NewDeleteParticipantHandler(svc))

	return router
}
