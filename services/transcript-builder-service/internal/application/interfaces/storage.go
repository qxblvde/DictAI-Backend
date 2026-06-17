package interfaces

import (
	"context"
	"io"
)

type Storage interface {
	Put(ctx context.Context, objectKey string, body io.Reader) error
}
