package audio_service

import (
	"io"

	"github.com/Microservices/services/transcript-builder-service/proto"
	"google.golang.org/grpc"
)

type grpcStreamReader struct {
	stream grpc.ServerStreamingClient[proto.AudioChunk]
	data   []byte
	closed bool
}

func newGrpcStreamReader(stream grpc.ServerStreamingClient[proto.AudioChunk]) *grpcStreamReader {
	return &grpcStreamReader{stream: stream, data: make([]byte, 0)}
}

func (g *grpcStreamReader) Read(p []byte) (int, error) {
	if g.closed {
		return 0, io.ErrClosedPipe
	}
	if len(g.data) == 0 {
		chunk, err := g.stream.Recv()
		if err == io.EOF {
			g.closed = true
			return 0, io.EOF
		} else if err != nil {
			return 0, err
		}
		g.data = chunk.Content
		if len(g.data) == 0 {
			return g.Read(p)
		}
	}
	n := copy(p, g.data)
	if n < len(g.data) {
		g.data = g.data[n:]
	} else {
		g.data = nil
	}
	return n, nil
}

func (g *grpcStreamReader) Close() error {
	if g.closed {
		return io.ErrClosedPipe
	}
	g.closed = true
	return nil
}
