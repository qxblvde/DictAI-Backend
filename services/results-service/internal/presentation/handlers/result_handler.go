package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Microservices/services/results-service/internal/service"
)

type ResultHandler struct {
	svc *service.ResultService
}

func NewResultHandler(svc *service.ResultService) *ResultHandler {
	return &ResultHandler{svc: svc}
}

func (h *ResultHandler) List(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-Id header is required"})
		return
	}

	var filters service.ListFilters

	if v := c.Query("start_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, use RFC3339 or YYYY-MM-DD"})
				return
			}
		}
		filters.StartDate = &t
	}

	if v := c.Query("end_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, use RFC3339 or YYYY-MM-DD"})
				return
			}
			t = t.Add(24*time.Hour - time.Second)
		}
		filters.EndDate = &t
	}

	if v := c.Query("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page must be a positive integer"})
			return
		}
		filters.Page = p
	}

	if v := c.Query("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err != nil || l < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return
		}
		filters.Limit = l
	}

	out, err := h.svc.List(c.Request.Context(), userID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": out.Results,
		"pagination": gin.H{
			"page":        out.Page,
			"limit":       out.Limit,
			"total":       out.Total,
			"total_pages": out.TotalPages,
		},
	})
}

func (h *ResultHandler) CreatePending(c *gin.Context) {
	var body struct {
		AudioID      string `json:"audio_id"`
		WorkspaceID  string `json:"workspace_id"`
		UploadUserID string `json:"upload_user_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.AudioID == "" || body.UploadUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_id and upload_user_id are required"})
		return
	}
	if err := h.svc.CreatePending(c.Request.Context(), body.AudioID, body.WorkspaceID, body.UploadUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *ResultHandler) SetStatus(c *gin.Context) {
	audioID := c.Param("audio_id")
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || (body.Status != "done" && body.Status != "failed") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'done' or 'failed'"})
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), audioID, body.Status); err != nil {
		if errors.Is(err, service.ErrResultNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "result not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ResultHandler) Upload(c *gin.Context) {
	audioID := c.PostForm("audio_id")
	workspaceID := c.PostForm("workspace_id")
	uploadUserID := c.PostForm("upload_user_id")
	if audioID == "" || workspaceID == "" || uploadUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_id, workspace_id, upload_user_id are required"})
		return
	}

	summaryFile, _, err := c.Request.FormFile("summary")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "summary file is required"})
		return
	}
	defer func() { _ = summaryFile.Close() }()
	summaryBytes, err := io.ReadAll(summaryFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read summary file"})
		return
	}

	transcriptFile, _, err := c.Request.FormFile("transcript")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "transcript file is required"})
		return
	}
	defer func() { _ = transcriptFile.Close() }()
	transcriptBytes, err := io.ReadAll(transcriptFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read transcript file"})
		return
	}

	out, err := h.svc.Upload(c.Request.Context(), service.UploadInput{
		AudioID:       audioID,
		WorkspaceID:   workspaceID,
		UploadUserID:  uploadUserID,
		SummaryPDF:    summaryBytes,
		TranscriptPDF: transcriptBytes,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"result_id":      out.ResultID,
		"summary_url":    out.SummaryURL,
		"transcript_url": out.TranscriptURL,
	})
}

func (h *ResultHandler) Delete(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-Id header is required"})
		return
	}

	audioID := c.Param("audio_id")
	if audioID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_id is required"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, audioID); err != nil {
		if errors.Is(err, service.ErrResultNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "result not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ResultHandler) GetSummary(c *gin.Context) {
	h.serveFile(c, "summary")
}

func (h *ResultHandler) GetTranscript(c *gin.Context) {
	h.serveFile(c, "transcript")
}

func (h *ResultHandler) serveFile(c *gin.Context, fileType string) {
	audioID := c.Param("audio_id")
	data, err := h.svc.GetFile(c.Request.Context(), audioID, fileType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/pdf", data)
}

func (h *ResultHandler) GetAudio(c *gin.Context) {
	audioID := c.Param("audio_id")
	data, contentType, err := h.svc.GetAudio(c.Request.Context(), audioID)
	if err != nil {
		slog.Error("get audio failed", "audio_id", audioID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+audioID+".mp3\"")
	c.Data(http.StatusOK, contentType, data)
}
