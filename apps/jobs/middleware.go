package jobs

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/iesreza/homa-backend/lib/response"
)

var apiKey string

// InitAPIKey loads the API key from settings
func InitAPIKey() {
	apiKey = settings.Get("JOBS.API_KEY").String()
	if apiKey == "" {
		apiKey = settings.Get("JOBS_API_KEY").String() // Fallback
	}
}

// GetAPIKey returns the configured API key
func GetAPIKey() string {
	return apiKey
}

// APIKeyMiddleware validates the API key for job trigger endpoints
// Expected header format: Authorization: APIKEY <key>
func APIKeyMiddleware(req *evo.Request) error {
	// Get Authorization header
	authHeader := req.Header("Authorization")

	// Parse "APIKEY <key>" format
	var providedKey string
	if len(authHeader) > 7 && authHeader[:7] == "APIKEY " {
		providedKey = authHeader[7:]
	}

	// Check if API key is configured
	if apiKey == "" {
		req.WriteResponse(response.InternalError(nil, "Jobs API key not configured"))
		return nil
	}

	// Validate API key
	if providedKey != apiKey {
		req.WriteResponse(response.Unauthorized(nil, "Invalid or missing API key. Use header: Authorization: APIKEY <key>"))
		return nil
	}

	return req.Next()
}
