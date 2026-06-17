package audio_service

import (
	"context"
	"io"

	"github.com/Microservices/services/transcript-builder-service/proto"
)

type GrpcAudioService struct {
	client proto.AudioIngestServiceClient
}

func NewGrpcAudioService(client proto.AudioIngestServiceClient) GrpcAudioService {
	return GrpcAudioService{client: client}
}

func (g GrpcAudioService) GetAudio(audioId, workspaceId, uploadUserId string) (io.ReadCloser, error) {
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
