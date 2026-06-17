package interfaces

import "context"

type TranscriptionService interface {
	Transcript(ctx context.Context, audioId, workspaceId, uploadUserId, sessionID string) error
}
