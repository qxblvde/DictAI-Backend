package service

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/Microservices/services/voice-profile-service/internal/application/contracts/persistence"
)

type mockRepo struct {
	createdID       string
	saved           [192]float32
	savedFor        string
	savedOwner      string
	savedName       string
	assignedOwner   string
	assignedWS      string
	assignedPart    string
	assignedProfile string
	profiles        []persistence.ProfileSummary
	err             error
}

func (m *mockRepo) CreateProfile(embeddings [192]float32, ownerUserId, participantId, displayName string) (string, error) {
	m.saved = embeddings
	m.savedOwner = ownerUserId
	m.savedFor = participantId
	m.savedName = displayName
	return m.createdID, m.err
}

func (m *mockRepo) CreateLibraryProfile(embeddings [192]float32, ownerUserId, displayName string) (string, error) {
	m.saved = embeddings
	m.savedOwner = ownerUserId
	m.savedName = displayName
	return m.createdID, m.err
}

func (m *mockRepo) AssignProfile(ownerUserId, workspaceId, participantId, voiceProfileId string) error {
	m.assignedOwner = ownerUserId
	m.assignedWS = workspaceId
	m.assignedPart = participantId
	m.assignedProfile = voiceProfileId
	return m.err
}

func (m *mockRepo) ListProfiles(_ string) ([]persistence.ProfileSummary, error) {
	return m.profiles, m.err
}

func (m *mockRepo) GetProfile(_ string) ([192]float32, error) { return m.saved, m.err }

type mockEmbeddingService struct {
	emb [192]float32
	err error
}

func (m *mockEmbeddingService) Get(audio io.ReadCloser) ([192]float32, error) {
	_, _ = io.Copy(io.Discard, audio)
	return m.emb, m.err
}

type mockWorkspaceService struct {
	participants []string
	err          error
}

func (m *mockWorkspaceService) GetParticipants(_, _ string) ([]string, error) {
	return m.participants, m.err
}

func TestCreateVoiceProfile_Success(t *testing.T) {
	var expected [192]float32
	for i := 0; i < 192; i++ {
		expected[i] = float32(i)
	}
	repo := &mockRepo{createdID: "uuid-123"}
	embSvc := &mockEmbeddingService{emb: expected}
	workspaceSvc := &mockWorkspaceService{participants: []string{"participant-1", "participant-2"}}

	svc := NewVoiceProfileService(repo, embSvc, workspaceSvc)

	audio := io.NopCloser(bytes.NewReader([]byte("dummy")))
	defer func() {
		_ = audio.Close()
	}()

	id, err := svc.CreateVoiceProfile(audio, "user-1", "workspace-1", "participant-2", "Marta")
	if err != nil {
		t.Fatalf("CreateVoiceProfile returned error: %v", err)
	}
	if id != "uuid-123" {
		t.Fatalf("unexpected id: %s", id)
	}
	for i := 0; i < 192; i++ {
		if repo.saved[i] != expected[i] {
			t.Fatalf("saved embedding mismatch at %d: got %v want %v", i, repo.saved[i], expected[i])
		}
	}
	if repo.savedOwner != "user-1" {
		t.Fatalf("unexpected owner id: %s", repo.savedOwner)
	}
	if repo.savedFor != "participant-2" {
		t.Fatalf("unexpected participant id: %s", repo.savedFor)
	}
	if repo.savedName != "Marta" {
		t.Fatalf("unexpected display name: %s", repo.savedName)
	}
	if repo.assignedProfile != "uuid-123" || repo.assignedPart != "participant-2" || repo.assignedWS != "workspace-1" {
		t.Fatalf("profile was not assigned to participant: repo=%+v", repo)
	}
}

func TestCreateVoiceProfile_EmbeddingError(t *testing.T) {
	repo := &mockRepo{}
	embSvc := &mockEmbeddingService{err: io.ErrUnexpectedEOF}
	workspaceSvc := &mockWorkspaceService{participants: []string{"participant-1"}}
	svc := NewVoiceProfileService(repo, embSvc, workspaceSvc)

	audio := io.NopCloser(bytes.NewReader([]byte("dummy")))
	defer func() {
		_ = audio.Close()
	}()

	_, err := svc.CreateVoiceProfile(audio, "user-1", "workspace-1", "participant-1", "")
	if err == nil {
		t.Fatalf("expected error from embedding service")
	}
}

func TestListVoiceProfiles_Success(t *testing.T) {
	participantID := "participant-1"
	repo := &mockRepo{profiles: []persistence.ProfileSummary{{
		VoiceProfileID: "profile-1",
		ParticipantID:  &participantID,
		DisplayName:    "Participant One",
		CreatedAt:      time.Date(2026, 5, 11, 16, 0, 0, 0, time.UTC),
	}}}
	svc := NewVoiceProfileService(repo, &mockEmbeddingService{}, &mockWorkspaceService{})

	profiles, err := svc.ListVoiceProfiles("user-1")
	if err != nil {
		t.Fatalf("ListVoiceProfiles returned error: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("unexpected profiles count: %d", len(profiles))
	}
	if profiles[0].VoiceProfileID != "profile-1" || profiles[0].ParticipantID == nil || *profiles[0].ParticipantID != participantID {
		t.Fatalf("unexpected profile summary: %+v", profiles[0])
	}
}

func TestGetVoiceProfile_Success(t *testing.T) {
	var expected [192]float32
	for i := 0; i < 192; i++ {
		expected[i] = float32(i * 2)
	}
	repo := &mockRepo{saved: expected}
	embSvc := &mockEmbeddingService{}
	workspaceSvc := &mockWorkspaceService{}
	svc := NewVoiceProfileService(repo, embSvc, workspaceSvc)

	got, err := svc.GetVoiceProfile("uuid-123")
	if err != nil {
		t.Fatalf("GetVoiceProfile returned error: %v", err)
	}
	for i := 0; i < 192; i++ {
		if got[i] != expected[i] {
			t.Fatalf("profile mismatch at %d: got %v want %v", i, got[i], expected[i])
		}
	}
}
