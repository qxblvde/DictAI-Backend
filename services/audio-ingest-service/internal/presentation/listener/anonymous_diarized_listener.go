package listener

import (
	"context"
	"encoding/json"
	"log/slog"

	"audio-ingest-service/internal/application"
	"audio-ingest-service/internal/domain"

	"github.com/nats-io/nats.go"
)

const AnonymousDiarizedSubject = "audio.anonymous_diarized"

type AnonymousDiarizedListener struct {
	conn *nats.Conn
	ch   chan *nats.Msg
	sub  *nats.Subscription
	repo domain.AnonymousSessionRepository
}

func NewAnonymousDiarizedListener(conn *nats.Conn, repo domain.AnonymousSessionRepository) (*AnonymousDiarizedListener, error) {
	ch := make(chan *nats.Msg, 64)
	sub, err := conn.ChanSubscribe(AnonymousDiarizedSubject, ch)
	if err != nil {
		return nil, err
	}
	return &AnonymousDiarizedListener{conn: conn, ch: ch, sub: sub, repo: repo}, nil
}

func (l *AnonymousDiarizedListener) Listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			_ = l.sub.Unsubscribe()
			return
		case msg := <-l.ch:
			go l.handle(ctx, msg)
		}
	}
}

func (l *AnonymousDiarizedListener) handle(ctx context.Context, msg *nats.Msg) {
	log := slog.Default().With("subject", msg.Subject)

	var event application.AnonymousDiarizedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Error("failed to unmarshal anonymous_diarized", "error", err)
		return
	}
	if event.SessionID == "" || event.AudioID == "" {
		log.Warn("invalid anonymous_diarized payload: missing session_id or audio_id")
		return
	}

	log = log.With("session_id", event.SessionID, "audio_id", event.AudioID)

	for _, sp := range event.Speakers {
		segs := make([]domain.SegmentInfo, len(sp.Segments))
		for i, s := range sp.Segments {
			segs[i] = domain.SegmentInfo{Start: s.Start, End: s.End, Text: s.Text}
		}
		speaker := &domain.AnonymousSpeaker{
			SessionID:   event.SessionID,
			Label:       sp.Label,
			FragmentKey: sp.FragmentKey,
			Segments:    segs,
		}
		if err := l.repo.AddSpeaker(ctx, speaker); err != nil {
			log.Error("failed to add speaker", "label", sp.Label, "error", err)
			return
		}
	}

	if err := l.repo.UpdateStatus(ctx, event.SessionID, domain.SessionStatusReady); err != nil {
		log.Error("failed to update session status to ready", "error", err)
		return
	}

	log.Info("anonymous session ready", "speakers", len(event.Speakers))
}
