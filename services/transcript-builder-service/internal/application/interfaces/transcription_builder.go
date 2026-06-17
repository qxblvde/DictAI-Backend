package interfaces

import (
	"io"

	"github.com/Microservices/services/transcript-builder-service/internal/model"
)

type TranscriptionSegment struct {
	Interval model.Interval
	Text     string
	Speaker  string
}

type SpeakerCluster struct {
	Label        string
	FragmentData []byte // WAV bytes of the representative fragment
	Segments     []TranscriptionSegment
}

type TranscriptionBuilder interface {
	GetTranscriptionWithDiarization(workspaceId, userId string, segments []TranscriptionSegment, audio io.ReadCloser) ([]model.ResultSegment, error)
	ClusterBySpeaker(segments []TranscriptionSegment, audioData []byte, sessionID string) ([]SpeakerCluster, error)
}
