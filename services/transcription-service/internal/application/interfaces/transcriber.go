package interfaces

import (
	"context"
	"io"
)

type Transcriber interface {
	Transcript(ctx context.Context, audio io.ReadCloser) (string, error)
}
