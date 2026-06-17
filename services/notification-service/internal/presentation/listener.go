package presentation

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Microservices/services/notification-service/internal/application/service"
	"github.com/nats-io/nats.go"
)

type summarizedPayload struct {
	AudioID       string `json:"audioId"`
	WorkspaceID   string `json:"workspaceId"`
	UploadUserID  string `json:"uploadUserId"`
	SummaryURL    string `json:"summaryUrl"`
	TranscriptURL string `json:"transcriptUrl"`
}

type Listener struct {
	natsConn *nats.Conn
	ch       chan *nats.Msg
	sub      *nats.Subscription
	service  *service.NotificationService
}

func NewListener(natsURL, subject string, svc *service.NotificationService) (*Listener, error) {
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
	var p summarizedPayload
	if err := json.Unmarshal(msg.Data, &p); err != nil {
		slog.Error("failed to unmarshal payload", "error", err)
		return
	}
	if p.WorkspaceID == "" || p.UploadUserID == "" {
		slog.Warn("invalid payload: missing required fields")
		return
	}
	l.service.Notify(ctx, p.AudioID, p.WorkspaceID, p.UploadUserID, p.SummaryURL, p.TranscriptURL)
}
