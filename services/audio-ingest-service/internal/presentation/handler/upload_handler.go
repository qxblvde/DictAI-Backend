package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"audio-ingest-service/internal/application"
	domainerrors "audio-ingest-service/internal/domain/errors"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	service        *application.UploadService
	log            *slog.Logger
	resultsBaseURL string
}

func NewUploadHandler(service *application.UploadService, log *slog.Logger, resultsBaseURL string) *UploadHandler {
	return &UploadHandler{service: service, log: log, resultsBaseURL: resultsBaseURL}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	reqID := c.GetHeader("X-Request-ID")
	l := h.log.With("request_id", reqID)

	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
		return
	}

	workspaceID := c.PostForm("workspace_id")
	l = l.With("user_id", userID, "workspace_id", workspaceID)

	audioHeader, err := c.FormFile("audio")
	if err != nil {
		audioHeader, err = c.FormFile("file")
		if err != nil {
			l.Warn("audio file missing in upload request")
			c.JSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
			return
		}
	}

	l = l.With("filename", audioHeader.Filename, "size_bytes", audioHeader.Size)
	l.Debug("processing audio upload")

	file, err := audioHeader.Open()
	if err != nil {
		l.Error("failed to open uploaded file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			l.Warn("failed to close uploaded file", "error", closeErr)
		}
	}()

	audioID, err := h.service.Upload(c.Request.Context(), application.UploadInput{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Filename:    audioHeader.Filename,
		File:        file,
	})
	if err != nil {
		switch {
		case errors.Is(err, domainerrors.ErrInvalidInput):
			l.Warn("invalid upload input", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, domainerrors.ErrForbidden):
			l.Warn("upload forbidden", "error", err)
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			l.Error("failed to upload audio", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload audio"})
		}
		return
	}

	l.Info("audio uploaded successfully", "audio_id", audioID)

	if h.resultsBaseURL != "" {
		go h.createPendingResult(context.Background(), audioID, workspaceID, userID, l)
	}

	c.JSON(http.StatusOK, gin.H{"audio_id": audioID})
}

func (h *UploadHandler) createPendingResult(ctx context.Context, audioID, workspaceID, userID string, l *slog.Logger) {
	body, _ := json.Marshal(map[string]string{
		"audio_id":       audioID,
		"workspace_id":   workspaceID,
		"upload_user_id": userID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.resultsBaseURL+"/meetings/results/pending", bytes.NewReader(body))
	if err != nil {
		l.Warn("failed to create pending result request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.Warn("failed to create pending result", "error", err)
		return
	}
	_ = resp.Body.Close()
	l.Debug("pending result created", "audio_id", audioID, "status", resp.StatusCode)
}
