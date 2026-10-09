package domain

import (
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrNotFound           = errors.New("not found")

	ErrForbidden    = errors.New("forbidden")
	ErrNameConflict = errors.New("name already exists")
	ErrNotFolder    = errors.New("not a folder")
	ErrNotFile      = errors.New("not a file")
	ErrInvalidMove  = errors.New("cannot move object into itself or its descendant")
	ErrReadOnly     = errors.New("storage is in readonly mode")
)
