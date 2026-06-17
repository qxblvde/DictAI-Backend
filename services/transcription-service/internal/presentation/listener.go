package presentation

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Microservices/services/transcription-service/internal/application/interfaces"
	"github.com/nats-io/nats.go"
)

type AudioUploadedPayload struct {
	AudioId      string `json:"audio_id"`
	WorkspaceId  string `json:"workspace_id"`
	UploadUserId string `json:"uploader_user_id"`
	SessionID    string `json:"session_id,omitempty"`
}

type Listener struct {
	natsConn             *nats.Conn
	ch                   chan *nats.Msg
	sub                  *nats.Subscription
	transcriptionService interfaces.TranscriptionService
}

func NewListener(natsURL, subject string, transcriptionService interfaces.TranscriptionService) (*Listener, error) {
	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	ch := make(chan *nats.Msg, 64)
	sub, err := natsConn.ChanSubscribe(subject, ch)
	if err != nil {
		return nil, err
	}

	return &Listener{natsConn: natsConn, ch: ch, sub: sub, transcriptionService: transcriptionService}, nil
}

func (l *Listener) Listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("nats listener stopping", "reason", ctx.Err())
			_ = l.sub.Unsubscribe()
			return
		case msg := <-l.ch:
			go l.handleMessage(ctx, msg)
		}
	}
}

func (l *Listener) handleMessage(ctx context.Context, msg *nats.Msg) {
	log := slog.Default().With("subject", msg.Subject)
	log.Debug("nats message received", "payload", string(msg.Data))

	payload := &AudioUploadedPayload{}
	if err := json.Unmarshal(msg.Data, payload); err != nil {
		log.Error("failed to unmarshal audio uploaded payload", "error", err)
		return
	}

	if payload.AudioId == "" || payload.UploadUserId == "" {
		log.Warn("invalid audio uploaded payload: missing required fields",
			"audio_id", payload.AudioId,
			"workspace_id", payload.WorkspaceId,
		)
		return
	}

	log = log.With("audio_id", payload.AudioId, "workspace_id", payload.WorkspaceId)
	log.Info("processing transcription")

	if err := l.transcriptionService.Transcript(ctx, payload.AudioId, payload.WorkspaceId, payload.UploadUserId, payload.SessionID); err != nil {
		log.Error("transcription failed", "error", err)
		return
	}

	log.Info("transcription completed successfully")
}
