package service

import "errors"

var (
	ErrConflict     = errors.New("resource conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrValidation   = errors.New("validation error")
	ErrNotFound     = errors.New("not found")
	ErrInsufficient = errors.New("insufficient funds")
)
