package interfaces

import (
	"io"
)

type DiarizationService interface {
	GetDiarization(workspaceId, userId string, audioSegment io.ReadCloser) (string, error)
}
