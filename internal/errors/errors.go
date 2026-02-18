package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// APIError is the standard production error wrapper
type APIError struct {
	Status  int               `json:"-"`       // Internal HTTP status
	Message string            `json:"message"` // User-friendly message
	Details map[string]string `json:"errors,omitempty"` // For 422 field errors
	Internal error            `json:"-"`       // For logging only, never expose to client
}

func (e *APIError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// Helper Constructors
func New(status int, message string, internal error) *APIError {
	return &APIError{
		Status:   status,
		Message:  message,
		Internal: internal,
	}
}

// NewValidationError converts go-playground errors to a map
func NewValidationError(err error) *APIError {
	details := make(map[string]string)
	
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, f := range ve {
			details[f.Field()] = fmt.Sprintf("Field validation for '%s' failed on the '%s' tag", f.Field(), f.Tag())
		}
	}

	return &APIError{
		Status:  http.StatusUnprocessableEntity,
		Message: "Validation failed",
		Details: details,
		Internal: err,
	}
}

// Common error helpers
func BadRequest(msg string, err error) *APIError { return New(http.StatusBadRequest, msg, err) }
func Unauthorized(msg string, err error) *APIError { return New(http.StatusUnauthorized, msg, err) }
func Forbidden(msg string, err error) *APIError { return New(http.StatusForbidden, msg, err) }
func NotFound(msg string, err error) *APIError { return New(http.StatusNotFound, msg, err) }
func UnprocessableEntity(msg string, err error) *APIError { return New(http.StatusUnprocessableEntity, msg, err) }
func Conflict(msg string, err error) *APIError { return New(http.StatusConflict, msg, err) }
func Internal(err error) *APIError { return New(http.StatusInternalServerError, "An unexpected error occurred", err) }