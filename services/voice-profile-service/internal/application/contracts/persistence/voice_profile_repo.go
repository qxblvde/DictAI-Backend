package persistence

import "time"

type ProfileSummary struct {
	VoiceProfileID string
	ParticipantID  *string
	OwnerUserID    *string
	DisplayName    string
	CreatedAt      time.Time
}

type VoiceProfileRepository interface {
	CreateProfile(embeddings [192]float32, ownerUserId, participantId, displayName string) (string, error)
	CreateLibraryProfile(embeddings [192]float32, ownerUserId, displayName string) (string, error)
	AssignProfile(ownerUserId, workspaceId, participantId, voiceProfileId string) error
	ListProfiles(ownerUserId string) ([]ProfileSummary, error)
	GetProfile(participantId string) ([192]float32, error)
}
