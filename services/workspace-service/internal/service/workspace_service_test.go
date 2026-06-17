package service

import (
	"errors"
	"testing"

	"github.com/DictAI/Microservices/services/workspace-service/internal/domain"
	"github.com/DictAI/Microservices/services/workspace-service/internal/domain/mocks"
	"github.com/golang/mock/gomock"
)

func InitialSetup(t *testing.T) (*WorkspaceService, *mocks.MockWorkspacesRepository,
	*mocks.MockParticipantRepository) {

	controller := gomock.NewController(t)

	workspaceRepoMock := mocks.NewMockWorkspacesRepository(controller)
	participantRepoMock := mocks.NewMockParticipantRepository(controller)

	service := NewWorkspaceService(workspaceRepoMock, participantRepoMock)

	return service, workspaceRepoMock, participantRepoMock
}

func TestCreationOfWorkspace_CorrectCreation(t *testing.T) {
	service, workspaceRepoMock, _ := InitialSetup(t)

	workspaceRepoMock.EXPECT().CreateWorkspace(gomock.Any()).Return(nil)
	workspace, err := service.CreateWorkspace("userId", "correctName")

	if err != nil {
		t.Fatalf("Unexpected error occurred while creating the workspace, got %v", err)
	}
	if workspace == nil {
		t.Fatal("Workspace is expected to return, got nil")
	}
	if workspace.Name != "correctName" {
		t.Errorf("Expected workspace name : \"correctName\", got %v", workspace.Name)
	}
	if workspace.OwnerID != "userId" {
		t.Errorf("Expected ownerID : \"userId\", got %v", workspace.OwnerID)
	}
}

func TestCreationOfWorkspace_InvalidName(t *testing.T) {
	service, _, _ := InitialSetup(t)

	workspace, err := service.CreateWorkspace("userId", "")

	if !errors.Is(err, ErrInvalidWorkspaceName) {
		t.Errorf("Expected ErrInvalidWorkspaceName, got %v", err)
	}
	if workspace != nil {
		t.Errorf("Expected nil workspace, got %v", workspace)
	}
}

func TestCreationOfWorkspace_RepositoryError(t *testing.T) {
	service, workspaceRepoMock, _ := InitialSetup(t)

	dbErr := errors.New("DB insert error")
	workspaceRepoMock.EXPECT().CreateWorkspace(gomock.Any()).Return(dbErr)

	workspace, err := service.CreateWorkspace("userId", "workspaceName")

	if err != dbErr {
		t.Errorf("Expected db error, got %v", err)
	}
	if workspace != nil {
		t.Errorf("Expected nil workspace, got %v", workspace)
	}
}

func TestAddParticipant_CorrectAdding(t *testing.T) {
	service, workspaceRepoMock, participantRepoMock := InitialSetup(t)

	workspace := &domain.Workspace{
		WorkspaceID: "workspaceId",
		OwnerID:     "ownerId",
		Name:        "workspace",
	}

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(workspace, nil)
	participantRepoMock.EXPECT().GetWorkspaceParticipants("workspaceId").Return([]*domain.Participant{}, nil)
	participantRepoMock.EXPECT().AddWorkspaceParticipant(gomock.Any()).Return(nil)

	participant, err := service.AddParticipant("ownerId", "workspaceId", "Gleb", "gleb@itmo.com")

	if err != nil {
		t.Fatalf("Unexpected error occurred, got %v", err)
	}
	if participant == nil {
		t.Fatal("Expected participant to be returned, got nil")
	}
	if participant.Email != "gleb@itmo.com" {
		t.Errorf("Expected email gleb@itmo.com, got %v", participant.Email)
	}
	if participant.Name != "Gleb" {
		t.Errorf("Expected name Gleb, got %v", participant.Name)
	}
}

func TestAddParticipant_WorkspaceNotFound(t *testing.T) {
	service, workspaceRepoMock, _ := InitialSetup(t)

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(nil, nil)
	participant, err := service.AddParticipant("ownerId", "workspaceId", "Gleb", "gleb@itmo.com")

	if !errors.Is(err, ErrWorkspaceNotFound) {
		t.Errorf("Expected ErrWorkspaceNotFound, got %v", err)
	}
	if participant != nil {
		t.Errorf("Expected nil participant, got %v", participant)
	}
}

