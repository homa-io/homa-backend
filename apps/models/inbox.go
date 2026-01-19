package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"gorm.io/datatypes"
)

// Inbox represents a web chat inbox with its own SDK configuration and timeout
type Inbox struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	Name                string         `gorm:"size:255;not null" json:"name"`
	Description         string         `gorm:"size:500" json:"description"`
	SDKConfig           datatypes.JSON `gorm:"type:json" json:"sdk_config"`
	ConversationTimeout int            `gorm:"default:48" json:"conversation_timeout"` // hours until auto-close, 0 = disabled
	APIKey              string         `gorm:"size:100;uniqueIndex;not null" json:"api_key"`
	Enabled             bool           `gorm:"default:1" json:"enabled"`
	CreatedAt           time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Inbox) TableName() string {
	return "inboxes"
}

// GenerateAPIKey generates a random API key for an inbox
func GenerateInboxAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "inbox_" + hex.EncodeToString(bytes)
}

// GetInboxByID retrieves an inbox by ID
func GetInboxByID(id uint) (*Inbox, error) {
	var inbox Inbox
	if err := db.First(&inbox, id).Error; err != nil {
		return nil, err
	}
	return &inbox, nil
}

// GetInboxByAPIKey retrieves an inbox by API key
func GetInboxByAPIKey(apiKey string) (*Inbox, error) {
	var inbox Inbox
	if err := db.Where("api_key = ?", apiKey).First(&inbox).Error; err != nil {
		return nil, err
	}
	return &inbox, nil
}

// GetAllInboxes retrieves all inboxes
func GetAllInboxes() ([]Inbox, error) {
	var inboxes []Inbox
	if err := db.Order("created_at ASC").Find(&inboxes).Error; err != nil {
		return nil, err
	}
	return inboxes, nil
}

// GetEnabledInboxes retrieves all enabled inboxes
func GetEnabledInboxes() ([]Inbox, error) {
	var inboxes []Inbox
	if err := db.Where("enabled = ?", true).Order("created_at ASC").Find(&inboxes).Error; err != nil {
		return nil, err
	}
	return inboxes, nil
}

// CreateInbox creates a new inbox
func CreateInbox(inbox *Inbox) error {
	if inbox.APIKey == "" {
		inbox.APIKey = GenerateInboxAPIKey()
	}
	return db.Create(inbox).Error
}

// UpdateInbox updates an existing inbox
func UpdateInbox(inbox *Inbox) error {
	return db.Save(inbox).Error
}

// DeleteInbox deletes an inbox by ID
func DeleteInbox(id uint) error {
	return db.Delete(&Inbox{}, id).Error
}

// GetDefaultInbox returns the first inbox or creates one if none exists
func GetDefaultInbox() (*Inbox, error) {
	var inbox Inbox
	err := db.Order("created_at ASC").First(&inbox).Error
	if err != nil {
		return nil, err
	}
	return &inbox, nil
}

// EnsureDefaultInbox creates a default inbox if none exists
func EnsureDefaultInbox() (*Inbox, error) {
	var count int64
	db.Model(&Inbox{}).Count(&count)

	if count == 0 {
		// Create default inbox with empty SDK config
		inbox := &Inbox{
			Name:                "Default Inbox",
			Description:         "Default web chat inbox",
			SDKConfig:           []byte("{}"), // Empty JSON object
			ConversationTimeout: 48,           // 48 hours default
			Enabled:             true,
		}
		if err := CreateInbox(inbox); err != nil {
			return nil, err
		}
		return inbox, nil
	}

	return GetDefaultInbox()
}
