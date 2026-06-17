package event

import (
	"context"

	"audio-ingest-service/internal/application"
)

const AudioDiarizedSubject = "audio.diarized"

func (p *NATSPublisher) PublishDiarized(ctx context.Context, payload application.DiarizedPayload) error {
	return p.publish(ctx, AudioDiarizedSubject, payload)
}
