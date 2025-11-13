package auth

import (
	"github.com/getevo/evo/v2/lib/settings"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

// OAuth configuration will be loaded from environment variables
var (
	GoogleOAuthConfig    *oauth2.Config
	MicrosoftOAuthConfig *oauth2.Config
)

// InitOAuthConfigs initializes OAuth configurations
func InitOAuthConfigs() {
	// Initialize Google OAuth config
	GoogleOAuthConfig = &oauth2.Config{
		ClientID:     settings.Get("OAUTH.GOOGLE.CLIENT_ID").String(), // To be set from environment/config
		ClientSecret: settings.Get("OAUTH.GOOGLE.SECRET").String(),    // To be set from environment/config
		RedirectURL:  settings.Get("APP.BASE_PATH", "http://localhost:8000").String() + "/api/auth/oauth/google/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// Initialize Microsoft OAuth config
	MicrosoftOAuthConfig = &oauth2.Config{
		ClientID:     settings.Get("OAUTH.MICROSOFT.CLIENT_ID").String(), // To be set from environment/config
		ClientSecret: settings.Get("OAUTH.MICROSOFT.SECRET").String(),    // To be set from environment/config
		RedirectURL:  settings.Get("APP.BASE_PATH", "http://localhost:8000").String() + "/api/auth/oauth/microsoft/callback",
		Scopes: []string{
			"https://graph.microsoft.com/user.read",
		},
		Endpoint: microsoft.AzureADEndpoint("common"),
	}
}
