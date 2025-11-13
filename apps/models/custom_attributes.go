package models

import (
	"github.com/getevo/restify"
	"time"
)

// CustomAttribute scope constants
const (
	CustomAttributeScopeClient       = "client"
	CustomAttributeScopeConversation = "conversation"
)

// CustomAttribute data type constants
const (
	CustomAttributeDataTypeInt    = "int"
	CustomAttributeDataTypeFloat  = "float"
	CustomAttributeDataTypeDate   = "date"
	CustomAttributeDataTypeString = "string"
)

// CustomAttribute visibility constants
const (
	CustomAttributeVisibilityEveryone      = "everyone"
	CustomAttributeVisibilityAdministrator = "administrator"
	CustomAttributeVisibilityHidden        = "hidden"
)

// CustomAttribute defines configurable attributes that can be assigned to conversations or clients.
// This model allows administrators to create dynamic form fields with validation that can be
// used during conversation or client creation/modification. The system validates data types and
// stores custom attribute values as JSON in the target entity's custom_fields or data columns.
//
// Usage:
// - Create custom attributes via admin interface
// - Use in conversation/client creation APIs by passing custom attribute values
// - System validates according to data_type and validation rules
// - Values are automatically cast to correct types and stored as JSON
//
// Example:
// 1. Create CustomAttribute: scope="conversation", name="priority_level", data_type="int", validation="min:1,max:5"
// 2. When creating conversation, pass: {"priority_level": 3}
// 3. System validates (int between 1-5) and stores in conversation.custom_fields as {"priority_level": 3}
type CustomAttribute struct {
	Scope       string    `gorm:"column:scope;size:15;not null;primaryKey;check:scope IN ('client','conversation')" json:"scope"`
	Name        string    `gorm:"column:name;size:100;not null;primaryKey;check:name REGEXP '^[a-z_]+$'" json:"name"`
	DataType    string    `gorm:"column:data_type;size:20;not null;check:data_type IN ('int','float','date','string')" json:"data_type"`
	Validation  *string   `gorm:"column:validation;size:500" json:"validation"`
	Title       string    `gorm:"column:title;size:255;not null" json:"title"`
	Description *string   `gorm:"column:description;type:text" json:"description"`
	Visibility  string    `gorm:"column:visibility;size:20;not null;check:visibility IN ('everyone','administrator','hidden')" json:"visibility"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	restify.API
}

func (CustomAttribute) TableName() string {
	return "custom_attributes"
}
