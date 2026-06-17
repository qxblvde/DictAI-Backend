package application

import (
	"context"
	"fmt"
	"time"

	"audio-ingest-service/internal/domain"
)

type SpeakerAssignment struct {
	Label string
	Name  string
	Email string
}

type ConfirmInput struct {
	SessionID     string
	UserID        string
	WorkspaceName string
	Speakers      []SpeakerAssignment
}

type ConfirmOutput struct {
	WorkspaceID string
}

type DiarizedSegment struct {
	ParticipantID string  `json:"participantId"`
	Start         float64 `json:"start"`
	End           float64 `json:"end"`
	Text          string  `json:"text"`
}

type DiarizedPayload struct {
	AudioID      string            `json:"audioId"`
	WorkspaceID  string            `json:"workspaceId"`
	UploadUserID string            `json:"uploadUserId"`
	Segments     []DiarizedSegment `json:"segments"`
}

type DiarizedPublisher interface {
	PublishDiarized(ctx context.Context, payload DiarizedPayload) error
}

type ConfirmService struct {
	repo           domain.AnonymousSessionRepository
	storage        Storage
	workspaceClient WorkspaceClient
	voiceClient    VoiceProfileClient
	publisher      DiarizedPublisher
}

func NewConfirmService(
	repo domain.AnonymousSessionRepository,
	storage Storage,
	workspaceClient WorkspaceClient,
	voiceClient VoiceProfileClient,
	publisher DiarizedPublisher,
) *ConfirmService {
	return &ConfirmService{
		repo:            repo,
		storage:         storage,
		workspaceClient: workspaceClient,
		voiceClient:     voiceClient,
		publisher:       publisher,
	}
}

func (s *ConfirmService) Confirm(ctx context.Context, in ConfirmInput) (*ConfirmOutput, error) {
	session, err := s.repo.GetByID(ctx, in.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	if session.OwnerUserID != in.UserID {
		return nil, fmt.Errorf("forbidden")
	}
	if session.Status != domain.SessionStatusReady {
		return nil, fmt.Errorf("session is not ready (status: %s)", session.Status)
	}

	dbSpeakers, err := s.repo.GetSpeakers(ctx, in.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get speakers: %w", err)
	}

	// Validate all labels in request exist in DB
	dbLabels := map[string]domain.AnonymousSpeaker{}
	for _, sp := range dbSpeakers {
		dbLabels[sp.Label] = sp
	}
	for _, a := range in.Speakers {
		if _, ok := dbLabels[a.Label]; !ok {
			return nil, fmt.Errorf("unknown speaker label: %s", a.Label)
		}
	}

	workspaceName := in.WorkspaceName
	if workspaceName == "" {
		workspaceName = fmt.Sprintf("Meeting %s", time.Now().Format("2006-01-02"))
	}

	workspaceID, err := s.workspaceClient.CreateWorkspace(ctx, in.UserID, workspaceName)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	labelToParticipant := map[string]string{}

	for _, assignment := range in.Speakers {
		sp := dbLabels[assignment.Label]

		participantID, err := s.workspaceClient.AddParticipant(ctx, in.UserID, workspaceID, assignment.Name, assignment.Email)
		if err != nil {
			return nil, fmt.Errorf("add participant %s: %w", assignment.Label, err)
		}

		fragmentReader, err := s.storage.Get(ctx, sp.FragmentKey)
		if err != nil {
			return nil, fmt.Errorf("get fragment for %s: %w", assignment.Label, err)
		}

		voiceProfileID, err := s.voiceClient.CreateLibraryProfile(ctx, in.UserID, assignment.Name, fragmentReader)
		_ = fragmentReader.Close()
		if err != nil {
			return nil, fmt.Errorf("create library profile for %s: %w", assignment.Label, err)
		}

		if err := s.voiceClient.AssignProfile(ctx, in.UserID, workspaceID, participantID, voiceProfileID); err != nil {
			return nil, fmt.Errorf("assign profile for %s: %w", assignment.Label, err)
		}

		labelToParticipant[assignment.Label] = participantID
	}

	// Build audio.diarized payload from stored segments
	var allSegments []DiarizedSegment
	for _, sp := range dbSpeakers {
		participantID := labelToParticipant[sp.Label]
		for _, seg := range sp.Segments {
			allSegments = append(allSegments, DiarizedSegment{
				ParticipantID: participantID,
				Start:         seg.Start,
				End:           seg.End,
				Text:          seg.Text,
			})
		}
	}

	// Sort segments by start time
	sortSegmentsByStart(allSegments)

	diarizedPayload := DiarizedPayload{
		AudioID:      session.AudioID,
		WorkspaceID:  workspaceID,
		UploadUserID: in.UserID,
		Segments:     allSegments,
	}
	if err := s.publisher.PublishDiarized(ctx, diarizedPayload); err != nil {
		return nil, fmt.Errorf("publish audio.diarized: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, in.SessionID, domain.SessionStatusConfirmed); err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return &ConfirmOutput{WorkspaceID: workspaceID}, nil
}

func sortSegmentsByStart(segs []DiarizedSegment) {
	for i := 1; i < len(segs); i++ {
		for j := i; j > 0 && segs[j].Start < segs[j-1].Start; j-- {
			segs[j], segs[j-1] = segs[j-1], segs[j]
		}
	}
}
