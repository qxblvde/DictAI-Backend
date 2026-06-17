package presentation

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Microservices/services/summarization-service/internal/application/interfaces"
	"github.com/Microservices/services/summarization-service/internal/application/service"
	"github.com/nats-io/nats.go"
)

type diarizedPayload struct {
	AudioID      string               `json:"audioId"`
	WorkspaceID  string               `json:"workspaceId"`
	UploadUserID string               `json:"uploadUserId"`
	Segments     []interfaces.Segment `json:"segments"`
}

type Listener struct {
	natsConn *nats.Conn
	ch       chan *nats.Msg
	sub      *nats.Subscription
	service  *service.SummarizationService
}

func NewListener(natsURL, subject string, svc *service.SummarizationService) (*Listener, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	ch := make(chan *nats.Msg, 64)
	sub, err := conn.ChanSubscribe(subject, ch)
	if err != nil {
		return nil, err
	}
	return &Listener{natsConn: conn, ch: ch, sub: sub, service: svc}, nil
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

	var payload diarizedPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Error("failed to unmarshal diarized payload", "error", err)
		return
	}

	if payload.AudioID == "" || payload.WorkspaceID == "" || payload.UploadUserID == "" {
		log.Warn("invalid diarized payload: missing required fields",
			"audio_id", payload.AudioID,
			"workspace_id", payload.WorkspaceID,
		)
		return
	}

	log = log.With("audio_id", payload.AudioID, "workspace_id", payload.WorkspaceID)
	log.Info("processing summarization", "segments", len(payload.Segments))

	if err := l.service.Process(ctx, payload.AudioID, payload.WorkspaceID, payload.UploadUserID, payload.Segments); err != nil {
		log.Error("summarization failed", "error", err)
		return
	}

	log.Info("summarization completed successfully")
}
