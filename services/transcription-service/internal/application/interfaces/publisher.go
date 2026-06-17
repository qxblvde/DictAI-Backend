package interfaces

type Publisher interface {
	Publish(audioId, workSpaceId, uploadUserId, transcription, sessionID string) error
}
