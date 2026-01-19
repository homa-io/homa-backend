package jobs

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
)

// App represents the Jobs application module
type App struct{}

var _ application.Application = (*App)(nil)

// Register initializes the jobs module
func (App) Register() error {
	log.Info("Registering Jobs app...")

	// Register model for migration
	db.UseModel(JobExecution{})

	return nil
}

// Router registers HTTP routes for job management
func (App) Router() error {
	// Public endpoint to list jobs (no auth required)
	evo.Get("/api/v1/jobs", ListJobs)

	// Protected endpoints with API key authentication
	evo.Use("/api/v1/jobs/:name", APIKeyMiddleware)

	// Trigger a job
	evo.Post("/api/v1/jobs/:name", TriggerJob)

	// Get job status
	evo.Get("/api/v1/jobs/:name/status", GetJobStatus)

	// Get job execution history
	evo.Get("/api/v1/jobs/:name/history", GetJobHistory)

	// Job settings endpoints (admin protected via system middleware)
	evo.Get("/api/settings/jobs", GetJobSettings)
	evo.Put("/api/settings/jobs", UpdateJobSettings)

	return nil
}

// WhenReady initializes the job registry after all apps are ready
func (App) WhenReady() error {
	// Initialize API key from settings
	InitAPIKey()

	apiKey := GetAPIKey()
	if apiKey == "" {
		log.Warning("[jobs] No API key configured (JOBS.API_KEY). Job trigger endpoint will reject all requests.")
	} else {
		log.Info("[jobs] API key configured for job trigger endpoint")
	}

	// Initialize default job settings
	InitJobSettings()

	// Register all jobs
	RegisterAllJobs()

	log.Info("[jobs] Jobs app ready - %d jobs registered", GetRegistry().Count())
	return nil
}

// Shutdown gracefully stops the jobs module
func (App) Shutdown() error {
	log.Info("Shutting down Jobs app...")
	return nil
}

// Name returns the app name
func (App) Name() string {
	return "jobs"
}
