package domain

type WorkspacesRepository interface {
	CreateWorkspace(workspace *Workspace) error
	FindWorkspace(workspaceID string) (*Workspace, error)
	GetByOwnerID(ownerID string, limit, offset int) ([]*Workspace, int, error)
	UpdateWorkspace(workspaceID, name string) (*Workspace, error)
	DeleteWorkspace(workspaceID string) error
}

type ParticipantRepository interface {
	AddWorkspaceParticipant(participant *Participant) error
	GetWorkspaceParticipants(workspaceID string) ([]*Participant, error)
	DeleteParticipant(workspaceID, participantID string) error
	UpdateParticipant(workspaceID, participantID, name, email string) (*Participant, error)
}
