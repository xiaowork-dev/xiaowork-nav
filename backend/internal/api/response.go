package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/service"
)

// Stable, machine-readable error codes. See docs/api-spec.md appendix A.
const (
	CodeValidationFailed     = "validation_failed"
	CodeInvalidCredentials   = "invalid_credentials"
	CodeUnauthorized         = "unauthorized"
	CodeForbidden            = "forbidden"
	CodeCSRFFailed           = "csrf_failed"
	CodeNotFound             = "not_found"
	CodeAlreadyInitialized   = "already_initialized"
	CodeUsernameTaken        = "username_taken"
	CodeImportConflict       = "import_conflict"
	CodePayloadTooLarge      = "payload_too_large"
	CodeInvalidURLScheme     = "invalid_url_scheme"
	CodeUnsupportedMediaType = "unsupported_media_type"
	CodeRateLimited          = "rate_limited"
	CodeInternalError        = "internal_error"
)

// errorEnvelope is the wire format for every error response.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Details   []errorField `json:"details,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
}

type errorField struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

// APIError is a typed error carrying an HTTP status and a stable code.
type APIError struct {
	Status  int
	Code    string
	Message string
	Details []errorField
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAPIError builds an APIError.
func NewAPIError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// ValidationError builds a 400 error with field details.
func ValidationError(message string, details ...errorField) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: CodeValidationFailed, Message: message, Details: details}
}

// FieldDetail is a convenience constructor for a field-level issue.
func FieldDetail(field, issue string) errorField {
	return errorField{Field: field, Issue: issue}
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The response is already partially written; nothing useful to do.
		return
	}
}

// writeError writes a standard error envelope.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := &APIError{
		Status:  http.StatusInternalServerError,
		Code:    CodeInternalError,
		Message: "服务器内部错误",
	}

	var typed *APIError
	if errors.As(err, &typed) {
		apiErr = typed
	} else {
		var svc *service.Error
		if errors.As(err, &svc) {
			apiErr = &APIError{
				Status:  statusForKind(svc.Kind),
				Code:    svc.Code,
				Message: svc.Message,
				Details: toFieldDetails(svc.Details),
			}
		}
	}

	writeJSON(w, r, apiErr.Status, errorEnvelope{Error: errorBody{
		Code:      apiErr.Code,
		Message:   apiErr.Message,
		Details:   apiErr.Details,
		RequestID: requestIDFrom(r.Context()),
	}})
}

func statusForKind(kind service.Kind) int {
	switch kind {
	case service.KindValidation:
		return http.StatusBadRequest
	case service.KindNotFound:
		return http.StatusNotFound
	case service.KindConflict:
		return http.StatusConflict
	case service.KindUnauthorized:
		return http.StatusUnauthorized
	case service.KindForbidden:
		return http.StatusForbidden
	case service.KindUnsupported:
		return http.StatusUnprocessableEntity
	case service.KindTooLarge:
		return http.StatusRequestEntityTooLarge
	case service.KindRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func toFieldDetails(details []service.FieldError) []errorField {
	if len(details) == 0 {
		return nil
	}
	out := make([]errorField, 0, len(details))
	for _, d := range details {
		out = append(out, errorField{Field: d.Field, Issue: d.Issue})
	}
	return out
}

// notFound is the JSON 404 handler for unmatched API routes.
func notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, NewAPIError(http.StatusNotFound, CodeNotFound, "资源不存在"))
}
