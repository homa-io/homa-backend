package admin

import (
	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/lib/response"
)

// AdminAuthMiddleware ensures the user is logged in and is an administrator
func AdminAuthMiddleware(request *evo.Request) error {
	// Check if user is anonymous (not logged in) first
	if request.User().Anonymous() {
		return response.ErrUnauthorized
	}

	// Get user from request context with proper type assertion
	var user = request.User().Interface().(*auth.User)

	// Check if user is administrator
	if user.Type != auth.UserTypeAdministrator {
		return response.ErrForbidden
	}

	return request.Next()
}
