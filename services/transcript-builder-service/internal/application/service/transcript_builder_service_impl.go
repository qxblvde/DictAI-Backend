package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/Microservices/services/transcript-builder-service/internal/application/interfaces"
	"github.com/Microservices/services/transcript-builder-service/internal/model"
	"github.com/Microservices/services/transcript-builder-service/internal/utils"
)

type transcriptBuilderService struct {
	builder      interfaces.TranscriptionBuilder
	audioService interfaces.AudioService
	publisher    interfaces.Publisher
}

func NewTranscriptBuilderService(
	builder interfaces.TranscriptionBuilder,
	audioService interfaces.AudioService,
	publisher interfaces.Publisher,
) interfaces.TranscriptBuilderService {
	return &transcriptBuilderService{
		builder:      builder,
		audioService: audioService,
		publisher:    publisher,
	}
}

func (s *transcriptBuilderService) Process(
	_ context.Context,
	audioId, workspaceId, uploadUserId, transcription string,
) error {
	segments, err := utils.ParseTranscription(transcription)
	if err != nil {
		return fmt.Errorf("parse transcription: %w", err)
	}

	audio, err := s.audioService.GetAudio(audioId, workspaceId, uploadUserId)
	if err != nil {
		return fmt.Errorf("get audio: %w", err)
	}
	defer func() { _ = audio.Close() }()

	audioData, err := io.ReadAll(audio)
	if err != nil {
		return fmt.Errorf("read audio: %w", err)
	}

	wavData, err := convertToWAV(audioData)
	if err != nil {
		return fmt.Errorf("convert to wav: %w", err)
	}

	results, err := s.builder.GetTranscriptionWithDiarization(workspaceId, uploadUserId, segments, io.NopCloser(bytes.NewReader(wavData)))
	if err != nil {
		return fmt.Errorf("diarize: %w", err)
	}

	return s.publisher.Publish(audioId, workspaceId, uploadUserId, results)
}

func (s *transcriptBuilderService) ProcessAnonymous(
	_ context.Context,
	audioId, uploadUserId, sessionID, transcription string,
) error {
	segments, err := utils.ParseTranscription(transcription)
	if err != nil {
		return fmt.Errorf("parse transcription: %w", err)
	}

	// For anonymous mode, audio is stored under anonymous/{audioId}.mp3
	audio, err := s.audioService.GetAudio(audioId, "", uploadUserId)
	if err != nil {
		return fmt.Errorf("get audio: %w", err)
	}
	defer func() { _ = audio.Close() }()

	audioData, err := io.ReadAll(audio)
	if err != nil {
		return fmt.Errorf("read audio: %w", err)
	}

	// Convert to 16kHz mono WAV for speaker clustering
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return fmt.Errorf("convert to wav: %w", err)
	}

	clusters, err := s.builder.ClusterBySpeaker(segments, wavData, sessionID)
	if err != nil {
		return fmt.Errorf("cluster speakers: %w", err)
	}

	// Convert clusters to Publisher format
	pubSpeakers := make([]interfaces.AnonymousSpeaker, len(clusters))
	for i, c := range clusters {
		segs := make([]model.ResultSegment, len(c.Segments))
		for j, seg := range c.Segments {
			segs[j] = model.ResultSegment{
				Interval:      seg.Interval,
				Text:          seg.Text,
				ParticipantId: c.Label,
			}
		}
		pubSpeakers[i] = interfaces.AnonymousSpeaker{
			Label:        c.Label,
			FragmentData: c.FragmentData,
			Segments:     segs,
		}
	}

	return s.publisher.PublishAnonymous(sessionID, audioId, uploadUserId, pubSpeakers)
}

// convertToWAV converts any audio format to 16kHz mono WAV using ffmpeg.
func convertToWAV(data []byte) ([]byte, error) {
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-ar", "16000",
		"-ac", "1",
		"-f", "wav",
		"pipe:1",
	)
	cmd.Stdin = bytes.NewReader(data)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, errBuf.String())
	}
	return out.Bytes(), nil
}

