package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"audio-ingest-service/internal/application"

	"github.com/gin-gonic/gin"
)

type AnonymousUploadHandler struct {
	service        *application.AnonymousUploadService
	log            *slog.Logger
	resultsBaseURL string
}

func NewAnonymousUploadHandler(service *application.AnonymousUploadService, log *slog.Logger, resultsBaseURL string) *AnonymousUploadHandler {
	return &AnonymousUploadHandler{service: service, log: log, resultsBaseURL: resultsBaseURL}
}

func (h *AnonymousUploadHandler) Upload(c *gin.Context) {
	reqID := c.GetHeader("X-Request-ID")
	l := h.log.With("request_id", reqID)

	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
		return
	}

	audioHeader, err := c.FormFile("audio")
	if err != nil {
		audioHeader, err = c.FormFile("file")
		if err != nil {
			l.Warn("audio file missing in anonymous upload request")
			c.JSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
			return
		}
	}

	file, err := audioHeader.Open()
	if err != nil {
		l.Error("failed to open uploaded file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer func() { _ = file.Close() }()

	audioID, sessionID, err := h.service.Upload(c.Request.Context(), application.AnonymousUploadInput{
		UserID:   userID,
		Filename: audioHeader.Filename,
		File:     file,
	})
	if err != nil {
		l.Error("failed to upload anonymous audio", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload audio"})
		return
	}

	l.Info("anonymous audio uploaded", "audio_id", audioID, "session_id", sessionID)

	if h.resultsBaseURL != "" {
		go func() {
			body, _ := json.Marshal(map[string]string{
				"audio_id":       audioID,
				"workspace_id":   "",
				"upload_user_id": userID,
			})
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
				h.resultsBaseURL+"/meetings/results/pending", bytes.NewReader(body))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					_ = resp.Body.Close()
				}
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"audio_id":   audioID,
		"status":     "processing",
	})
}
