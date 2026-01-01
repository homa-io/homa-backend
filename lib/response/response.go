package response

import (
	"encoding/json"
	"fmt"
	"github.com/getevo/evo/v2/lib/text"
	"net/http"

	"github.com/getevo/evo/v2/lib/outcome"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Authentication & Authorization errors
	ErrorCodeUnauthorized ErrorCode = "unauthorized"
	ErrorCodeForbidden    ErrorCode = "forbidden"
	ErrorCodeInvalidToken ErrorCode = "invalid_token"

	// Input validation errors
	ErrorCodeInvalidInput        ErrorCode = "invalid_input"
	ErrorCodeInvalidConversationID ErrorCode = "invalid_conversation_id"
	ErrorCodeInvalidUserID       ErrorCode = "invalid_user_id"
	ErrorCodeMissingRequired     ErrorCode = "missing_required"

	// Resource errors
	ErrorCodeNotFound           ErrorCode = "not_found"
	ErrorCodeConversationNotFound ErrorCode = "conversation_not_found"
	ErrorCodeUserNotFound       ErrorCode = "user_not_found"
	ErrorCodeTagNotFound        ErrorCode = "tag_not_found"

	// Permission errors
	ErrorCodeAccessDenied            ErrorCode = "access_denied"
	ErrorCodeInsufficientPermissions ErrorCode = "insufficient_permissions"

	// Internal errors
	ErrorCodeInternalError   ErrorCode = "internal_error"
	ErrorCodeDatabaseError   ErrorCode = "database_error"
	ErrorCodeValidationError ErrorCode = "validation_error"
	ErrorCodeConflict        ErrorCode = "conflict"
)

// AppError represents a structured application error
type AppError struct {
	Code       ErrorCode `json:"error"`
	Message    string    `json:"message"`
	StatusCode int       `json:"-"`
	Details    string    `json:"details,omitempty"`
}

