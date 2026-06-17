package handler

import (
	"io"
	"log/slog"
	"net/http"

	"audio-ingest-service/internal/application"
	"audio-ingest-service/internal/domain"

	"github.com/gin-gonic/gin"
)

type AnonymousFragmentHandler struct {
	repo    domain.AnonymousSessionRepository
	storage application.Storage
	log     *slog.Logger
}

func NewAnonymousFragmentHandler(repo domain.AnonymousSessionRepository, storage application.Storage, log *slog.Logger) *AnonymousFragmentHandler {
	return &AnonymousFragmentHandler{repo: repo, storage: storage, log: log}
}

func (h *AnonymousFragmentHandler) GetFragment(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
		return
	}

	sessionID := c.Param("session_id")
	speakerID := c.Param("speaker_id")

	session, err := h.repo.GetByID(c.Request.Context(), sessionID)
	if err != nil || session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	if session.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	speaker, err := h.repo.GetSpeakerByID(c.Request.Context(), speakerID)
	if err != nil || speaker == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "speaker not found"})
		return
	}
	if speaker.SessionID != sessionID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	reader, err := h.storage.Get(c.Request.Context(), speaker.FragmentKey)
	if err != nil {
		h.log.Error("failed to get fragment", "fragment_key", speaker.FragmentKey, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get fragment"})
		return
	}
	defer func() { _ = reader.Close() }()

	c.Header("Content-Type", "audio/wav")
	c.Header("Content-Disposition", "inline; filename=\""+speaker.Label+".wav\"")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, reader)
}
