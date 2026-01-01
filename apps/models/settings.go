package models

import (
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"gorm.io/gorm"
)

// Setting represents a configurable setting stored in the database
// Note: Settings don't use soft delete - they are hard deleted
type Setting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"column:setting_key;type:varchar(255);uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Type      string    `gorm:"type:varchar(50);default:'string'" json:"type"` // string, number, boolean, json
	Category  string    `gorm:"type:varchar(100);index" json:"category"`       // ai, workflow, general, etc.
	Label     string    `gorm:"type:varchar(255)" json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Setting) TableName() string {
	return "settings"
}

// GetSetting retrieves a setting by key
func GetSetting(key string) (*Setting, error) {
	var setting Setting
	err := db.Where("setting_key = ?", key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetSettingValue retrieves only the value of a setting
func GetSettingValue(key string, defaultValue string) string {
	setting, err := GetSetting(key)
	if err != nil {
		return defaultValue
	}
	return setting.Value
}

// SetSetting creates or updates a setting
func SetSetting(key, value, settingType, category, label string) error {
	var setting Setting
	err := db.Where("setting_key = ?", key).First(&setting).Error
	if err != nil {
		// Create new setting
		setting = Setting{
			Key:      key,
			Value:    value,
			Type:     settingType,
			Category: category,
			Label:    label,
		}
		return db.Create(&setting).Error
	}

	// Update existing setting
	setting.Value = value
	if settingType != "" {
		setting.Type = settingType
	}
	if category != "" {
		setting.Category = category
	}
	if label != "" {
		setting.Label = label
	}
	return db.Save(&setting).Error
}

// GetSettingsByCategory retrieves all settings in a category
func GetSettingsByCategory(category string) ([]Setting, error) {
	var settings []Setting
	err := db.Where("category = ?", category).Find(&settings).Error
	return settings, err
}

// GetAllSettings retrieves all settings
func GetAllSettings() ([]Setting, error) {
	var settings []Setting
	err := db.Find(&settings).Error
	return settings, err
}

// DeleteSetting deletes a setting by key
func DeleteSetting(key string) error {
	return db.Where("setting_key = ?", key).Delete(&Setting{}).Error
}

// BulkUpdateSettings updates multiple settings at once
func BulkUpdateSettings(updates map[string]string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for key, value := range updates {
			var setting Setting
			err := tx.Where("setting_key = ?", key).First(&setting).Error
			if err != nil {
				// Create new setting if not exists
				setting = Setting{
					Key:   key,
					Value: value,
					Type:  "string",
				}
				if err := tx.Create(&setting).Error; err != nil {
					return err
				}
			} else {
				setting.Value = value
				if err := tx.Save(&setting).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
