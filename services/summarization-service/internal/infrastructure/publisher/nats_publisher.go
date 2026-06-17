package publisher

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

type summarizedPayload struct {
	AudioID       string `json:"audioId"`
	WorkspaceID   string `json:"workspaceId"`
	UploadUserID  string `json:"uploadUserId"`
	SummaryURL    string `json:"summaryUrl"`
	TranscriptURL string `json:"transcriptUrl"`
}

type NatsPublisher struct {
	conn    *nats.Conn
	subject string
}

func NewNatsPublisher(url, subject string) (*NatsPublisher, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &NatsPublisher{conn: conn, subject: subject}, nil
}

func (p *NatsPublisher) Publish(audioID, workspaceID, uploadUserID, summaryURL, transcriptURL string) error {
	payload := summarizedPayload{
		AudioID:       audioID,
		WorkspaceID:   workspaceID,
		UploadUserID:  uploadUserID,
		SummaryURL:    summaryURL,
		TranscriptURL: transcriptURL,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	return p.conn.PublishMsg(&nats.Msg{
		Subject: p.subject,
		Data:    data,
		Header:  nats.Header{"Content-Type": []string{"application/json"}},
	})
}
