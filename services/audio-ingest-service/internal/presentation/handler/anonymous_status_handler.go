package handler

import (
	"log/slog"
	"net/http"

	"audio-ingest-service/internal/domain"

	"github.com/gin-gonic/gin"
)

type AnonymousStatusHandler struct {
	repo domain.AnonymousSessionRepository
	log  *slog.Logger
}

func NewAnonymousStatusHandler(repo domain.AnonymousSessionRepository, log *slog.Logger) *AnonymousStatusHandler {
	return &AnonymousStatusHandler{repo: repo, log: log}
}

type speakerResponse struct {
	SpeakerID   string        `json:"speaker_id"`
	Label       string        `json:"label"`
	FragmentURL string        `json:"fragment_url"`
	Segments    []segmentResp `json:"segments"`
}

type segmentResp struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

func (h *AnonymousStatusHandler) GetStatus(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
		return
	}

	sessionID := c.Param("session_id")
	session, err := h.repo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		h.log.Error("failed to get session", "session_id", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get session"})
		return
	}
	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	if session.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if session.Status != domain.SessionStatusReady {
		c.JSON(http.StatusOK, gin.H{
			"session_id": session.SessionID,
			"audio_id":   session.AudioID,
			"status":     string(session.Status),
			"speakers":   []speakerResponse{},
		})
		return
	}

	dbSpeakers, err := h.repo.GetSpeakers(c.Request.Context(), sessionID)
	if err != nil {
		h.log.Error("failed to get speakers", "session_id", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get speakers"})
		return
	}

	speakers := make([]speakerResponse, len(dbSpeakers))
	for i, sp := range dbSpeakers {
		segs := make([]segmentResp, len(sp.Segments))
		for j, s := range sp.Segments {
			segs[j] = segmentResp{Start: s.Start, End: s.End, Text: s.Text}
		}
		speakers[i] = speakerResponse{
			SpeakerID:   sp.SpeakerID,
			Label:       sp.Label,
			FragmentURL: "/ingest/audio/anonymous/" + sessionID + "/speakers/" + sp.SpeakerID + "/fragment",
			Segments:    segs,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": session.SessionID,
		"audio_id":   session.AudioID,
		"status":     string(session.Status),
		"speakers":   speakers,
	})
}
