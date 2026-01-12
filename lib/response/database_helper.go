package response

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"gorm.io/gorm"
)

// HandleDBError handles common database errors with consistent responses
// Returns nil if no error, otherwise returns appropriate error response
func HandleDBError(err error, req *evo.Request, notFoundMsg string, context string) interface{} {
	if err == nil {
		return nil
	}

	// Log the error for debugging
	log.Error("%s: %v", context, err)

	if err == gorm.ErrRecordNotFound {
		return NotFound(req, notFoundMsg)
	}

	return Error(ErrInternalError)
}

// HandleDBErrorWithDetails handles DB errors with detailed error info
func HandleDBErrorWithDetails(err error, req *evo.Request, notFoundMsg string, context string) interface{} {
	if err == nil {
		return nil
	}

	// Log the error for debugging
	log.Error("%s: %v", context, err)

	if err == gorm.ErrRecordNotFound {
		return Error(NewErrorWithDetails(ErrorCodeNotFound, notFoundMsg, 404, "Resource not found"))
	}

	return Error(NewErrorWithDetails(ErrorCodeDatabaseError, "Database operation failed", 500, err.Error()))
}

// MustFind is a helper that returns error response if record not found
// Usage: if resp := MustFind(err, req, "User not found", "GetUser"); resp != nil { return resp }
func MustFind(err error, req *evo.Request, notFoundMsg string, context string) interface{} {
	return HandleDBError(err, req, notFoundMsg, context)
}

// MustFindWithLog wraps DB lookup with logging and error handling
func MustFindWithLog(err error, req *evo.Request, notFoundMsg string, context string, additionalInfo string) interface{} {
	if err == nil {
		return nil
	}

	log.Error("%s [%s]: %v", context, additionalInfo, err)

	if err == gorm.ErrRecordNotFound {
		return NotFound(req, notFoundMsg)
	}

	return Error(ErrInternalError)
}

// HandlePaginationError handles pagination-related errors
func HandlePaginationError(err error, req *evo.Request, context string) interface{} {
	if err == nil {
		return nil
	}

	log.Error("%s pagination error: %v", context, err)

	// For pagination, we return empty results instead of error in some cases
	if err == gorm.ErrRecordNotFound {
		return nil // Let caller handle empty result
	}

	return Error(ErrInternalError)
}
