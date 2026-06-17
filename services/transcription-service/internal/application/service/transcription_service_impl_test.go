package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type mockAudioService struct {
	rc     io.ReadCloser
	err    error
	called bool
}

func (m *mockAudioService) GetAudio(audioId, workspaceId, uploadUserId string) (io.ReadCloser, error) {
	m.called = true
	return m.rc, m.err
}

type mockTranscriber struct {
	resp string
	err  error
	got  []byte
}

func (m *mockTranscriber) Transcript(ctx context.Context, audio io.ReadCloser) (string, error) {
	if audio != nil {
		b, _ := io.ReadAll(audio)
		m.got = b
	}
	return m.resp, m.err
}

type mockPublisher struct {
	err           error
	called        bool
	audioId       string
	workspaceId   string
	uploadUserId  string
	transcription string
	sessionID     string
}

func (m *mockPublisher) Publish(audioId, workSpaceId, uploadUserId, transcription, sessionID string) error {
	m.called = true
	m.audioId = audioId
	m.workspaceId = workSpaceId
	m.uploadUserId = uploadUserId
	m.transcription = transcription
	m.sessionID = sessionID
	return m.err
}

func TestTranscript_Success(t *testing.T) {
	audioData := []byte("audio-bytes")
	audio := io.NopCloser(bytes.NewReader(audioData))

	ma := &mockAudioService{rc: audio}
	mt := &mockTranscriber{resp: "transcribed"}
	mp := &mockPublisher{}

	svc := NewTranscriptionService(mt, mp, ma)
	err := svc.Transcript(context.Background(), "aid", "wid", "uid", "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !ma.called {
		t.Error("expected audioService.GetAudio to be called")
	}
	if string(mt.got) != string(audioData) {
		t.Errorf("transcriber got wrong audio: %q", mt.got)
	}
	if !mp.called || mp.transcription != "transcribed" {
		t.Errorf("publisher called=%v transcription=%q", mp.called, mp.transcription)
	}
}

func TestTranscript_GetAudioError(t *testing.T) {
	want := errors.New("get audio failed")
	ma := &mockAudioService{err: want}
	mt := &mockTranscriber{resp: "x"}
	mp := &mockPublisher{}

	svc := NewTranscriptionService(mt, mp, ma)
	err := svc.Transcript(context.Background(), "a", "w", "u", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTranscript_TranscriberError(t *testing.T) {
	ma := &mockAudioService{rc: io.NopCloser(bytes.NewReader([]byte("a")))}
	want := errors.New("transcribe failed")
	mt := &mockTranscriber{err: want}
	mp := &mockPublisher{}

	svc := NewTranscriptionService(mt, mp, ma)
	err := svc.Transcript(context.Background(), "a", "w", "u", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mp.called {
		t.Fatal("publisher should not be called on transcriber error")
	}
}

func TestTranscript_PublishError(t *testing.T) {
	ma := &mockAudioService{rc: io.NopCloser(bytes.NewReader([]byte("a")))}
	mt := &mockTranscriber{resp: "ok"}
	want := errors.New("publish failed")
	mp := &mockPublisher{err: want}

	svc := NewTranscriptionService(mt, mp, ma)
	err := svc.Transcript(context.Background(), "a", "w", "u", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !mp.called {
		t.Fatal("expected publisher to be called")
	}
}
