package task

import "errors"

var (
	ErrNotFound       = errors.New("task not found")
	ErrAlreadyExists  = errors.New("task already exists")
	ErrInvalidStatus  = errors.New("invalid task status")
	ErrInvalidType    = errors.New("invalid task type")
	ErrInvalidPayload = errors.New("invalid task payload")
	ErrCancelled      = errors.New("task cancelled")
	ErrTimeout        = errors.New("task timeout")
)
