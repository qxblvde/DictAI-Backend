package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainerrors "audio-ingest-service/internal/domain/errors"

	"github.com/google/uuid"
)

type UploadService struct {
	storage       Storage
	publisher     EventPublisher
	accessChecker WorkspaceAccessChecker
}

func NewUploadService(storage Storage, publisher EventPublisher, accessChecker WorkspaceAccessChecker) *UploadService {
	return &UploadService{
		storage:       storage,
		publisher:     publisher,
		accessChecker: accessChecker,
	}
}

func (s *UploadService) Upload(ctx context.Context, in UploadInput) (string, error) {
	if s.storage == nil || s.publisher == nil || s.accessChecker == nil {
		return "", errors.New("upload service is not configured")
	}

	if in.File == nil {
		return "", invalidInput("audio file is required")
	}
	if _, err := uuid.Parse(in.WorkspaceID); err != nil {
		return "", invalidInput("workspace_id must be a valid UUID")
	}
	if strings.TrimSpace(in.UserID) == "" {
		return "", invalidInput("user_id is required")
	}

	allowed, err := s.accessChecker.CanUpload(ctx, in.WorkspaceID, in.UserID)
	if err != nil {
		return "", fmt.Errorf("workspace access check: %w", err)
	}
	if !allowed {
		return "", domainerrors.ErrForbidden
	}

	audioID := uuid.NewString()
	objectKey := buildObjectKey(in.WorkspaceID, audioID)

	if err := s.storage.Put(ctx, objectKey, in.File); err != nil {
		return "", fmt.Errorf("store audio: %w", err)
	}

	event := UploadedEvent{
		AudioID:        audioID,
		WorkspaceID:    in.WorkspaceID,
		UploaderUserID: in.UserID,
	}
	if err := s.publisher.PublishAudioUploaded(ctx, event); err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return "", fmt.Errorf("publish audio.uploaded: %w", err)
	}

	return audioID, nil
}

func buildObjectKey(workspaceID, audioID string) string {
	return fmt.Sprintf("%s/%s.mp3", workspaceID, audioID)
}

func invalidInput(message string) error {
	return fmt.Errorf("%w: %s", domainerrors.ErrInvalidInput, message)
}
