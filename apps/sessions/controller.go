package sessions

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/outcome"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/datatypes"
)

type Controller struct{}

// Request/Response types
type StartSessionRequest struct {
	SessionID  string         `json:"session_id"`
	TabID      string         `json:"tab_id,omitempty"`
	DeviceInfo map[string]any `json:"device_info,omitempty"`
}

type HeartbeatRequest struct {
	SessionID string `json:"session_id"`
	TabID     string `json:"tab_id,omitempty"`
}

type EndSessionRequest struct {
	SessionID string `json:"session_id"`
	TabID     string `json:"tab_id,omitempty"`
	Reason    string `json:"reason,omitempty"` // logout, tab_close
}

// Heartbeat interval constant (30 seconds is typical)
const HeartbeatInterval = 30 * time.Second
const SessionTimeout = 5 * time.Minute // Mark session as ended if no heartbeat for 5 minutes

// getRealIP extracts the real client IP from request headers (prioritizes X-Real-IP, then X-Forwarded-For, then fallback)
func getRealIP(request *evo.Request) string {
	// First try X-Real-IP (set by nginx)
	if realIP := request.Header("X-Real-IP"); realIP != "" {
		return realIP
	}
	// Then try the first IP from X-Forwarded-For
	if forwardedFor := request.Header("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one (original client)
		if idx := strings.Index(forwardedFor, ","); idx > 0 {
			return strings.TrimSpace(forwardedFor[:idx])
		}
		return strings.TrimSpace(forwardedFor)
	}
	// Fallback to request.IP()
	return request.IP()
}

// StartSession creates or updates a session when user starts their browsing session
// Uses session_id from cookie - if session exists (regardless of is_active), just update last_activity
func (c Controller) StartSession(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var req StartSessionRequest
	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.SessionID == "" {
		return response.Error(response.NewError(response.ErrorCodeMissingRequired, "session_id is required", http.StatusBadRequest))
	}

	now := time.Now()
	ip := getRealIP(request)
	userAgent := request.Header("User-Agent")

	// Check for existing session with this session_id (regardless of is_active status)
	var existingSession models.UserSession
	err := db.Where("user_id = ? AND session_id = ?", user.UserID, req.SessionID).First(&existingSession).Error

	if err == nil {
		// Session exists, update last activity and IP (may have changed)
		existingSession.LastActivity = now
		existingSession.IPAddress = ip
		existingSession.UserAgent = userAgent
		if req.TabID != "" {
			existingSession.TabID = &req.TabID
		}
		db.Save(&existingSession)

		// Update daily activity
		go updateDailyActivity(user.UserID, now)

		return response.OKWithMessage(map[string]any{
			"session_id": existingSession.ID,
		}, "session resumed")
	}

	// Create new session
	deviceInfoJSON, _ := json.Marshal(req.DeviceInfo)
	session := models.UserSession{
		UserID:       user.UserID,
		SessionID:    req.SessionID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		DeviceInfo:   datatypes.JSON(deviceInfoJSON),
		StartedAt:    now,
		LastActivity: now,
	}
	if req.TabID != "" {
		session.TabID = &req.TabID
	}

	if err := db.Create(&session).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Update daily activity
	go updateDailyActivity(user.UserID, now)

	return response.OKWithMessage(map[string]any{
		"session_id": session.ID,
	}, "session started")
}

// Heartbeat updates the session's last activity timestamp
// No longer checks is_active - just updates last_activity if session exists
func (c Controller) Heartbeat(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var req HeartbeatRequest
	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.SessionID == "" {
		return response.Error(response.NewError(response.ErrorCodeMissingRequired, "session_id is required", http.StatusBadRequest))
	}

	now := time.Now()

	// Find session (regardless of is_active status)
	var session models.UserSession
	if err := db.Where("user_id = ? AND session_id = ?", user.UserID, req.SessionID).First(&session).Error; err != nil {
		// Session not found, need to restart
		return outcome.Response{
			StatusCode: http.StatusNotFound,
			Data: response.APIResponse{
				Success: false,
				Message: "session not found",
				Data:    map[string]string{"code": "SESSION_NOT_FOUND"},
			}.ToJSON(),
		}
	}

	// Update last activity
	session.LastActivity = now
	if err := db.Save(&session).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Update daily activity async
	go updateDailyActivity(user.UserID, now)

	return response.OK(map[string]any{
		"last_activity": now,
	})
}

// EndSession marks a session as ended by setting last_activity to far past
// This effectively marks it as inactive (older than 5 min threshold)
func (c Controller) EndSession(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var req EndSessionRequest
	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.SessionID == "" {
		return response.Error(response.NewError(response.ErrorCodeMissingRequired, "session_id is required", http.StatusBadRequest))
	}

	// Set last_activity to 1 hour ago to mark as inactive
	inactiveTime := time.Now().Add(-1 * time.Hour)
	result := db.Model(&models.UserSession{}).
		Where("user_id = ? AND session_id = ?", user.UserID, req.SessionID).
		Update("last_activity", inactiveTime)

	if result.RowsAffected == 0 {
		return response.Error(response.ErrNotFound)
	}

	return response.Message("session ended")
}

