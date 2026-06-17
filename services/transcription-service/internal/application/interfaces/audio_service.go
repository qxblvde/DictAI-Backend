package interfaces

import "io"

type AudioService interface {
	GetAudio(audioId, workspaceId, uploadUserId string) (io.ReadCloser, error)
}
