package handlers

import (
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

type UpdateWorkspaceRequest struct {
	Name string `json:"name"`
}

func NewUpdateWorkspaceHandler(svc *service.WorkspaceService) gin.HandlerFunc {
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

		var req UpdateWorkspaceRequest
		if err := ctx.ShouldBindJSON(&req); err != nil || req.Name == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		workspace, err := svc.UpdateWorkspace(userID, workspaceID, req.Name)
		if err != nil {
			handleError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"workspace_id": workspace.WorkspaceID,
			"owner_id":     workspace.OwnerID,
			"name":         workspace.Name,
			"created_at":   workspace.CreatedAt,
		})
	}
}
