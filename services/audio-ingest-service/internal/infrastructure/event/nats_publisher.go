package event

import (
	"context"
	"encoding/json"
	"fmt"

	"audio-ingest-service/internal/application"

	"github.com/nats-io/nats.go"
)

const (
	AudioUploadedSubject          = "audio.uploaded"
	AudioAnonymousUploadedSubject = "audio.uploaded"
)

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) PublishAudioUploaded(ctx context.Context, event application.UploadedEvent) error {
	return p.publish(ctx, AudioUploadedSubject, event)
}

func (p *NATSPublisher) PublishAnonymousAudioUploaded(ctx context.Context, event application.UploadedEvent) error {
	return p.publish(ctx, AudioUploadedSubject, event)
}

func (p *NATSPublisher) publish(ctx context.Context, subject string, v any) error {
	if p.conn == nil {
		return errorsNotConfigured("nats connection is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	if err := p.conn.Publish(subject, payload); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}

	return nil
}

func errorsNotConfigured(message string) error {
	return fmt.Errorf("publisher is not configured: %s", message)
}
