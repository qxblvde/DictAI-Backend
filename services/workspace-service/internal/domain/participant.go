package domain

type Participant struct {
	ParticipantID  string
	WorkspaceID    string
	Name           string
	Email          string
	VoiceProfileID *string
}
