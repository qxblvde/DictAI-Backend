package interfaces

import (
	"github.com/Microservices/services/transcript-builder-service/internal/model"
)

type AnonymousSpeaker struct {
	Label        string
	FragmentKey  string
	FragmentData []byte
	Segments     []model.ResultSegment
}

type Publisher interface {
	Publish(audioId, workspaceId, uploadUserId string, segments []model.ResultSegment) error
	PublishAnonymous(sessionID, audioID, ownerUserID string, speakers []AnonymousSpeaker) error
}
