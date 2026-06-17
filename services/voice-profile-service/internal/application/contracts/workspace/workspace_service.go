package workspace

type Service interface {
	GetParticipants(userId, workspaceId string) ([]string, error)
}
