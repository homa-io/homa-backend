package jobs

import (
	"strconv"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// Job settings keys
const (
	// CSAT Email Settings
	SettingCSATEnabled   = "jobs.csat.enabled"
	SettingCSATDelayHours = "jobs.csat.delay_hours"

	// Close Chat Settings
	SettingCloseChatEnabled    = "jobs.close_chat.enabled"
	SettingCloseChatAfterHours = "jobs.close_chat.after_hours"

	// Close Email Conversations Settings
	SettingCloseEmailEnabled    = "jobs.close_email.enabled"
	SettingCloseEmailAfterHours = "jobs.close_email.after_hours"

	// Archive Tickets Settings
	SettingArchiveEnabled   = "jobs.archive.enabled"
	SettingArchiveAfterDays = "jobs.archive.after_days"

	// Delete Archived Tickets Settings
	SettingDeleteArchivedEnabled   = "jobs.delete_archived.enabled"
	SettingDeleteArchivedAfterDays = "jobs.delete_archived.after_days"
)

// JobSettingsCategory is the settings category for job settings
const JobSettingsCategory = "jobs"

// DefaultJobSettings defines the default values for job settings
var DefaultJobSettings = []models.Setting{
	// CSAT Email Settings
	{
		Key:      SettingCSATEnabled,
		Value:    "true",
		Type:     "boolean",
		Category: JobSettingsCategory,
		Label:    "Enable CSAT Emails",
	},
	{
		Key:      SettingCSATDelayHours,
		Value:    "24",
		Type:     "number",
		Category: JobSettingsCategory,
		Label:    "Send CSAT Email After (hours)",
	},

	// Close Chat Settings
	{
		Key:      SettingCloseChatEnabled,
		Value:    "true",
		Type:     "boolean",
		Category: JobSettingsCategory,
		Label:    "Enable Auto-Close Chats",
	},
	{
		Key:      SettingCloseChatAfterHours,
		Value:    "48",
		Type:     "number",
		Category: JobSettingsCategory,
		Label:    "Close Chats After (hours)",
	},

	// Close Email Conversations Settings
	{
		Key:      SettingCloseEmailEnabled,
		Value:    "true",
		Type:     "boolean",
		Category: JobSettingsCategory,
		Label:    "Enable Auto-Close Email Conversations",
	},
	{
		Key:      SettingCloseEmailAfterHours,
		Value:    "168",
		Type:     "number",
		Category: JobSettingsCategory,
		Label:    "Close Email Conversations After (hours)",
	},

	// Archive Tickets Settings
	{
		Key:      SettingArchiveEnabled,
		Value:    "true",
		Type:     "boolean",
		Category: JobSettingsCategory,
		Label:    "Enable Auto-Archive Tickets",
	},
	{
		Key:      SettingArchiveAfterDays,
		Value:    "90",
		Type:     "number",
		Category: JobSettingsCategory,
		Label:    "Archive Tickets After (days)",
	},

	// Delete Archived Tickets Settings
	{
		Key:      SettingDeleteArchivedEnabled,
		Value:    "false",
		Type:     "boolean",
		Category: JobSettingsCategory,
		Label:    "Enable Delete Archived Tickets",
	},
	{
		Key:      SettingDeleteArchivedAfterDays,
		Value:    "365",
		Type:     "number",
		Category: JobSettingsCategory,
		Label:    "Delete Archived Tickets After (days)",
	},
}

// InitJobSettings creates default job settings if they don't exist
func InitJobSettings() {
	for _, setting := range DefaultJobSettings {
		existing, err := models.GetSetting(setting.Key)
		if err != nil || existing == nil {
			if err := models.SetSetting(setting.Key, setting.Value, setting.Type, setting.Category, setting.Label); err != nil {
				log.Error("[jobs] Failed to create default setting %s: %v", setting.Key, err)
			} else {
				log.Debug("[jobs] Created default setting: %s = %s", setting.Key, setting.Value)
			}
		}
	}
}

// JobSettingsResponse represents the job settings for API response
type JobSettingsResponse struct {
	CSAT struct {
		Enabled    bool `json:"enabled"`
		DelayHours int  `json:"delay_hours"`
	} `json:"csat"`
	CloseChat struct {
		Enabled    bool `json:"enabled"`
		AfterHours int  `json:"after_hours"`
	} `json:"close_chat"`
	CloseEmail struct {
		Enabled    bool `json:"enabled"`
		AfterHours int  `json:"after_hours"`
	} `json:"close_email"`
	Archive struct {
		Enabled   bool `json:"enabled"`
		AfterDays int  `json:"after_days"`
	} `json:"archive"`
	DeleteArchived struct {
		Enabled   bool `json:"enabled"`
		AfterDays int  `json:"after_days"`
	} `json:"delete_archived"`
}

// JobSettingsUpdateRequest represents the request to update job settings
type JobSettingsUpdateRequest struct {
	CSAT *struct {
		Enabled    *bool `json:"enabled"`
		DelayHours *int  `json:"delay_hours"`
	} `json:"csat,omitempty"`
	CloseChat *struct {
		Enabled    *bool `json:"enabled"`
		AfterHours *int  `json:"after_hours"`
	} `json:"close_chat,omitempty"`
	CloseEmail *struct {
		Enabled    *bool `json:"enabled"`
		AfterHours *int  `json:"after_hours"`
	} `json:"close_email,omitempty"`
	Archive *struct {
		Enabled   *bool `json:"enabled"`
		AfterDays *int  `json:"after_days"`
	} `json:"archive,omitempty"`
	DeleteArchived *struct {
		Enabled   *bool `json:"enabled"`
		AfterDays *int  `json:"after_days"`
	} `json:"delete_archived,omitempty"`
}

// GetJobSettings returns all job settings
// GET /api/settings/jobs
func GetJobSettings(req *evo.Request) interface{} {
	settings := JobSettingsResponse{}

	// CSAT settings
	settings.CSAT.Enabled = getSettingBool(SettingCSATEnabled, true)
	settings.CSAT.DelayHours = getSettingInt(SettingCSATDelayHours, 24)

	// Close chat settings
	settings.CloseChat.Enabled = getSettingBool(SettingCloseChatEnabled, true)
	settings.CloseChat.AfterHours = getSettingInt(SettingCloseChatAfterHours, 48)

	// Close email settings
	settings.CloseEmail.Enabled = getSettingBool(SettingCloseEmailEnabled, true)
	settings.CloseEmail.AfterHours = getSettingInt(SettingCloseEmailAfterHours, 168)

	// Archive settings
	settings.Archive.Enabled = getSettingBool(SettingArchiveEnabled, true)
	settings.Archive.AfterDays = getSettingInt(SettingArchiveAfterDays, 90)

	// Delete archived settings
	settings.DeleteArchived.Enabled = getSettingBool(SettingDeleteArchivedEnabled, false)
	settings.DeleteArchived.AfterDays = getSettingInt(SettingDeleteArchivedAfterDays, 365)

	return response.OK(settings)
}

// UpdateJobSettings updates job settings
// PUT /api/settings/jobs
func UpdateJobSettings(req *evo.Request) interface{} {
	var request JobSettingsUpdateRequest
	if err := req.BodyParser(&request); err != nil {
		return response.BadRequest(nil, "Invalid request body")
	}

	// Update CSAT settings
	if request.CSAT != nil {
		if request.CSAT.Enabled != nil {
			updateSettingBool(SettingCSATEnabled, *request.CSAT.Enabled)
		}
		if request.CSAT.DelayHours != nil {
			if *request.CSAT.DelayHours < 1 {
				return response.BadRequest(nil, "CSAT delay hours must be at least 1")
			}
			updateSettingInt(SettingCSATDelayHours, *request.CSAT.DelayHours)
		}
	}

	// Update close chat settings
	if request.CloseChat != nil {
		if request.CloseChat.Enabled != nil {
			updateSettingBool(SettingCloseChatEnabled, *request.CloseChat.Enabled)
		}
		if request.CloseChat.AfterHours != nil {
			if *request.CloseChat.AfterHours < 1 {
				return response.BadRequest(nil, "Close chat after hours must be at least 1")
			}
			updateSettingInt(SettingCloseChatAfterHours, *request.CloseChat.AfterHours)
		}
	}

	// Update close email settings
	if request.CloseEmail != nil {
		if request.CloseEmail.Enabled != nil {
			updateSettingBool(SettingCloseEmailEnabled, *request.CloseEmail.Enabled)
		}
		if request.CloseEmail.AfterHours != nil {
			if *request.CloseEmail.AfterHours < 1 {
				return response.BadRequest(nil, "Close email after hours must be at least 1")
			}
			updateSettingInt(SettingCloseEmailAfterHours, *request.CloseEmail.AfterHours)
		}
	}

	// Update archive settings
	if request.Archive != nil {
		if request.Archive.Enabled != nil {
			updateSettingBool(SettingArchiveEnabled, *request.Archive.Enabled)
		}
		if request.Archive.AfterDays != nil {
			if *request.Archive.AfterDays < 1 {
				return response.BadRequest(nil, "Archive after days must be at least 1")
			}
			updateSettingInt(SettingArchiveAfterDays, *request.Archive.AfterDays)
		}
	}

	// Update delete archived settings
	if request.DeleteArchived != nil {
		if request.DeleteArchived.Enabled != nil {
			updateSettingBool(SettingDeleteArchivedEnabled, *request.DeleteArchived.Enabled)
		}
		if request.DeleteArchived.AfterDays != nil {
			if *request.DeleteArchived.AfterDays < 1 {
				return response.BadRequest(nil, "Delete archived after days must be at least 1")
			}
			updateSettingInt(SettingDeleteArchivedAfterDays, *request.DeleteArchived.AfterDays)
		}
	}

	// Return updated settings
	return GetJobSettings(req)
}

// Helper functions

func getSettingBool(key string, defaultValue bool) bool {
	setting, err := models.GetSetting(key)
	if err != nil || setting == nil {
		return defaultValue
	}
	return setting.Value == "true" || setting.Value == "1"
}

func getSettingInt(key string, defaultValue int) int {
	setting, err := models.GetSetting(key)
	if err != nil || setting == nil {
		return defaultValue
	}
	val, err := strconv.Atoi(setting.Value)
	if err != nil {
		return defaultValue
	}
	return val
}

func updateSettingBool(key string, value bool) {
	strValue := "false"
	if value {
		strValue = "true"
	}
	models.SetSetting(key, strValue, "", "", "")
}

func updateSettingInt(key string, value int) {
	models.SetSetting(key, strconv.Itoa(value), "", "", "")
}

// GetCSATSettings returns CSAT settings for use in jobs
func GetCSATSettings() (enabled bool, delayHours int) {
	return getSettingBool(SettingCSATEnabled, true), getSettingInt(SettingCSATDelayHours, 24)
}

// GetCloseChatSettings returns close chat settings for use in jobs
func GetCloseChatSettings() (enabled bool, afterHours int) {
	return getSettingBool(SettingCloseChatEnabled, true), getSettingInt(SettingCloseChatAfterHours, 48)
}

// GetCloseEmailSettings returns close email settings for use in jobs
func GetCloseEmailSettings() (enabled bool, afterHours int) {
	return getSettingBool(SettingCloseEmailEnabled, true), getSettingInt(SettingCloseEmailAfterHours, 168)
}

// GetArchiveSettings returns archive settings for use in jobs
func GetArchiveSettings() (enabled bool, afterDays int) {
	return getSettingBool(SettingArchiveEnabled, true), getSettingInt(SettingArchiveAfterDays, 90)
}

// GetDeleteArchivedSettings returns delete archived settings for use in jobs
func GetDeleteArchivedSettings() (enabled bool, afterDays int) {
	return getSettingBool(SettingDeleteArchivedEnabled, false), getSettingInt(SettingDeleteArchivedAfterDays, 365)
}