// GetMySessions returns all sessions for the current user
func (c Controller) GetMySessions(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var sessions []models.UserSession
	db.Where("user_id = ?", user.UserID).
		Order("started_at DESC").
		Limit(50).
		Find(&sessions)

	return response.OK(map[string]any{
		"sessions": sessions,
	})
}

// GetActiveSessions returns sessions where last_activity is within the last 5 minutes
// Active status is now determined by last_activity time, not is_active field
func (c Controller) GetActiveSessions(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Return sessions with recent activity (within 5 minutes)
	activeThreshold := time.Now().Add(-SessionTimeout)

	var sessions []models.UserSession
	db.Where("user_id = ? AND last_activity >= ?", user.UserID, activeThreshold).
		Order("last_activity DESC").
		Find(&sessions)

	return response.OK(map[string]any{
		"sessions": sessions,
	})
}

// TerminateSession ends a specific session by ID
// Sets last_activity to past to mark as inactive
func (c Controller) TerminateSession(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	sessionID := request.Param("id").String()
	if sessionID == "" {
		return response.Error(response.NewError(response.ErrorCodeMissingRequired, "session id required", http.StatusBadRequest))
	}

	now := time.Now()
	// Only terminate sessions that are still "active" (last_activity within 5 min)
	activeThreshold := now.Add(-SessionTimeout)
	inactiveTime := now.Add(-1 * time.Hour)
	result := db.Model(&models.UserSession{}).
		Where("id = ? AND user_id = ? AND last_activity >= ?", sessionID, user.UserID, activeThreshold).
		Update("last_activity", inactiveTime)

	if result.RowsAffected == 0 {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "session not found or already ended", http.StatusNotFound))
	}

	return response.Message("session terminated")
}

// TerminateAllOtherSessions ends all active sessions except the current one
// Sets last_activity to past to mark as inactive
func (c Controller) TerminateAllOtherSessions(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var req struct {
		CurrentSessionID string `json:"current_session_id"`
	}
	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	now := time.Now()
	// Only terminate sessions that are still "active" (last_activity within 5 min)
	activeThreshold := now.Add(-SessionTimeout)
	inactiveTime := now.Add(-1 * time.Hour)
	result := db.Model(&models.UserSession{}).
		Where("user_id = ? AND session_id != ? AND last_activity >= ?", user.UserID, req.CurrentSessionID, activeThreshold).
		Update("last_activity", inactiveTime)

	return response.OKWithMessage(map[string]any{
		"terminated_count": result.RowsAffected,
	}, "other sessions terminated")
}

