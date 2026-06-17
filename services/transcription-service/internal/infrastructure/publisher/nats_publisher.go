package publisher

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

type AudioTranscribedPayload struct {
	AudioId       string `json:"audio_id"`
	WorkspaceId   string `json:"workspace_id"`
	UploadUserId  string `json:"upload_user_id"`
	Transcription string `json:"transcription"`
	SessionID     string `json:"session_id,omitempty"`
}

type NatsPublisher struct {
	client  natsPublisherClient
	subject string
}

func NewNatsPublisher(url string, subject string) (*NatsPublisher, error) {
	natsConn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsPublisher{client: natsConn, subject: subject}, nil
}

// NewNatsPublisherFromConn creates a publisher from an existing nats.Conn (useful for tests)
func NewNatsPublisherFromConn(conn *nats.Conn, subject string) *NatsPublisher {
	return &NatsPublisher{client: conn, subject: subject}
}

// natsPublisherClient is a small interface to allow testing without real NATS server.
type natsPublisherClient interface {
	PublishMsg(m *nats.Msg) error
}

func (n *NatsPublisher) Publish(audioId, workSpaceId, uploadUserid, transcription, sessionID string) error {
	payload := AudioTranscribedPayload{
		AudioId:       audioId,
		WorkspaceId:   workSpaceId,
		UploadUserId:  uploadUserid,
		Transcription: transcription,
		SessionID:     sessionID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: n.subject,
		Data:    data,
		Header:  nats.Header{"Content-Type": []string{"application/json"}},
	}

	err = n.client.PublishMsg(msg)
	if err != nil {
		return err
	}

	return nil
}
