package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	voiceprofile "github.com/Microservices/services/voice-profile-service/internal/application/contracts/voiceprofile"
	appservice "github.com/Microservices/services/voice-profile-service/internal/application/service"
	"github.com/gin-gonic/gin"
)

func NewListProfilesHandler(svc voiceprofile.VoiceProfileService, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		userId := c.GetHeader("X-User-Id")
		if userId == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
			return
		}

		profiles, err := svc.ListVoiceProfiles(userId)
		if err != nil {
			log.Error("failed to list voice profiles", "request_id", reqID, "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to list voice profiles"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": profiles})
	}
}

func NewUploadAudioHandler(svc voiceprofile.VoiceProfileService, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		l := log.With("request_id", reqID)

		userId := c.GetHeader("X-User-Id")
		if userId == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
			return
		}

		workspaceId := c.PostForm("workspace_id")
		if workspaceId == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
			return
		}

		participantId := c.PostForm("participant_id")
		if participantId == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "participant_id is required"})
			return
		}

		audioHeader, err := c.FormFile("audio")
		if err != nil {
			audioHeader, err = c.FormFile("file")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
				return
			}
		}

		l = l.With("user_id", userId, "workspace_id", workspaceId, "participant_id", participantId, "filename", audioHeader.Filename)
		l.Debug("processing voice profile upload")

		audio, err := audioHeader.Open()
		if err != nil {
			l.Error("failed to open uploaded file", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "cannot open uploaded file"})
			return
		}
		defer func() {
			if closeErr := audio.Close(); closeErr != nil {
				l.Warn("failed to close uploaded file", "error", closeErr)
			}
		}()

		voiceProfileId, err := svc.CreateVoiceProfile(audio, userId, workspaceId, participantId, c.PostForm("display_name"))
		if err != nil {
			handleVoiceProfileError(c, l, err)
			return
		}

		embedding, err := svc.GetVoiceProfile(participantId)
		if err != nil {
			l.Error("failed to load profile embedding after creation", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to load profile embedding"})
			return
		}

		l.Info("voice profile created", "voice_profile_id", voiceProfileId)
		c.JSON(http.StatusOK, gin.H{
			"voice_profile_id": voiceProfileId,
			"participant_id":   participantId,
			"embedding":        embedding,
		})
	}
}

func NewCreateLibraryProfileHandler(svc voiceprofile.VoiceProfileService, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		l := log.With("request_id", reqID)

		userId := c.GetHeader("X-User-Id")
		if userId == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
			return
		}

		audioHeader, err := c.FormFile("audio")
		if err != nil {
			audioHeader, err = c.FormFile("file")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
				return
			}
		}

		audio, err := audioHeader.Open()
		if err != nil {
			l.Error("failed to open uploaded file", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "cannot open uploaded file"})
			return
		}
		defer func() { _ = audio.Close() }()

		voiceProfileId, err := svc.CreateLibraryProfile(audio, userId, c.PostForm("display_name"))
		if err != nil {
			handleVoiceProfileError(c, l, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"voice_profile_id": voiceProfileId})
	}
}

func NewAssignProfileHandler(svc voiceprofile.VoiceProfileService, log *slog.Logger) gin.HandlerFunc {
	type request struct {
		WorkspaceID   string `json:"workspace_id"`
		ParticipantID string `json:"participant_id"`
	}

	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		l := log.With("request_id", reqID)

		userId := c.GetHeader("X-User-Id")
		if userId == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
			return
		}

		var req request
		if err := c.ShouldBindJSON(&req); err != nil || req.WorkspaceID == "" || req.ParticipantID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "workspace_id and participant_id are required"})
			return
		}

		voiceProfileId := c.Param("voice_profile_id")
		if voiceProfileId == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "voice_profile_id is required"})
			return
		}

		if err := svc.AssignVoiceProfile(userId, req.WorkspaceID, req.ParticipantID, voiceProfileId); err != nil {
			handleVoiceProfileError(c, l, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func NewGetProfileHandler(svc voiceprofile.VoiceProfileService, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		participantId := c.Param("participant_id")
		if participantId == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "participant_id is required"})
			return
		}

		embedding, err := svc.GetVoiceProfile(participantId)
		if err != nil {
			log.Warn("voice profile not found", "participant_id", participantId, "request_id", reqID, "error", err)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "voice profile not found"})
			return
		}

		log.Debug("voice profile retrieved", "participant_id", participantId, "request_id", reqID)
		c.JSON(http.StatusOK, gin.H{
			"participant_id": participantId,
			"embedding":      embedding,
		})
	}
}

func handleVoiceProfileError(c *gin.Context, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, appservice.ErrParticipantNotFound), errors.Is(err, appservice.ErrProfileNotFound):
		log.Warn("voice profile entity not found", "error", err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, appservice.ErrForbidden):
		log.Warn("voice profile access denied", "error", err)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		log.Error("voice profile operation failed", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to process voice profile"})
	}
}
