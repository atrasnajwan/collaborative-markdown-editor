package errors

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppError represents an application error
type AppError struct {
	Code    int    // HTTP status code
	Message string // Error message
	Err     error  // Original error
}

// Error returns the error message
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the original error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithMessage returns a copy of the AppError with a custom message
func (e *AppError) WithMessage(msg string) *AppError {
    return &AppError{
        Code:    e.Code,
        Message: msg,
        Err:     e.Err,
    }
}

// NewAppError creates a new application error
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error types
var (
	ErrInvalidInput      = func(err error) *AppError { return NewAppError(http.StatusBadRequest, "Invalid input", err) }
	ErrUnauthorized      = func(err error) *AppError { return NewAppError(http.StatusUnauthorized, "Unauthorized", err) }
	ErrNotFound          = func(err error) *AppError { return NewAppError(http.StatusNotFound, "Resource not found", err) }
	ErrInternalServer    = func(err error) *AppError { return NewAppError(http.StatusInternalServerError, "Internal server error", err) }
	ErrUnprocessableEntity = func(err error) *AppError { return NewAppError(http.StatusUnprocessableEntity, "Unprocessable entity", err) }
)

// HandleError handles an error and responds with the appropriate status code and message
func HandleError(c *gin.Context, err error) {
	var appErr *AppError
	log.Printf("%v\n", err.Error())
	if errors.As(err, &appErr) {
		c.JSON(appErr.Code, gin.H{"error": appErr.Message })
		return
	}
	
	// Default to internal server error
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
