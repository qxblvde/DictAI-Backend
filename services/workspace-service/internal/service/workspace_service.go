package service

import (
	"database/sql"
	"errors"
	"net/mail"

	"github.com/DictAI/Microservices/services/workspace-service/internal/domain"
)
type WorkspaceService struct {
	rpWorkspaces   domain.WorkspacesRepository
	rpParticipants domain.ParticipantRepository
}

func NewWorkspaceService(rpWorkspaces domain.WorkspacesRepository, rpParticipants domain.ParticipantRepository) *WorkspaceService {
	return &WorkspaceService{rpWorkspaces: rpWorkspaces, rpParticipants: rpParticipants}
}
func (svc *WorkspaceService) UpdateParticipant(userID, workspaceID, participantID, name, email string) (*domain.Participant, error) {
	workspace, err := svc.rpWorkspaces.FindWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, ErrWorkspaceNotFound
	}
	if workspace.OwnerID != userID {
		return nil, ErrUpdateParticipantForbidden
	}

	if name == "" && email == "" {
		return nil, ErrInvalidUserName
	}
	if name != "" {
		if err := svc.isUserNameValid(name); err != nil {
			return nil, err
		}
	}
	if email != "" {
		if err := svc.isEmailValid(email); err != nil {
			return nil, err
		}
		// Проверяем уникальность email (исключая самого участника)
		participants, err := svc.rpParticipants.GetWorkspaceParticipants(workspaceID)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			if p.Email == email && p.ParticipantID != participantID {
				return nil, ErrEmailNotUnique
			}
		}
	}

	p, err := svc.rpParticipants.UpdateParticipant(workspaceID, participantID, name, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrParticipantNotFound
		}
		return nil, err
	}
	return p, nil
}

func (svc *WorkspaceService) UpdateWorkspace(userID, workspaceID, name string) (*domain.Workspace, error) {
	workspace, err := svc.rpWorkspaces.FindWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, ErrWorkspaceNotFound
	}
	if workspace.OwnerID != userID {
		return nil, ErrUpdateWorkspaceForbidden
	}
	if err := svc.isWorkspaceNameValid(name); err != nil {
		return nil, err
	}
	return svc.rpWorkspaces.UpdateWorkspace(workspaceID, name)
}

func (svc *WorkspaceService) DeleteWorkspace(userID, workspaceID string) error {
	workspace, err := svc.rpWorkspaces.FindWorkspace(workspaceID)
	if err != nil {
		return err
	}
	if workspace == nil {
		return ErrWorkspaceNotFound
	}
	if workspace.OwnerID != userID {
		return ErrDeleteWorkspaceForbidden
	}
	if err := svc.rpWorkspaces.DeleteWorkspace(workspaceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrWorkspaceNotFound
		}
		return err
	}
	return nil
}

func (svc *WorkspaceService) DeleteParticipant(userID, workspaceID, participantID string) error {
	workspace, err := svc.rpWorkspaces.FindWorkspace(workspaceID)
	if err != nil {
		return err
	}
	if workspace == nil {
		return ErrWorkspaceNotFound
	}
	if workspace.OwnerID != userID {
		return ErrDeleteParticipantForbidden
	}

	if err := svc.rpParticipants.DeleteParticipant(workspaceID, participantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrParticipantNotFound
		}
		return err
	}
	return nil
}

func (svc *WorkspaceService) GetWorkspacesByUser(userID string, page, limit int) ([]*domain.Workspace, int, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	workspaces, total, err := svc.rpWorkspaces.GetByOwnerID(userID, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	return workspaces, total, totalPages, nil
}

func (svc *WorkspaceService) CreateWorkspace(userID, name string) (*domain.Workspace, error) {
	err := svc.isWorkspaceNameValid(name)
	if err != nil {
		return nil, err
	}

	newWorkspace := &domain.Workspace{
		OwnerID: userID,
		Name:    name,
	}

	err = svc.rpWorkspaces.CreateWorkspace(newWorkspace)
	if err != nil {
		return nil, err
	}

	return newWorkspace, nil
}

func (svc *WorkspaceService) AddParticipant(userID, workspaceID, name, email string) (*domain.Participant, error) {
	workspace, err := svc.rpWorkspaces.FindWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, ErrWorkspaceNotFound
	}
	if workspace.OwnerID != userID {
		return nil, ErrCreateUserForbidden
	}

	err = svc.isUserNameValid(name)
	if err != nil {
		return nil, err
	}
	err = svc.isEmailValid(email)
	if err != nil {
		return nil, err
	}

	participants, err := svc.rpParticipants.GetWorkspaceParticipants(workspaceID)
	if err != nil {
		return nil, err
	}
	for i := range participants {
		if participants[i].Email == email {
			return nil, ErrEmailNotUnique
		}
	}

	newParticipant := &domain.Participant{
		WorkspaceID: workspaceID,
		Name:        name,
		Email:       email,
	}

	err = svc.rpParticipants.AddWorkspaceParticipant(newParticipant)
	if err != nil {
		return nil, err
	}

	return newParticipant, nil
}

func (svc *WorkspaceService) GetWorkspace(workspaceID string) (*domain.Workspace, error) {
	workspace, err := svc.rpWorkspaces.FindWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, ErrWorkspaceNotFound
	}
	return workspace, nil
}

func (svc *WorkspaceService) GetParticipants(userID, workspaceID string) ([]*domain.Participant, error) {
	workspace, err := svc.rpWorkspaces.FindWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, ErrWorkspaceNotFound
	}

	participants, err := svc.rpParticipants.GetWorkspaceParticipants(workspaceID)
	if err != nil {
		return nil, err
	}

	if workspace.OwnerID == userID {
		return participants, nil
	}
	for i := range participants {
		if participants[i].ParticipantID == userID {
			return participants, nil
		}
	}

	return nil, ErrViewParticipantsForbidden
}

func (svc *WorkspaceService) isWorkspaceNameValid(name string) error {
	if name == "" {
		return ErrInvalidWorkspaceName
	}

	return nil
}

func (svc *WorkspaceService) isEmailValid(email string) error {
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		return ErrInvalidEmail
	}

	return nil
}

func (svc *WorkspaceService) isUserNameValid(name string) error {
	if name == "" {
		return ErrInvalidUserName
	}

	return nil
}
