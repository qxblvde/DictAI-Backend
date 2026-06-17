package interfaces

import "context"

type TranscriptBuilderService interface {
	Process(ctx context.Context, audioId, workspaceId, uploadUserId, transcription string) error
	ProcessAnonymous(ctx context.Context, audioId, uploadUserId, sessionID, transcription string) error
}
