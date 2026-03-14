package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrInternal     = errors.New("internal error")
	ErrInvalidInput = errors.New("invalid input")
)

type AppError struct {
	Code    error
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %v", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Code }

func New(code error, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code error, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}

func Is(err, target error) bool { return errors.Is(err, target) }
