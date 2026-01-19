package jobs

import (
	"fmt"
	"sync"
)

// Registry manages job registration and lookup
type Registry struct {
	jobs map[string]*JobDefinition
	mu   sync.RWMutex
}

var (
	registry *Registry
	regOnce  sync.Once
)

// GetRegistry returns the singleton registry instance
func GetRegistry() *Registry {
	regOnce.Do(func() {
		registry = &Registry{
			jobs: make(map[string]*JobDefinition),
		}
	})
	return registry
}

// Register adds a job to the registry
func (r *Registry) Register(job JobDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if job.Name == "" {
		return fmt.Errorf("job name cannot be empty")
	}

	if job.Handler == nil {
		return fmt.Errorf("job handler cannot be nil")
	}

	if _, exists := r.jobs[job.Name]; exists {
		return fmt.Errorf("job %s is already registered", job.Name)
	}

	// Set default timeout if not specified
	if job.TimeoutSeconds <= 0 {
		job.TimeoutSeconds = 300 // 5 minutes default
	}

	r.jobs[job.Name] = &job
	return nil
}

// Get returns a job by name
func (r *Registry) Get(name string) (*JobDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, exists := r.jobs[name]
	return job, exists
}

// List returns all registered jobs
func (r *Registry) List() []*JobDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jobs := make([]*JobDefinition, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// Count returns the number of registered jobs
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.jobs)
}

// RegisterJob is a convenience function to register a job with the global registry
func RegisterJob(job JobDefinition) error {
	return GetRegistry().Register(job)
}
