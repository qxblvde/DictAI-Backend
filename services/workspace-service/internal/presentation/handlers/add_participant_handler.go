package handlers

import (
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

type AddParticipantRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AddParticipantResponse struct {
	ParticipantID string `json:"participant_id"`
}

func NewAddParticipantHandler(svc *service.WorkspaceService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.GetHeader("X-User-Id")
		if userID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "user unauthorized",
			})
			return
		}

		workspaceID := ctx.Param("workspace_id")
		if workspaceID == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "workspace_id missing",
			})
			return
		}

		var req AddParticipantRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		if req.Name == "" || req.Email == "" {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		participant, err := svc.AddParticipant(userID, workspaceID, req.Name, req.Email)
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
