package handlers

import (
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

type GetParticipantsResponse struct {
	ParticipantID  string  `json:"participant_id"`
	WorkspaceID    string  `json:"workspace_id"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	VoiceProfileID *string `json:"voice_profile_id,omitempty"`
}

func NewGetParticipantHandler(svc *service.WorkspaceService) gin.HandlerFunc {
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

		participants, err := svc.GetParticipants(userID, workspaceID)
		if err != nil {
			handleError(ctx, err)
			return
		}

		var response []GetParticipantsResponse
		for _, participant := range participants {
			response = append(response, GetParticipantsResponse{
				ParticipantID:  participant.ParticipantID,
				WorkspaceID:    participant.WorkspaceID,
				Name:           participant.Name,
				Email:          participant.Email,
				VoiceProfileID: participant.VoiceProfileID,
			})
		}

		ctx.JSON(http.StatusOK, response)
	}
}
