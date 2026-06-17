package handlers

import (
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

func NewDeleteParticipantHandler(svc *service.WorkspaceService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.GetHeader("X-User-Id")
		if userID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
			return
		}

		workspaceID := ctx.Param("workspace_id")
		participantID := ctx.Param("participant_id")
		if workspaceID == "" || participantID == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "workspace_id and participant_id are required"})
			return
		}

		if err := svc.DeleteParticipant(userID, workspaceID, participantID); err != nil {
			handleError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "participant removed successfully"})
	}
}