func TestAddParticipant_UserHasNoRights(t *testing.T) {
	service, workspaceRepoMock, _ := InitialSetup(t)

	workspace := &domain.Workspace{
		WorkspaceID: "workspaceId",
		OwnerID:     "ownerId",
	}

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(workspace, nil)
	participant, err := service.AddParticipant("guestId", "workspaceId", "Gleb", "gleb@itmo.com")

	if !errors.Is(err, ErrCreateUserForbidden) {
		t.Errorf("Expected ErrCreateUserForbidden, got %v", err)
	}
	if participant != nil {
		t.Errorf("Expected nil participant, got %v", participant)
	}
}

func TestAddParticipant_EmailNotUnique(t *testing.T) {
	service, workspaceRepoMock, participantRepoMock := InitialSetup(t)

	workspace := &domain.Workspace{
		WorkspaceID: "workspaceId",
		OwnerID:     "ownerId",
	}

	participant := &domain.Participant{
		Email: "gleb@itmo.com",
	}

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(workspace, nil)
	participantRepoMock.EXPECT().GetWorkspaceParticipants("workspaceId").Return([]*domain.Participant{participant}, nil)

	participant, err := service.AddParticipant("ownerId", "workspaceId", "Veronika", "gleb@itmo.com")

	if !errors.Is(err, ErrEmailNotUnique) {
		t.Errorf("Expected ErrEmailNotUnique, got %v", err)
	}
	if participant != nil {
		t.Errorf("Expected nil participant, got %v", participant)
	}
}

func TestAddParticipant_InvalidEmail(t *testing.T) {
	service, workspaceRepoMock, _ := InitialSetup(t)

	workspace := &domain.Workspace{
		WorkspaceID: "workspaceId",
		OwnerID:     "ownerId",
	}

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(workspace, nil)

	participant, err := service.AddParticipant("ownerId", "workspaceId", "Gleb", "invalid_email")

	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("Expected ErrInvalidEmail, got %v", err)
	}
	if participant != nil {
		t.Errorf("Expected nil participant, got %v", participant)
	}
}

func TestAddParticipant_InvalidUserName(t *testing.T) {
	service, workspaceRepoMock, _ := InitialSetup(t)

	workspace := &domain.Workspace{
		WorkspaceID: "workspaceId",
		OwnerID:     "ownerId",
	}

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(workspace, nil)

	participant, err := service.AddParticipant("ownerId", "workspaceId", "", "gleb@itmo.com")

	if !errors.Is(err, ErrInvalidUserName) {
		t.Errorf("Expected ErrInvalidUserName, got %v", err)
	}
	if participant != nil {
		t.Errorf("Expected nil participant, got %v", participant)
	}
}

func TestGetParticipants_Success(t *testing.T) {
	service, workspaceRepoMock, participantRepoMock := InitialSetup(t)

	workspace := &domain.Workspace{
		WorkspaceID: "workspaceId",
		OwnerID:     "ownerId",
	}

	participants := []*domain.Participant{
		{ParticipantID: "participant1"},
		{ParticipantID: "participant2"},
	}

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(workspace, nil)
	participantRepoMock.EXPECT().GetWorkspaceParticipants("workspaceId").Return(participants, nil)

	result, err := service.GetParticipants("ownerId", "workspaceId")

	if err != nil {
		t.Fatalf("Unexpected error occurred, got %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(result))
	}
	if result[0].ParticipantID != "participant1" || result[1].ParticipantID != "participant2" {
		t.Errorf("Unexpected participants id, expected: \"participant1\", \"participant2\", got %v, %v", result[0].ParticipantID, result[1].ParticipantID)
	}
}

func TestGetParticipants_NoRights(t *testing.T) {
	service, workspaceRepoMock, participantRepoMock := InitialSetup(t)

	workspace := &domain.Workspace{
		WorkspaceID: "workspaceId",
		OwnerID:     "ownerId",
	}

	participants := []*domain.Participant{
		{ParticipantID: "participant"},
	}

	workspaceRepoMock.EXPECT().FindWorkspace("workspaceId").Return(workspace, nil)
	participantRepoMock.EXPECT().GetWorkspaceParticipants("workspaceId").Return(participants, nil)

	result, err := service.GetParticipants("guestId", "workspaceId")

	if !errors.Is(err, ErrViewParticipantsForbidden) {
		t.Errorf("Expected ErrViewParticipantsForbidden, got %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}
