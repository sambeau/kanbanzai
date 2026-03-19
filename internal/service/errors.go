package service

import "errors"

// Sentinel errors for service-layer operations.
var (
	ErrNotFound          = errors.New("entity not found")
	ErrInvalidTransition = errors.New("invalid lifecycle transition")
	ErrValidationFailed  = errors.New("validation failed")
	ErrReferenceNotFound = errors.New("referenced entity not found")
	ErrImmutableField    = errors.New("immutable field cannot be changed")
)
