package models

import (
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/google/uuid"
)

// UserPreference represents user notification preferences
type UserPreference struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	UserID               uuid.UUID `gorm:"uniqueIndex;not null" json:"user_id"`
	NotificationSound    string    `gorm:"type:varchar(50);default:'chime'" json:"notification_sound"` // chime, bell, ding, pop, none
	SoundVolume          int       `gorm:"default:50" json:"sound_volume"`                             // 0-100
	BrowserNotifications bool      `gorm:"type:tinyint(1);default:1" json:"browser_notifications"`
	DesktopBadge         bool      `gorm:"type:tinyint(1);default:1" json:"desktop_badge"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (UserPreference) TableName() string {
	return "user_preferences"
}

// GetUserPreferences retrieves preferences for a user, creating default if not exists
func GetUserPreferences(userID uuid.UUID) (*UserPreference, error) {
	var prefs UserPreference
	err := db.Where("user_id = ?", userID).First(&prefs).Error
	if err != nil {
		// Create default preferences
		prefs = UserPreference{
			UserID:               userID,
			NotificationSound:    "chime",
			SoundVolume:          50,
			BrowserNotifications: true,
			DesktopBadge:         true,
		}
		if err := db.Create(&prefs).Error; err != nil {
			return nil, err
		}
	}
	return &prefs, nil
}

// UpdateUserPreferences updates user preferences
func UpdateUserPreferences(userID uuid.UUID, updates map[string]interface{}) error {
	// Ensure preferences exist
	_, err := GetUserPreferences(userID)
	if err != nil {
		return err
	}

	return db.Model(&UserPreference{}).Where("user_id = ?", userID).Updates(updates).Error
}

// Available notification sounds
var NotificationSounds = []string{
	"none",
	"chime",
	"bell",
	"ding",
	"pop",
	"notification",
}

// IsValidNotificationSound checks if a sound name is valid
func IsValidNotificationSound(sound string) bool {
	for _, s := range NotificationSounds {
		if s == sound {
			return true
		}
	}
	return false
}