// GetSessionHistory returns paginated session history with optional date range filter
func (c Controller) GetSessionHistory(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Pagination params
	page := request.Query("page").Int()
	if page < 1 {
		page = 1
	}
	limit := request.Query("limit").Int()
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Date range params
	startDate := request.Query("start_date").String()
	endDate := request.Query("end_date").String()

	// Build query
	query := db.Model(&models.UserSession{}).Where("user_id = ?", user.UserID)

	if startDate != "" {
		startTime, err := time.Parse("2006-01-02", startDate)
		if err == nil {
			query = query.Where("started_at >= ?", startTime)
		}
	}
	if endDate != "" {
		endTime, err := time.Parse("2006-01-02", endDate)
		if err == nil {
			// Include the entire end date
			endTime = endTime.Add(24*time.Hour - time.Second)
			query = query.Where("started_at <= ?", endTime)
		}
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	var sessions []models.UserSession
	query.Order("started_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&sessions)

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return response.OK(map[string]any{
		"sessions": sessions,
		"pagination": map[string]any{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetDailyActivity returns daily activity for a date range
func (c Controller) GetDailyActivity(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Get date range from query params
	startDate := request.Query("start_date").String()
	endDate := request.Query("end_date").String()

	if startDate == "" {
		// Default to last 30 days
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	var activities []models.UserDailyActivity
	db.Where("user_id = ? AND activity_date BETWEEN ? AND ?", user.UserID, startDate, endDate).
		Order("activity_date DESC").
		Find(&activities)

	return response.OK(map[string]any{
		"activities": activities,
	})
}

// GetTodayActivity returns the current user's activity for today (server time)
func (c Controller) GetTodayActivity(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	today := time.Now().Format("2006-01-02")

	var activity models.UserDailyActivity
	err := db.Where("user_id = ? AND activity_date = ?", user.UserID, today).First(&activity).Error

	if err != nil {
		// No activity for today yet - return zero values
		return response.OK(map[string]any{
			"activity": map[string]any{
				"activity_date":        today,
				"total_active_seconds": 0,
				"session_count":        0,
			},
		})
	}

	return response.OK(map[string]any{
		"activity": activity,
	})
}

// GetActivitySummary returns aggregated activity summary
func (c Controller) GetActivitySummary(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Get period from query params (default: current month)
	period := request.Query("period").String()
	if period == "" {
		period = "month"
	}

	var startDate time.Time
	now := time.Now()

	switch period {
	case "week":
		startDate = now.AddDate(0, 0, -7)
	case "month":
		startDate = now.AddDate(0, -1, 0)
	case "year":
		startDate = now.AddDate(-1, 0, 0)
	default:
		startDate = now.AddDate(0, -1, 0)
	}

	// Calculate total active time
	var totalSeconds int64
	var totalDays int64
	db.Model(&models.UserDailyActivity{}).
		Where("user_id = ? AND activity_date >= ?", user.UserID, startDate.Format("2006-01-02")).
		Select("COALESCE(SUM(total_active_seconds), 0) as total_seconds, COUNT(*) as total_days").
		Row().Scan(&totalSeconds, &totalDays)

	// Calculate average per day
	avgSecondsPerDay := float64(0)
	if totalDays > 0 {
		avgSecondsPerDay = float64(totalSeconds) / float64(totalDays)
	}

	return response.OK(map[string]any{
		"summary": map[string]any{
			"total_hours":           float64(totalSeconds) / 3600,
			"total_days":            totalDays,
			"average_hours_per_day": avgSecondsPerDay / 3600,
			"period":                period,
			"start_date":            startDate.Format("2006-01-02"),
			"end_date":              now.Format("2006-01-02"),
		},
	})
}

// GetActivityStats returns detailed activity statistics
func (c Controller) GetActivityStats(request *evo.Request) any {
	user := request.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	month := request.Query("month").String()
	year := request.Query("year").String()

	if month == "" {
		month = time.Now().Format("01")
	}
	if year == "" {
		year = time.Now().Format("2006")
	}

	// Get all activity for the month
	datePrefix := year + "-" + month
	var activities []models.UserDailyActivity
	db.Where("user_id = ? AND activity_date LIKE ?", user.UserID, datePrefix+"%").
		Order("activity_date ASC").
		Find(&activities)

	// Format for chart data
	chartData := make([]map[string]any, 0)
	for _, activity := range activities {
		chartData = append(chartData, map[string]any{
			"date":   activity.ActivityDate,
			"hours":  float64(activity.TotalActiveSeconds) / 3600,
			"active": activity.TotalActiveSeconds > 0,
		})
	}

	// Calculate summary
	var totalSeconds int64
	for _, a := range activities {
		totalSeconds += int64(a.TotalActiveSeconds)
	}

	return response.OK(map[string]any{
		"stats": map[string]any{
			"month":       month,
			"year":        year,
			"chart_data":  chartData,
			"total_hours": float64(totalSeconds) / 3600,
			"active_days": len(activities),
		},
	})
}

// Helper function to update daily activity
func updateDailyActivity(userID uuid.UUID, activityTime time.Time) {
	dateStr := activityTime.Format("2006-01-02")

	// Try to find existing record for today
	var activity models.UserDailyActivity
	err := db.Where("user_id = ? AND activity_date = ?", userID, dateStr).First(&activity).Error

	if err != nil {
		// Create new record
		activity = models.UserDailyActivity{
			UserID:             userID,
			ActivityDate:       dateStr,
			TotalActiveSeconds: int(HeartbeatInterval.Seconds()),
			FirstActivity:      activityTime,
			LastActivity:       activityTime,
			SessionCount:       1,
			ActivePeriods:      datatypes.JSON("[]"),
		}
		if createErr := db.Create(&activity).Error; createErr != nil {
			log.Error("Failed to create daily activity: ", createErr)
		}
		return
	}

	// Update existing record
	// Add time since last activity (minimum 1 second, max heartbeat interval)
	timeSinceLast := activityTime.Sub(activity.LastActivity)
	additionalSeconds := int(HeartbeatInterval.Seconds())

	// Only add time if this isn't a duplicate heartbeat (at least 1 second apart)
	if timeSinceLast.Seconds() < 1 {
		return // Skip duplicate heartbeat
	}

	// Cap at heartbeat interval to avoid huge jumps after inactivity
	if timeSinceLast < HeartbeatInterval*2 {
		additionalSeconds = int(timeSinceLast.Seconds())
	}

	// Use direct update query for reliability
	updateResult := db.Model(&models.UserDailyActivity{}).
		Where("id = ?", activity.ID).
		Updates(map[string]any{
			"total_active_seconds": activity.TotalActiveSeconds + additionalSeconds,
			"last_activity":        activityTime,
		})

	if updateResult.Error != nil {
		log.Error("Failed to update daily activity: ", updateResult.Error)
	}
}
