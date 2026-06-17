package audio_service

import (
	"io"

	"github.com/Microservices/services/transcription-service/proto"
)

type GrpcStreamReader struct {
	stream proto.AudioIngestService_GetAudioClient
	data   []byte
	eof    bool
	closed bool
}

func newGrpcStreamReader(stream proto.AudioIngestService_GetAudioClient) *GrpcStreamReader {
	return &GrpcStreamReader{
		stream: stream,
		data:   make([]byte, 0),
		eof:    false,
		closed: false,
	}
}

func (g *GrpcStreamReader) Read(p []byte) (n int, err error) {
	if g.closed {
		return 0, io.ErrClosedPipe
	}
	if g.eof {
		return 0, io.EOF
	}

	if len(g.data) == 0 {
		chunk, err := g.stream.Recv()
		if err == io.EOF {
			g.eof = true
			return 0, io.EOF
		} else if err != nil {
			return 0, err
		}

		g.data = chunk.Content
		if len(g.data) == 0 {
			return g.Read(p)
		}
	}

	n = copy(p, g.data)
	if n < len(g.data) {
		g.data = g.data[n:]
	} else {
		g.data = nil
	}

	return n, nil
}

func (g *GrpcStreamReader) Close() error {
	g.closed = true
	return nil
}
