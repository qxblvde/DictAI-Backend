package handlers

import (
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

type UpdateParticipantRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewUpdateParticipantHandler(svc *service.WorkspaceService) gin.HandlerFunc {
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

		var req UpdateParticipantRequest
		if err := ctx.ShouldBindJSON(&req); err != nil || (req.Name == "" && req.Email == "") {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "at least one of name or email must be provided"})
			return
		}

		participant, err := svc.UpdateParticipant(userID, workspaceID, participantID, req.Name, req.Email)
		if err != nil {
			handleError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"participant_id":   participant.ParticipantID,
			"workspace_id":     participant.WorkspaceID,
			"name":             participant.Name,
			"email":            participant.Email,
			"voice_profile_id": participant.VoiceProfileID,
		})
	}
}
