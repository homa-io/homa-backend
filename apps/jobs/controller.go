package jobs

import (
	"strconv"

	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/lib/response"
)

// JobResponse represents a job in API responses
type JobResponse struct {
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Schedule      string        `json:"schedule"`
	Enabled       bool          `json:"enabled"`
	LastExecution *JobExecution `json:"last_execution,omitempty"`
}

// GetJobs returns all registered jobs with their last execution
func GetJobs(r *evo.Request) any {
	scheduler := GetScheduler()
	if scheduler == nil {
		return response.OK([]JobResponse{})
	}

	jobs := scheduler.GetJobs()
	jobResponses := make([]JobResponse, 0, len(jobs))

	for name, job := range jobs {
		jobResp := JobResponse{
			Name:        name,
			Description: job.Description,
			Schedule:    job.Schedule,
			Enabled:     job.Enabled,
		}

		// Get last execution
		lastExec, err := scheduler.GetLastExecution(name)
		if err == nil && lastExec != nil {
			jobResp.LastExecution = lastExec
		}

		jobResponses = append(jobResponses, jobResp)
	}

	return response.OK(jobResponses)
}

// GetJobExecutions returns recent executions for a specific job or all jobs
func GetJobExecutions(r *evo.Request) any {
	scheduler := GetScheduler()
	if scheduler == nil {
		return response.OK([]JobExecution{})
	}

	jobName := r.Query("job_name").String()
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

	executions, err := scheduler.GetRecentExecutions(jobName, limit)
	if err != nil {
		return response.InternalError(nil, err.Error())
	}

	return response.OK(executions)
}

// RunJob triggers immediate execution of a job
func RunJob(r *evo.Request) any {
	scheduler := GetScheduler()
	if scheduler == nil {
		return response.InternalError(nil, "Scheduler not initialized")
	}

	jobName := r.Param("name").String()
	if jobName == "" {
		return response.BadRequest(nil, "Job name is required")
	}

	err := scheduler.RunNow(jobName)
	if err != nil {
		return response.InternalError(nil, err.Error())
	}

	return response.OKWithMessage(nil, "Job triggered successfully")
}
