// Package service implements the application's business rules. It depends on
// store and models but knows nothing about HTTP; the api package maps its
// typed errors onto HTTP status codes and the contract's error codes.
package service

// Kind classifies a service error so the API layer can choose a status code.
type Kind int

const (
	KindValidation Kind = iota
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
	KindUnsupported
	KindTooLarge
	KindRateLimited
	KindInternal
)

// FieldError is a field-level validation issue.
type FieldError struct {
	Field string
	Issue string
}

// Error is a typed service error carrying a stable API error code.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Details []FieldError
}

func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

// New builds a service error.
func New(kind Kind, code, message string, details ...FieldError) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Details: details}
}

// Validation builds a validation error.
func Validation(code, message string, details ...FieldError) *Error {
	return New(KindValidation, code, message, details...)
}

// NotFound builds a not-found error.
func NotFound(message string) *Error {
	return New(KindNotFound, "not_found", message)
}

// Conflict builds a conflict error.
func Conflict(code, message string) *Error {
	return New(KindConflict, code, message)
}

// Unauthorized builds an authentication error.
func Unauthorized(message string) *Error {
	return New(KindUnauthorized, "unauthorized", message)
}

// Forbidden builds a permission error.
func Forbidden(code, message string) *Error {
	return New(KindForbidden, code, message)
}
