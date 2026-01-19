package redis

import (
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/log"
)

// App represents the Redis application module
type App struct{}

// Register initializes the Redis module
func (App) Register() error {
	log.Info("Registering Redis app...")
	return nil
}

// Router registers HTTP routes (none for Redis)
func (App) Router() error {
	return nil
}

// WhenReady connects to Redis after application is fully initialized
func (App) WhenReady() error {
	log.Info("Initializing Redis connection...")

	// Connect to Redis
	if err := Initialize(); err != nil {
		log.Error("Failed to connect to Redis: %v", err)
		return err
	}

	// Load rate limit settings from database
	LoadRateLimitSettings()

	// Subscribe to rate limit reload events
	SubscribeToRateLimitReload()

	log.Info("Redis app ready")
	return nil
}

// Name returns the app name
func (App) Name() string {
	return "redis"
}

// Shutdown gracefully closes the Redis connection
func (App) Shutdown() error {
	log.Info("Shutting down Redis connection...")
	return Close()
}

// GetInstance returns the singleton Redis app instance
func GetInstance() *App {
	return &App{}
}

var _ application.Application = (*App)(nil)
