package interfaces

import "context"

type Participant struct {
	ParticipantID string
	Name          string
}

type WorkspaceInfo struct {
	WorkspaceID string
	Name        string
}

type WorkspaceClient interface {
	GetParticipants(ctx context.Context, workspaceID, userID string) ([]Participant, error)
	GetWorkspace(ctx context.Context, workspaceID, userID string) (*WorkspaceInfo, error)
}
