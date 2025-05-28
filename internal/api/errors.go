package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorType определяет типы ошибок API
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeConflict     ErrorType = "conflict"
	ErrorTypeExternal     ErrorType = "external_service"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
)

// ErrorResponse стандартный формат ошибки API
type ErrorResponse struct {
	Type    ErrorType `json:"type"`              // Тип ошибки
	Code    int       `json:"code"`              // HTTP статус код
	Message string    `json:"message"`           // Человекочитаемое сообщение
	Details any       `json:"details,omitempty"` // Детали ошибки
}

// Error делает ErrorResponse реализацией error интерфейса
func (e ErrorResponse) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewError создает новую ошибку API
func NewError(errorType ErrorType, code int, message string, details any) ErrorResponse {
	return ErrorResponse{
		Type:    errorType,
		Code:    code,
		Message: message,
		Details: details,
	}
}

// ValidationErrorDetails детали ошибки валидации
type ValidationErrorDetails struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

func HandleValidationError(c *gin.Context, errs validator.ValidationErrors) {
	details := make([]ValidationErrorDetails, len(errs))
	for i, err := range errs {
		details[i] = ValidationErrorDetails{
			Field:   err.Field(),
			Message: validationMessage(err),
			Value:   err.Value(),
		}
	}

	c.JSON(http.StatusBadRequest, NewError(
		ErrorTypeValidation,
		http.StatusBadRequest,
		"Validation failed",
		details,
	))
}

// HandleError обрабатывает ошибки в Gin обработчиках
func HandleError(c *gin.Context, err error) {
	switch e := err.(type) {
	case ErrorResponse:
		c.JSON(e.Code, e)
	case validator.ValidationErrors:
		HandleValidationError(c, e)
	case *gin.Error:
		HandleError(c, e.Err)
	default:
		c.JSON(http.StatusInternalServerError, NewError(
			ErrorTypeInternal,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		))
	}
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", err.Field(), err.Param())
	default:
		return fmt.Sprintf("%s is invalid (%s)", err.Field(), err.Tag())
	}
}

// Предопределенные ошибки
var (
	ErrNotFound = NewError(
		ErrorTypeNotFound,
		http.StatusNotFound,
		"Resource not found",
		nil,
	)

	ErrDBOperation = NewError(
		ErrorTypeInternal,
		http.StatusInternalServerError,
		"Database operation failed",
		nil,
	)

	ErrExternalAPI = NewError(
		ErrorTypeExternal,
		http.StatusBadGateway,
		"External service unavailable",
		nil,
	)
)
