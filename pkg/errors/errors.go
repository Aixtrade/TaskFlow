package errors

import (
	"errors"
	"fmt"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrTaskCancelled     = errors.New("task cancelled")
	ErrTaskFailed        = errors.New("task failed")
	ErrInvalidPayload    = errors.New("invalid payload")
	ErrInvalidTaskType   = errors.New("invalid task type")
	ErrInvalidTaskID     = errors.New("invalid task id")
	ErrInvalidQueue      = errors.New("invalid queue")
	ErrQueueFull         = errors.New("queue is full")
	ErrTimeout           = errors.New("operation timeout")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrRateLimited       = errors.New("rate limited")
)

type TaskError struct {
	TaskID  string
	Type    string
	Message string
	Cause   error
}

func (e *TaskError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("task %s (%s): %s: %v", e.TaskID, e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("task %s (%s): %s", e.TaskID, e.Type, e.Message)
}

func (e *TaskError) Unwrap() error {
	return e.Cause
}

func NewTaskError(taskID, taskType, message string, cause error) *TaskError {
	return &TaskError{
		TaskID:  taskID,
		Type:    taskType,
		Message: message,
		Cause:   cause,
	}
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

type RetryableError struct {
	Cause      error
	RetryAfter int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (retry after %ds): %v", e.RetryAfter, e.Cause)
}

func (e *RetryableError) Unwrap() error {
	return e.Cause
}

func NewRetryableError(cause error, retryAfter int) *RetryableError {
	return &RetryableError{
		Cause:      cause,
		RetryAfter: retryAfter,
	}
}

func IsRetryable(err error) bool {
	var retryErr *RetryableError
	return errors.As(err, &retryErr)
}

func IsTaskError(err error) bool {
	var taskErr *TaskError
	return errors.As(err, &taskErr)
}

func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}
