package publisher

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
)

type fakeClient struct {
	lastMsg *nats.Msg
	retErr  error
}

func (f *fakeClient) PublishMsg(m *nats.Msg) error {
	f.lastMsg = m
	return f.retErr
}

func TestPublish_Success(t *testing.T) {
	fc := &fakeClient{}
	p := &NatsPublisher{client: fc, subject: "subj"}

	err := p.Publish("a1", "w1", "u1", "trans-123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fc.lastMsg == nil {
		t.Fatal("expected PublishMsg to be called")
	}
	var payload AudioTranscribedPayload
	if err := json.Unmarshal(fc.lastMsg.Data, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload.AudioId != "a1" || payload.WorkspaceId != "w1" || payload.UploadUserId != "u1" || payload.Transcription != "trans-123" {
		t.Fatalf("payload mismatch: %#v", payload)
	}
	if got := fc.lastMsg.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content-type: %q", got)
	}
}

func TestPublish_ClientError(t *testing.T) {
	fc := &fakeClient{retErr: errors.New("publish failed")}
	p := &NatsPublisher{client: fc, subject: "s"}
	err := p.Publish("a", "w", "u", "t", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
