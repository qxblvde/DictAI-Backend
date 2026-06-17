package handlers

import (
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

type CreateWorkspaceRequest struct {
	Name string `json:"name"`
}

type CreateWorkspaceResponse struct {
	WorkspaceID string `json:"workspace_id"`
}

func NewCreateWorkspaceHandler(svc *service.WorkspaceService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.GetHeader("X-User-Id")
		if userID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "user unauthorized",
			})
			return
		}

		var req CreateWorkspaceRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		if req.Name == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		workspace, err := svc.CreateWorkspace(userID, req.Name)
		if err != nil {
			handleError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"workspace_id":  workspace.WorkspaceID,
			"owner_user_id": workspace.OwnerID,
			"name":          workspace.Name,
			"created_at":    workspace.CreatedAt,
		})
	}
}
