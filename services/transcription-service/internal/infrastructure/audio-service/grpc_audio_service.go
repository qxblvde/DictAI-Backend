package audio_service

import (
	"context"
	"errors"
	"io"

	"github.com/Microservices/services/transcription-service/proto"
)

type GrpcAudioService struct {
	client proto.AudioIngestServiceClient
}

func NewGrpcAudioService(client proto.AudioIngestServiceClient) GrpcAudioService {
	return GrpcAudioService{client: client}
}

func (g GrpcAudioService) GetAudio(audioId, workspaceId, uploadUserId string) (io.ReadCloser, error) {
	if g.client == nil {
		return nil, errors.New("audio ingest gRPC client is nil")
	}

	stream, err := g.client.GetAudio(context.Background(), &proto.GetAudioRequest{
		AudioId:        audioId,
		WorkspaceId:    workspaceId,
		UploaderUserId: uploadUserId,
	})

	if err != nil {
		return nil, err
	}

	return newGrpcStreamReader(stream), nil
}
