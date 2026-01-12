package jobs

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	homadb "github.com/iesreza/homa-backend/apps/nats"
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
	var admin = evo.Group("/api/admin/jobs")
	admin.Get("/", GetJobs)
	admin.Get("/executions", GetJobExecutions)
	admin.Post("/:name/run", RunJob)
	return nil
}

// WhenReady initializes the scheduler after all apps are ready
func (App) WhenReady() error {
	// Check if jobs are enabled
	if !settings.Get("JOBS.ENABLED", true).Bool() {
		log.Info("Jobs are disabled, skipping scheduler initialization")
		return nil
	}

	// Get JetStream context
	js := homadb.GetJetStream()
	if js == nil {
		log.Warning("JetStream not available, jobs will not run")
		return nil
	}

	// Create lock manager
	locks, err := NewLockManager(js)
	if err != nil {
		log.Error("Failed to create lock manager: %v", err)
		return err
	}

	// Create scheduler
	scheduler := NewScheduler(locks)

	// Register all jobs
	RegisterAllJobs(scheduler)

	// Start the scheduler
	scheduler.Start()

	log.Info("Jobs app ready - scheduler running")
	return nil
}

// Shutdown gracefully stops the scheduler
func (App) Shutdown() error {
	log.Info("Shutting down Jobs app...")

	if scheduler != nil {
		scheduler.Stop()
	}

	return nil
}

// Name returns the app name
func (App) Name() string {
	return "jobs"
}
