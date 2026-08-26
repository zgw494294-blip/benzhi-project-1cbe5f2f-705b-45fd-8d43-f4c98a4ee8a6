package domain

import "fmt"

type ErrorCode string

const (
	ErrInvalid  ErrorCode = "invalid"
	ErrConflict ErrorCode = "conflict"
	ErrState    ErrorCode = "state"
	ErrNotFound ErrorCode = "not_found"
)

type DomainError struct {
	Code    ErrorCode
	Message string
}

func (e *DomainError) Error() string { return e.Message }

func Invalid(format string, args ...any) error {
	return &DomainError{Code: ErrInvalid, Message: fmt.Sprintf(format, args...)}
}

func StateError(format string, args ...any) error {
	return &DomainError{Code: ErrState, Message: fmt.Sprintf(format, args...)}
}

func NotFound(format string, args ...any) error {
	return &DomainError{Code: ErrNotFound, Message: fmt.Sprintf(format, args...)}
}
