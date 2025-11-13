package auth

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/args"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/settings"
	"os"
)

type App struct {
}

func (a App) Register() error {
	// Register auth models with GORM
	db.UseModel(User{})
	db.UseModel(UserLoginHistory{})

	// Set user interface for Evo framework
	evo.SetUserInterface(&User{})

	// Check for admin user creation command
	if args.Exists("--create-admin") {
		CreateAdminUser()
		os.Exit(0)
	}

	// Initialize JWT secret after settings are loaded
	InitializeJWTSecret()

	return nil
}

func (a App) Router() error {
	var controller Controller

	// Authentication endpoints
	evo.Post("/api/auth/login", controller.LoginHandler)
	evo.Post("/api/auth/refresh", controller.RefreshHandler)

	// Profile endpoints
	evo.Get("/api/auth/profile", controller.GetProfile)
	evo.Put("/api/auth/profile", controller.EditProfile)

	// API Key management endpoints
	evo.Post("/api/auth/api-key", controller.GenerateAPIKey)
	evo.Delete("/api/auth/api-key", controller.RevokeAPIKey)

	// OAuth endpoints
	if settings.Get("OAUTH.MICROSOFT.ENABLED").Bool() {
		evo.Get("/api/auth/oauth/microsoft", controller.MicrosoftOAuthLogin)
		evo.Get("/api/auth/oauth/microsoft/callback", controller.MicrosoftOAuthCallback)
	}
	if settings.Get("OAUTH.GOOGLE.ENABLED").Bool() {
		evo.Get("/api/auth/oauth/google", controller.GoogleOAuthLogin)
		evo.Get("/api/auth/oauth/google/callback", controller.GoogleOAuthCallback)
	}
	evo.Get("/api/auth/oauth/providers", controller.GetOAuthProviders)

	return nil
}

func (a App) WhenReady() error {
	// Initialize OAuth configurations
	InitOAuthConfigs()
	return nil
}

func (a App) Name() string {
	return "auth"
}
