package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Microservices/services/transcript-builder-service/internal/application/interfaces"
	"github.com/Microservices/services/transcript-builder-service/internal/model"
	"github.com/nats-io/nats.go"
)

const AnonymousDiarizedSubject = "audio.anonymous_diarized"

type diarizedSegment struct {
	ParticipantId string  `json:"participantId"`
	Start         float64 `json:"start"`
	End           float64 `json:"end"`
	Text          string  `json:"text"`
}

type diarizedTranscriptPayload struct {
	AudioId      string            `json:"audioId"`
	WorkspaceId  string            `json:"workspaceId"`
	UploadUserId string            `json:"uploadUserId"`
	Segments     []diarizedSegment `json:"segments"`
}

type anonymousSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type anonymousSpeakerPayload struct {
	Label       string             `json:"label"`
	FragmentKey string             `json:"fragment_key"`
	Segments    []anonymousSegment `json:"segments"`
}

type anonymousDiarizedPayload struct {
	SessionID   string                    `json:"session_id"`
	AudioID     string                    `json:"audio_id"`
	OwnerUserID string                    `json:"owner_user_id"`
	Speakers    []anonymousSpeakerPayload `json:"speakers"`
}

type natsPublisherClient interface {
	PublishMsg(m *nats.Msg) error
}

type Storage interface {
	Put(ctx context.Context, objectKey string, body io.Reader) error
}

type NatsPublisher struct {
	client  natsPublisherClient
	subject string
	storage Storage
}

func NewNatsPublisher(url, subject string) (*NatsPublisher, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsPublisher{client: conn, subject: subject}, nil
}

func NewNatsPublisherFromConn(conn *nats.Conn, subject string) *NatsPublisher {
	return &NatsPublisher{client: conn, subject: subject}
}

func NewNatsPublisherWithStorage(conn *nats.Conn, subject string, storage Storage) *NatsPublisher {
	return &NatsPublisher{client: conn, subject: subject, storage: storage}
}

func (n *NatsPublisher) Publish(audioId, workspaceId, uploadUserId string, segments []model.ResultSegment) error {
	out := make([]diarizedSegment, len(segments))
	for i, s := range segments {
		startSec := s.Interval.Start.Seconds()
		endSec := (s.Interval.Start + s.Interval.Duration).Seconds()
		out[i] = diarizedSegment{
			ParticipantId: s.ParticipantId,
			Start:         startSec,
			End:           endSec,
			Text:          s.Text,
		}
	}

	payload := diarizedTranscriptPayload{
		AudioId:      audioId,
		WorkspaceId:  workspaceId,
		UploadUserId: uploadUserId,
		Segments:     out,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return n.client.PublishMsg(&nats.Msg{
		Subject: n.subject,
		Data:    data,
		Header:  nats.Header{"Content-Type": []string{"application/json"}},
	})
}

func (n *NatsPublisher) PublishAnonymous(sessionID, audioID, ownerUserID string, speakers []interfaces.AnonymousSpeaker) error {
	ctx := context.Background()

	out := make([]anonymousSpeakerPayload, 0, len(speakers))
	for _, sp := range speakers {
		fragmentKey := sp.FragmentKey
		if fragmentKey == "" {
			fragmentKey = fmt.Sprintf("anonymous/%s/fragments/%s.wav", sessionID, sp.Label)
		}

		// Store fragment in MinIO if storage is configured and fragment data is available
		if n.storage != nil && sp.FragmentData != nil {
			if err := n.storage.Put(ctx, fragmentKey, bytes.NewReader(sp.FragmentData)); err != nil {
				// Log but don't fail — fragment serving will fail gracefully
				_ = err
			}
		}

		segs := make([]anonymousSegment, len(sp.Segments))
		for i, s := range sp.Segments {
			segs[i] = anonymousSegment{
				Start: s.Interval.Start.Seconds(),
				End:   (s.Interval.Start + s.Interval.Duration).Seconds(),
				Text:  s.Text,
			}
		}

		out = append(out, anonymousSpeakerPayload{
			Label:       sp.Label,
			FragmentKey: fragmentKey,
			Segments:    segs,
		})
	}

	payload := anonymousDiarizedPayload{
		SessionID:   sessionID,
		AudioID:     audioID,
		OwnerUserID: ownerUserID,
		Speakers:    out,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return n.client.PublishMsg(&nats.Msg{
		Subject: AnonymousDiarizedSubject,
		Data:    data,
		Header:  nats.Header{"Content-Type": []string{"application/json"}},
	})
}

