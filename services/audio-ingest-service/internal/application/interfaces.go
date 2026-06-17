package application

import (
	"context"
	"io"
)

type Storage interface {
	Put(ctx context.Context, objectKey string, body io.Reader) error
	Get(ctx context.Context, objectKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectKey string) error
}

type EventPublisher interface {
	PublishAudioUploaded(ctx context.Context, event UploadedEvent) error
	PublishAnonymousAudioUploaded(ctx context.Context, event UploadedEvent) error
}

type WorkspaceAccessChecker interface {
	CanUpload(ctx context.Context, workspaceID, userID string) (bool, error)
}

type WorkspaceClient interface {
	CreateWorkspace(ctx context.Context, userID, name string) (string, error)
	AddParticipant(ctx context.Context, userID, workspaceID, name, email string) (string, error)
}

type VoiceProfileClient interface {
	CreateLibraryProfile(ctx context.Context, userID, displayName string, audio io.Reader) (string, error)
	AssignProfile(ctx context.Context, userID, workspaceID, participantID, voiceProfileID string) error
}
