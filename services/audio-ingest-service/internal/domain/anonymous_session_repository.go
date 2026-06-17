package domain

import "context"

type AnonymousSessionRepository interface {
	Create(ctx context.Context, session *AnonymousSession) error
	GetByID(ctx context.Context, sessionID string) (*AnonymousSession, error)
	GetByAudioID(ctx context.Context, audioID string) (*AnonymousSession, error)
	UpdateStatus(ctx context.Context, sessionID string, status SessionStatus) error

	AddSpeaker(ctx context.Context, speaker *AnonymousSpeaker) error
	GetSpeakers(ctx context.Context, sessionID string) ([]AnonymousSpeaker, error)
	GetSpeakerByID(ctx context.Context, speakerID string) (*AnonymousSpeaker, error)

	// CleanupExpired marks expired sessions and returns their audio/fragment keys for MinIO cleanup.
	CleanupExpired(ctx context.Context) ([]string, error)
}
