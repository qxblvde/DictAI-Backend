package handlers

import (
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

func NewDeleteWorkspaceHandler(svc *service.WorkspaceService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.GetHeader("X-User-Id")
		if userID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
			return
		}

		workspaceID := ctx.Param("workspace_id")
		if workspaceID == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
			return
		}

		if err := svc.DeleteWorkspace(userID, workspaceID); err != nil {
			handleError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "workspace deleted successfully"})
	}
}
