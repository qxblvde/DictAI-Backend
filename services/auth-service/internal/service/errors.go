package service

import "errors"

var (
	ErrInvalidPassword    = errors.New("invalid password")
	ErrWrongPassword      = errors.New("wrong password")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
)
