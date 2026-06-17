package voiceprofile

import "io"

type ProfileSummary struct {
	VoiceProfileID string  `json:"voice_profile_id"`
	ParticipantID  *string `json:"participant_id,omitempty"`
	DisplayName    string  `json:"display_name"`
	CreatedAt      string  `json:"created_at"`
}

type VoiceProfileService interface {
	CreateVoiceProfile(audio io.ReadCloser, userId, workspaceId, participantId, displayName string) (string, error)
	CreateLibraryProfile(audio io.ReadCloser, userId, displayName string) (string, error)
	AssignVoiceProfile(userId, workspaceId, participantId, voiceProfileId string) error
	ListVoiceProfiles(userId string) ([]ProfileSummary, error)
	GetVoiceProfile(participantId string) ([192]float32, error)
}
