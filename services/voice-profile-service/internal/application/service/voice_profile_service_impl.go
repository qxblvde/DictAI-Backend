package service

import (
	"database/sql"
	"errors"
	"io"
	"strings"

	"github.com/Microservices/services/voice-profile-service/internal/application/contracts/embeddings"
	"github.com/Microservices/services/voice-profile-service/internal/application/contracts/persistence"
	voiceprofile "github.com/Microservices/services/voice-profile-service/internal/application/contracts/voiceprofile"
	"github.com/Microservices/services/voice-profile-service/internal/application/contracts/workspace"
)

type voiceProfileService struct {
	repo             persistence.VoiceProfileRepository
	embeddingService embeddings.EmbeddingService
	workspaceService workspace.Service
}

func NewVoiceProfileService(repo persistence.VoiceProfileRepository,
	embeddingService embeddings.EmbeddingService, workspaceService workspace.Service) voiceprofile.VoiceProfileService {
	return &voiceProfileService{
		repo:             repo,
		embeddingService: embeddingService,
		workspaceService: workspaceService,
	}
}

func (v *voiceProfileService) CreateVoiceProfile(audio io.ReadCloser, userId, workspaceId, participantId, displayName string) (string, error) {
	if err := v.ensureParticipant(userId, workspaceId, participantId); err != nil {
		return "", err
	}

	embd, err := v.embeddingService.Get(audio)
	if err != nil {
		return "", err
	}

	createdProfileId, err := v.repo.CreateProfile(embd, userId, participantId, displayName)
	if err != nil {
		return "", err
	}

	if err := v.repo.AssignProfile(userId, workspaceId, participantId, createdProfileId); err != nil {
		return "", normalizeRepoErr(err)
	}

	return createdProfileId, nil
}

func (v *voiceProfileService) CreateLibraryProfile(audio io.ReadCloser, userId, displayName string) (string, error) {
	embd, err := v.embeddingService.Get(audio)
	if err != nil {
		return "", err
	}
	return v.repo.CreateLibraryProfile(embd, userId, displayName)
}

func (v *voiceProfileService) AssignVoiceProfile(userId, workspaceId, participantId, voiceProfileId string) error {
	if err := v.ensureParticipant(userId, workspaceId, participantId); err != nil {
		return err
	}
	if err := v.repo.AssignProfile(userId, workspaceId, participantId, voiceProfileId); err != nil {
		return normalizeRepoErr(err)
	}
	return nil
}

func (v *voiceProfileService) ListVoiceProfiles(userId string) ([]voiceprofile.ProfileSummary, error) {
	profiles, err := v.repo.ListProfiles(userId)
	if err != nil {
		return nil, err
	}
	out := make([]voiceprofile.ProfileSummary, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, voiceprofile.ProfileSummary{
			VoiceProfileID: p.VoiceProfileID,
			ParticipantID:  p.ParticipantID,
			DisplayName:    p.DisplayName,
			CreatedAt:      p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return out, nil
}

func (v *voiceProfileService) GetVoiceProfile(participantId string) ([192]float32, error) {
	return v.repo.GetProfile(participantId)
}

func (v *voiceProfileService) ensureParticipant(userId, workspaceId, participantId string) error {
	participants, err := v.workspaceService.GetParticipants(userId, workspaceId)
	if err != nil {
		return err
	}

	for _, participant := range participants {
		if participantId == participant {
			return nil
		}
	}

	return ErrParticipantNotFound
}

func normalizeRepoErr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrProfileNotFound
	}
	if strings.Contains(err.Error(), "access denied") {
		return ErrForbidden
	}
	return err
}
