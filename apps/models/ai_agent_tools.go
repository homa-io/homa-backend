package models

import (
	"time"

	"gorm.io/datatypes"
)

// HTTP Method constants
const (
	ToolMethodGET    = "GET"
	ToolMethodPOST   = "POST"
	ToolMethodPUT    = "PUT"
	ToolMethodPATCH  = "PATCH"
	ToolMethodDELETE = "DELETE"
)

// Body type constants
const (
	ToolBodyTypeJSON      = "JSON"
	ToolBodyTypeFormValue = "FormValue"
	ToolBodyTypeCustom    = "Custom"
)

// Authorization type constants
const (
	ToolAuthTypeNone     = "None"
	ToolAuthTypeBearer   = "Bearer"
	ToolAuthTypeBasic    = "BasicAuth"
	ToolAuthTypeAPIKey   = "APIKey"
)

// Value type constants for params
const (
	ToolParamValueTypeVariable = "Variable"
	ToolParamValueTypeConstant = "Constant"
	ToolParamValueTypeByModel  = "ByModel"
)

// Data type constants for params
const (
	ToolParamDataTypeString = "string"
	ToolParamDataTypeInt    = "int"
	ToolParamDataTypeFloat  = "float"
	ToolParamDataTypeBool   = "bool"
)

// AIAgentTool represents an API endpoint tool that an AI agent can use
type AIAgentTool struct {
	ID                   uint           `gorm:"column:id;primaryKey" json:"id"`
	AIAgentID            uint           `gorm:"column:ai_agent_id;not null;index" json:"ai_agent_id"`
	Name                 string         `gorm:"column:name;size:255;not null" json:"name"`
	Description          string         `gorm:"column:description;type:text" json:"description"`
	Endpoint             string         `gorm:"column:endpoint;size:500;not null" json:"endpoint"`
	Method               string         `gorm:"column:method;size:10;not null;default:'GET'" json:"method"`
	QueryParams          datatypes.JSON `gorm:"column:query_params;type:json" json:"query_params"`
	HeaderParams         datatypes.JSON `gorm:"column:header_params;type:json" json:"header_params"`
	BodyType             string         `gorm:"column:body_type;size:20" json:"body_type"`
	BodyParams           datatypes.JSON `gorm:"column:body_params;type:json" json:"body_params"`
	AuthorizationType    string         `gorm:"column:authorization_type;size:20;default:'None'" json:"authorization_type"`
	AuthorizationHeader  string         `gorm:"column:authorization_header;size:255" json:"authorization_header"`
	AuthorizationValue   string         `gorm:"column:authorization_value;size:500" json:"authorization_value"`
	ResponseInstructions string         `gorm:"column:response_instructions;type:text" json:"response_instructions"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	AIAgent *AIAgent `gorm:"foreignKey:AIAgentID;references:ID" json:"ai_agent,omitempty"`
}

// TableName specifies the table name for GORM
func (AIAgentTool) TableName() string {
	return "ai_agent_tools"
}

// ToolParam represents a parameter configuration for tool API calls
// This is stored as JSON in the database
type ToolParam struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	ValueType string `json:"value_type"` // Variable, Constant, ByModel
	DataType  string `json:"data_type"`  // string, int, float, bool
	Example   string `json:"example"`
	Required  bool   `json:"required"`
}
