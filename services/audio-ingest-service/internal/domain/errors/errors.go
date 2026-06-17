package errors

import "errors"

var ErrInvalidInput = errors.New("invalid input")
var ErrForbidden = errors.New("workspace access denied")