// Error implements the error interface
func (e AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Response returns an outcome.Response for the error
func (e AppError) Response() outcome.Response {
	return outcome.Response{
		StatusCode: e.StatusCode,
		Data: text.ToJSON(map[string]interface{}{
			"error":   string(e.Code),
			"message": e.Message,
		}),
	}
}

// NewError creates a new AppError
func NewError(code ErrorCode, message string, statusCode int) AppError {
	return AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewErrorWithDetails creates a new AppError with additional details
func NewErrorWithDetails(code ErrorCode, message string, statusCode int, details string) AppError {
	return AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    details,
	}
}

// Predefined common errors
var (
	// Authentication errors
	ErrUnauthorized = AppError{
		Code:       ErrorCodeUnauthorized,
		Message:    "Authentication required",
		StatusCode: http.StatusUnauthorized,
	}

	ErrForbidden = AppError{
		Code:       ErrorCodeForbidden,
		Message:    "Only agents and administrators can access this endpoint",
		StatusCode: http.StatusForbidden,
	}

	ErrInvalidToken = AppError{
		Code:       ErrorCodeInvalidToken,
		Message:    "Invalid or expired token",
		StatusCode: http.StatusUnauthorized,
	}

	// Input validation errors
	ErrInvalidInput = AppError{
		Code:       ErrorCodeInvalidInput,
		Message:    "Invalid request data",
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidConversationID = AppError{
		Code:       ErrorCodeInvalidConversationID,
		Message:    "Invalid conversation ID",
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidUserID = AppError{
		Code:       ErrorCodeInvalidUserID,
		Message:    "Invalid user ID format",
		StatusCode: http.StatusBadRequest,
	}

	ErrMissingRequired = AppError{
		Code:       ErrorCodeMissingRequired,
		Message:    "Missing required fields",
		StatusCode: http.StatusBadRequest,
	}

	// Resource errors
	ErrConversationNotFound = AppError{
		Code:       ErrorCodeConversationNotFound,
		Message:    "Conversation not found or access denied",
		StatusCode: http.StatusNotFound,
	}

	ErrUserNotFound = AppError{
		Code:       ErrorCodeUserNotFound,
		Message:    "User not found",
		StatusCode: http.StatusNotFound,
	}

	ErrTagNotFound = AppError{
		Code:       ErrorCodeTagNotFound,
		Message:    "Tag not found",
		StatusCode: http.StatusNotFound,
	}

	ErrNotFound = AppError{
		Code:       ErrorCodeNotFound,
		Message:    "Resource not found",
		StatusCode: http.StatusNotFound,
	}

	// Permission errors
	ErrAccessDenied = AppError{
		Code:       ErrorCodeAccessDenied,
		Message:    "Access denied to this resource",
		StatusCode: http.StatusForbidden,
	}

	// Internal errors
	ErrInternalError = AppError{
		Code:       ErrorCodeInternalError,
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
	}

	ErrDatabaseError = AppError{
		Code:       ErrorCodeDatabaseError,
		Message:    "Database operation failed",
		StatusCode: http.StatusInternalServerError,
	}
)

// Specific error constructors for common scenarios
func ErrUserDepartments() AppError {
	return NewError(ErrorCodeInternalError, "Failed to get user departments", http.StatusInternalServerError)
}

func ErrFetchConversations() AppError {
	return NewError(ErrorCodeInternalError, "Failed to fetch conversations", http.StatusInternalServerError)
}

func ErrCountConversations() AppError {
	return NewError(ErrorCodeInternalError, "Failed to count conversations", http.StatusInternalServerError)
}

func ErrUpdateConversationStatus() AppError {
	return NewError(ErrorCodeInternalError, "Failed to update conversation status", http.StatusInternalServerError)
}

func ErrCreateMessage() AppError {
	return NewError(ErrorCodeInternalError, "Failed to create message", http.StatusInternalServerError)
}

func ErrAssignConversation() AppError {
	return NewError(ErrorCodeInternalError, "Failed to assign conversation", http.StatusInternalServerError)
}

func ErrUpdateConversationDepartment() AppError {
	return NewError(ErrorCodeInternalError, "Failed to update conversation department", http.StatusInternalServerError)
}

func ErrCreateTag() AppError {
	return NewError(ErrorCodeInternalError, "Failed to create tag", http.StatusInternalServerError)
}

func ErrCreateTagWithName(tagName string) AppError {
	return NewErrorWithDetails(
		ErrorCodeInternalError,
		"Failed to create tag",
		http.StatusInternalServerError,
		fmt.Sprintf("Tag name: %s", tagName),
	)
}

func ErrMissingTagsOrIDs() AppError {
	return NewError(ErrorCodeInvalidInput, "Either tag_ids or tag_names must be provided", http.StatusBadRequest)
}

func ErrMissingUserOrDepartment() AppError {
	return NewError(ErrorCodeInvalidInput, "Either user_id or department_id must be provided", http.StatusBadRequest)
}

// Helper function to create outcome.Response from AppError
func Error(err AppError) outcome.Response {
	return err.Response()
}

// =====================================================
// STANDARDIZED SUCCESS RESPONSE SYSTEM
// =====================================================

// APIResponse represents a standardized API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Message string      `json:"message,omitempty"`
}

func (r APIResponse) ToJSON() []byte {
	b, _ := json.Marshal(r)
	return b
}

// Meta contains metadata for API responses
type Meta struct {
	// Pagination
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`

	// List/Collection metadata
	Count  int `json:"count,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Custom metadata
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// OK creates a standardized success response
func OK(data interface{}) outcome.Response {
	return outcome.Response{
		ContentType: "application/json",
		StatusCode:  http.StatusOK,
		Data: APIResponse{
			Success: true,
			Data:    data,
		}.ToJSON(),
	}
}

// OKWithMessage creates a success response with a message
func OKWithMessage(data interface{}, message string) outcome.Response {
	return outcome.Response{
		StatusCode: http.StatusOK,
		Data: APIResponse{
			Success: true,
			Data:    data,
			Message: message,
		}.ToJSON(),
	}
}

// OKWithMeta creates a success response with metadata
func OKWithMeta(data interface{}, meta *Meta) outcome.Response {
	return outcome.Response{
		StatusCode: http.StatusOK,
		Data: APIResponse{
			Success: true,
			Data:    data,
			Meta:    meta,
		}.ToJSON(),
	}
}

// OKWithMessageAndMeta creates a success response with both message and metadata
func OKWithMessageAndMeta(data interface{}, message string, meta *Meta) outcome.Response {
	return outcome.Response{
		StatusCode: http.StatusOK,
		Data: APIResponse{
			Success: true,
			Data:    data,
			Message: message,
			Meta:    meta,
		}.ToJSON(),
	}
}

// Created creates a 201 Created response
func Created(data interface{}) outcome.Response {
	return outcome.Response{
		StatusCode: http.StatusCreated,
		Data: APIResponse{
			Success: true,
			Data:    data,
		}.ToJSON(),
	}
}

// CreatedWithMessage creates a 201 Created response with message
func CreatedWithMessage(data interface{}, message string) outcome.Response {
	return outcome.Response{
		StatusCode: http.StatusCreated,
		Data: APIResponse{
			Success: true,
			Data:    data,
		}.ToJSON(),
	}
}

// Paginated creates a paginated response
func Paginated(data interface{}, page, limit int, total int64) outcome.Response {
	totalPages := int((total + int64(limit) - 1) / int64(limit)) // Ceiling division
	meta := &Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return OKWithMeta(data, meta)
}

// List creates a response for lists/collections with count
func List(data interface{}, count int) outcome.Response {
	meta := &Meta{
		Count: count,
	}

	return OKWithMeta(data, meta)
}

// ListWithTotal creates a response for lists/collections with count and total
func ListWithTotal(data interface{}, total int) outcome.Response {
	meta := &Meta{
		Count: total,
		Total: int64(total),
	}

	return OKWithMeta(data, meta)
}

// Message creates a response with only a success message
func Message(message string) outcome.Response {
	return outcome.Response{
		StatusCode: http.StatusOK,
		Data: APIResponse{
			Success: true,
			Message: message,
		},
	}
}

// Unauthorized creates a 401 Unauthorized response
func Unauthorized(c interface{}, message string) outcome.Response {
	return Error(NewError(ErrorCodeUnauthorized, message, http.StatusUnauthorized))
}

// Forbidden creates a 403 Forbidden response
func Forbidden(c interface{}, message string) outcome.Response {
	return Error(NewError(ErrorCodeForbidden, message, http.StatusForbidden))
}

// BadRequest creates a 400 Bad Request response
func BadRequest(c interface{}, message string) outcome.Response {
	return Error(NewError(ErrorCodeInvalidInput, message, http.StatusBadRequest))
}

// NotFound creates a 404 Not Found response
func NotFound(c interface{}, message string) outcome.Response {
	return Error(NewError(ErrorCodeNotFound, message, http.StatusNotFound))
}

// InternalError creates a 500 Internal Server Error response
func InternalError(c interface{}, message string) outcome.Response {
	return Error(NewError(ErrorCodeInternalError, message, http.StatusInternalServerError))
}
