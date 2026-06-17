package handler

import (
	"log/slog"
	"net/http"

	"audio-ingest-service/internal/application"

	"github.com/gin-gonic/gin"
)

type AnonymousConfirmHandler struct {
	service *application.ConfirmService
	log     *slog.Logger
}

func NewAnonymousConfirmHandler(service *application.ConfirmService, log *slog.Logger) *AnonymousConfirmHandler {
	return &AnonymousConfirmHandler{service: service, log: log}
}

type confirmRequest struct {
	WorkspaceName string              `json:"workspace_name"`
	Speakers      []speakerAssignReq  `json:"speakers"`
}

type speakerAssignReq struct {
	Label string `json:"label"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *AnonymousConfirmHandler) Confirm(c *gin.Context) {
	reqID := c.GetHeader("X-Request-ID")
	l := h.log.With("request_id", reqID)

	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
		return
	}

	sessionID := c.Param("session_id")

	var req confirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if len(req.Speakers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "speakers list is required"})
		return
	}
	for _, sp := range req.Speakers {
		if sp.Label == "" || sp.Name == "" || sp.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each speaker must have label, name, and email"})
			return
		}
	}

	assignments := make([]application.SpeakerAssignment, len(req.Speakers))
	for i, sp := range req.Speakers {
		assignments[i] = application.SpeakerAssignment{
			Label: sp.Label,
			Name:  sp.Name,
			Email: sp.Email,
		}
	}

	out, err := h.service.Confirm(c.Request.Context(), application.ConfirmInput{
		SessionID:     sessionID,
		UserID:        userID,
		WorkspaceName: req.WorkspaceName,
		Speakers:      assignments,
	})
	if err != nil {
		switch err.Error() {
		case "session not found":
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		case "forbidden":
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			l.Error("confirm failed", "session_id", sessionID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm session"})
		}
		return
	}

	l.Info("anonymous session confirmed", "session_id", sessionID, "workspace_id", out.WorkspaceID)
	c.JSON(http.StatusOK, gin.H{
		"workspace_id":   out.WorkspaceID,
		"result_pending": true,
	})
}
