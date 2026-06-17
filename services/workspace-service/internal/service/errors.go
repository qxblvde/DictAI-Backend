package service

import "errors"

var (
	ErrWorkspaceNotFound             = errors.New("workspace not found")
	ErrCreateUserForbidden           = errors.New("non-owners of workspaces cannot add users")
	ErrViewParticipantsForbidden     = errors.New("person not associated with the workspace cannot view the list of participants")
	ErrDeleteParticipantForbidden    = errors.New("only workspace owner can remove participants")
	ErrUpdateParticipantForbidden    = errors.New("only workspace owner can update participants")
	ErrDeleteWorkspaceForbidden      = errors.New("only workspace owner can delete workspace")
	ErrUpdateWorkspaceForbidden      = errors.New("only workspace owner can update workspace")
	ErrParticipantNotFound           = errors.New("participant not found in workspace")
	ErrEmailNotUnique                = errors.New("user with this email already exists in the workspace")
	ErrInvalidEmail                  = errors.New("invalid email")
	ErrInvalidWorkspaceName          = errors.New("invalid workspace name")
	ErrInvalidUserName               = errors.New("invalid user name")
)
