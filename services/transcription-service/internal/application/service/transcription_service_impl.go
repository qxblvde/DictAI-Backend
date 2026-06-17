package service

import (
	"context"
	"errors"

	"github.com/Microservices/services/transcription-service/internal/application/interfaces"
)

type transcriptionService struct {
	transcriber  interfaces.Transcriber
	publisher    interfaces.Publisher
	audioService interfaces.AudioService
}

func NewTranscriptionService(transcriber interfaces.Transcriber,
	publisher interfaces.Publisher,
	audioService interfaces.AudioService) interfaces.TranscriptionService {
	return &transcriptionService{
		transcriber:  transcriber,
		publisher:    publisher,
		audioService: audioService,
	}
}

func (t *transcriptionService) Transcript(ctx context.Context, audioId, workspaceId, uploadUserId, sessionID string) error {
	if t.audioService == nil {
		return errors.New("audio service is not configured")
	}
	if t.transcriber == nil {
		return errors.New("transcriber is not configured")
	}
	if t.publisher == nil {
		return errors.New("publisher is not configured")
	}

	if audioId == "" || uploadUserId == "" {
		return errors.New("audioId and uploadUserId are required")
	}

	audio, err := t.audioService.GetAudio(audioId, workspaceId, uploadUserId)
	if err != nil {
		return err
	}
	defer func() {
		_ = audio.Close()
	}()

	transcription, err := t.transcriber.Transcript(ctx, audio)
	if err != nil {
		return err
	}

	err = t.publisher.Publish(audioId, workspaceId, uploadUserId, transcription, sessionID)
	if err != nil {
		return err
	}

	return nil
}
