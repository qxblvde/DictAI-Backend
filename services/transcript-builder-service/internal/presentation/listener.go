package presentation

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Microservices/services/transcript-builder-service/internal/application/interfaces"
	"github.com/nats-io/nats.go"
)

type AudioTranscribedPayload struct {
	AudioId       string `json:"audio_id"`
	WorkspaceId   string `json:"workspace_id"`
	UploadUserId  string `json:"upload_user_id"`
	Transcription string `json:"transcription"`
	SessionID     string `json:"session_id,omitempty"`
}

type Listener struct {
	natsConn *nats.Conn
	ch       chan *nats.Msg
	sub      *nats.Subscription
	service  interfaces.TranscriptBuilderService
}

func NewListener(natsURL, subject string, service interfaces.TranscriptBuilderService) (*Listener, error) {
	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	ch := make(chan *nats.Msg, 64)
	sub, err := natsConn.ChanSubscribe(subject, ch)
	if err != nil {
		return nil, err
	}
	return &Listener{natsConn: natsConn, ch: ch, sub: sub, service: service}, nil
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

	var payload AudioTranscribedPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Error("failed to unmarshal audio transcribed payload", "error", err)
		return
	}

	if payload.AudioId == "" {
		log.Warn("invalid audio transcribed payload: missing audio_id")
		return
	}

	log = log.With("audio_id", payload.AudioId, "workspace_id", payload.WorkspaceId)

	// Anonymous mode: workspace_id is empty, session_id is set
	if payload.WorkspaceId == "" && payload.SessionID != "" {
		log.Info("processing anonymous diarization", "session_id", payload.SessionID)
		if err := l.service.ProcessAnonymous(ctx, payload.AudioId, payload.UploadUserId, payload.SessionID, payload.Transcription); err != nil {
			log.Error("anonymous diarization failed", "error", err)
		}
		return
	}

	if payload.WorkspaceId == "" {
		log.Warn("invalid audio transcribed payload: missing workspace_id and session_id")
		return
	}

	log.Info("processing diarization")
	if err := l.service.Process(ctx, payload.AudioId, payload.WorkspaceId, payload.UploadUserId, payload.Transcription); err != nil {
		log.Error("diarization failed", "error", err)
		return
	}

	log.Info("diarization completed successfully")
}
