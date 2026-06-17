package audio_service

import (
	"context"
	"io"
	"testing"

	"github.com/Microservices/services/transcription-service/proto"
	"google.golang.org/grpc/metadata"
)

type fakeAudioStream struct {
	chunks [][]byte
	idx    int
}

func (f *fakeAudioStream) Recv() (*proto.AudioChunk, error) {
	if f.idx >= len(f.chunks) {
		return nil, io.EOF
	}
	chunk := f.chunks[f.idx]
	f.idx++
	return &proto.AudioChunk{Content: chunk}, nil
}

func (f *fakeAudioStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (f *fakeAudioStream) Trailer() metadata.MD         { return metadata.MD{} }
func (f *fakeAudioStream) CloseSend() error             { return nil }
func (f *fakeAudioStream) Context() context.Context     { return context.Background() }
func (f *fakeAudioStream) SendMsg(any) error            { return nil }
func (f *fakeAudioStream) RecvMsg(any) error            { return nil }

func TestGrpcStreamReader_EOFIsSticky(t *testing.T) {
	reader := newGrpcStreamReader(&fakeAudioStream{chunks: [][]byte{[]byte("ab")}})

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(data) != "ab" {
		t.Fatalf("unexpected data: %q", string(data))
	}

	buf := make([]byte, 8)
	n, err := reader.Read(buf)
	if n != 0 || err != io.EOF {
		t.Fatalf("expected EOF after stream end, got n=%d err=%v", n, err)
	}
}

func TestGrpcStreamReader_CloseBehavior(t *testing.T) {
	reader := newGrpcStreamReader(&fakeAudioStream{chunks: [][]byte{[]byte("ab")}})

	if err := reader.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("unexpected second close error: %v", err)
	}

	buf := make([]byte, 8)
	n, err := reader.Read(buf)
	if n != 0 || err != io.ErrClosedPipe {
		t.Fatalf("expected ErrClosedPipe after close, got n=%d err=%v", n, err)
	}
}
