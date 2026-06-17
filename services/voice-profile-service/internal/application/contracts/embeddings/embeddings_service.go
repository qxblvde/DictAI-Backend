package embeddings

import "io"

type EmbeddingService interface {
	Get(audio io.ReadCloser) ([192]float32, error)
}
