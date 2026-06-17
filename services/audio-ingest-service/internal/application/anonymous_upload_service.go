package application

import (
	"context"
	"fmt"
	"strings"

	"audio-ingest-service/internal/domain"

	"github.com/google/uuid"
)

type AnonymousUploadService struct {
	storage   Storage
	publisher EventPublisher
	repo      domain.AnonymousSessionRepository
}

func NewAnonymousUploadService(
	storage Storage,
	publisher EventPublisher,
	repo domain.AnonymousSessionRepository,
) *AnonymousUploadService {
	return &AnonymousUploadService{
		storage:   storage,
		publisher: publisher,
		repo:      repo,
	}
}

func (s *AnonymousUploadService) Upload(ctx context.Context, in AnonymousUploadInput) (string, string, error) {
	if in.File == nil {
		return "", "", invalidInput("audio file is required")
	}
	if strings.TrimSpace(in.UserID) == "" {
		return "", "", invalidInput("user_id is required")
	}

	audioID := uuid.NewString()
	objectKey := fmt.Sprintf("anonymous/%s.mp3", audioID)

	if err := s.storage.Put(ctx, objectKey, in.File); err != nil {
		return "", "", fmt.Errorf("store audio: %w", err)
	}

	session := &domain.AnonymousSession{
		AudioID:     audioID,
		OwnerUserID: in.UserID,
	}
	if err := s.repo.Create(ctx, session); err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return "", "", fmt.Errorf("create session: %w", err)
	}

	event := UploadedEvent{
		AudioID:        audioID,
		WorkspaceID:    "",
		UploaderUserID: in.UserID,
		SessionID:      session.SessionID,
	}
	if err := s.publisher.PublishAnonymousAudioUploaded(ctx, event); err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return "", "", fmt.Errorf("publish audio.uploaded: %w", err)
	}

	return audioID, session.SessionID, nil
}
