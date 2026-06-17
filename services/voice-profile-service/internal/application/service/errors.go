package service

import "errors"

var (
	ErrParticipantNotFound = errors.New("participant not found")
	ErrProfileNotFound     = errors.New("voice profile not found")
	ErrForbidden           = errors.New("voice profile access denied")
)
