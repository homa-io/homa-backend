package jobs

import (
	"strconv"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/iesreza/homa-backend/lib/response"
)

// ListJobs returns all registered jobs with their status
// GET /api/v1/jobs
func ListJobs(r *evo.Request) any {
	executor := GetExecutor()
	jobs := executor.GetJobsInfo()

	// Get base URL for trigger URLs
	baseURL := getAPIBaseURL(r)

	// Add trigger URL to each job
	for i := range jobs {
		jobs[i].TriggerURL = baseURL + "/api/v1/jobs/" + jobs[i].Name
	}

	// Trigger lazy cleanup of old logs
	go CleanupOldLogs()

	return response.OK(JobListResponse{Jobs: jobs})
}

// getAPIBaseURL returns the API base URL from settings or request
func getAPIBaseURL(r *evo.Request) string {
	// First check for configured API base URL
	apiBaseURL := settings.Get("APP.API_BASE_URL").String()
	if apiBaseURL != "" {
		return apiBaseURL
	}

	// Fallback to request headers for reverse proxy setups
	proto := r.Get("X-Forwarded-Proto").String()
	if proto == "" {
		proto = "https"
	}

	host := r.Get("X-Forwarded-Host").String()
	if host == "" {
		host = r.Hostname()
	}

	return proto + "://" + host
}

// TriggerJob executes a job by name and returns the result
// POST /api/v1/jobs/:name
func TriggerJob(r *evo.Request) any {
	jobName := r.Param("name").String()
	if jobName == "" {
		return response.BadRequest(nil, "Job name is required")
	}

	executor := GetExecutor()
	result, err := executor.Execute(jobName)
	if err != nil {
		return response.InternalError(nil, err.Error())
	}

	// If job was skipped (already running)
	if result.Status == JobStatusSkipped {
		return response.Conflict(nil, result.Error)
	}

	return response.OK(result)
}

// GetJobHistory returns execution history for a job
// GET /api/v1/jobs/:name/history
func GetJobHistory(r *evo.Request) any {
	jobName := r.Param("name").String()

	limitStr := r.Query("limit").String()
	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	executions, err := GetExecutionHistory(jobName, limit)
	if err != nil {
		return response.InternalError(nil, err.Error())
	}

	return response.OK(executions)
}

// GetJobStatus returns the current status of a job
// GET /api/v1/jobs/:name/status
func GetJobStatus(r *evo.Request) any {
	jobName := r.Param("name").String()
	if jobName == "" {
		return response.BadRequest(nil, "Job name is required")
	}

	registry := GetRegistry()
	job, exists := registry.Get(jobName)
	if !exists {
		return response.NotFound(nil, "Job not found")
	}

	executor := GetExecutor()
	isRunning := executor.IsRunning(jobName)

	// Get last execution
	lastExec, _ := GetLastExecution(jobName)

	return response.OK(map[string]interface{}{
		"name":           job.Name,
		"description":    job.Description,
		"timeout_seconds": job.TimeoutSeconds,
		"is_running":     isRunning,
		"last_execution": lastExec,
	})
}
