package swagger

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// OpenAPI represents the root OpenAPI 3.0 specification
type OpenAPI struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers"`
	Paths      map[string]PathItem   `json:"paths"`
	Components Components            `json:"components"`
	Security   []map[string][]string `json:"security,omitempty"`
}

type Info struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Version     string  `json:"version"`
	Contact     Contact `json:"contact"`
}

type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Components struct {
	Schemas         map[string]Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty"`
}

type Schema struct {
	Type                 string            `json:"type,omitempty"`
	Format               string            `json:"format,omitempty"`
	Properties           map[string]Schema `json:"properties,omitempty"`
	Items                *Schema           `json:"items,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Description          string            `json:"description,omitempty"`
	Example              interface{}       `json:"example,omitempty"`
	Ref                  string            `json:"$ref,omitempty"`
	AdditionalProperties interface{}       `json:"additionalProperties,omitempty"`
	Enum                 []interface{}     `json:"enum,omitempty"`
}

// GenerateOpenAPI generates the complete OpenAPI 3.0 specification
func GenerateOpenAPI() *OpenAPI {
	spec := &OpenAPI{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       "Homa API",
			Description: "Intelligent support system API built with Evo v2 framework",
			Version:     "1.0.0",
			Contact: Contact{
				Name:  "Homa Support",
				Email: "support@homa.example.com",
				URL:   "https://homa.example.com",
			},
		},
		Servers: []Server{
			{
				URL:         "http://localhost:8000",
				Description: "Development server",
			},
		},
		Paths: make(map[string]PathItem),
		Components: Components{
			Schemas: make(map[string]Schema),
			SecuritySchemes: map[string]SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
					Description:  "JWT Bearer token authentication",
				},
			},
		},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}

	// Generate schemas for all models
	generateSchemas(spec)

	// Generate API paths
	generatePaths(spec)

	return spec
}

// generateSchemas creates OpenAPI schemas for all models
func generateSchemas(spec *OpenAPI) {
	// Auth models
	spec.Components.Schemas["User"] = generateSchemaFromStruct(reflect.TypeOf(auth.User{}))
	spec.Components.Schemas["UserLoginHistory"] = generateSchemaFromStruct(reflect.TypeOf(auth.UserLoginHistory{}))

	// Core models
	spec.Components.Schemas["Client"] = generateSchemaFromStruct(reflect.TypeOf(models.Client{}))
	spec.Components.Schemas["ClientExternalID"] = generateSchemaFromStruct(reflect.TypeOf(models.ClientExternalID{}))
	spec.Components.Schemas["Department"] = generateSchemaFromStruct(reflect.TypeOf(models.Department{}))
	spec.Components.Schemas["Channel"] = generateSchemaFromStruct(reflect.TypeOf(models.Channel{}))
	spec.Components.Schemas["Conversation"] = generateSchemaFromStruct(reflect.TypeOf(models.Conversation{}))
	spec.Components.Schemas["Message"] = generateSchemaFromStruct(reflect.TypeOf(models.Message{}))
	spec.Components.Schemas["Tag"] = generateSchemaFromStruct(reflect.TypeOf(models.Tag{}))

	// Request/Response schemas
	spec.Components.Schemas["LoginRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"email": {
				Type:        "string",
				Format:      "email",
				Description: "User email address",
				Example:     "admin@example.com",
			},
			"password": {
				Type:        "string",
				Description: "User password",
				Example:     "secret123",
			},
		},
		Required: []string{"email", "password"},
	}

	spec.Components.Schemas["LoginResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"access_token": {
				Type:        "string",
				Description: "JWT access token",
				Example:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			},
			"refresh_token": {
				Type:        "string",
				Description: "JWT refresh token",
				Example:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			},
			"expires_in": {
				Type:        "integer",
				Description: "Token expiration time in seconds",
				Example:     86400,
			},
			"user": {
				Ref: "#/components/schemas/User",
			},
		},
	}

	spec.Components.Schemas["RefreshRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"refresh_token": {
				Type:        "string",
				Description: "JWT refresh token",
				Example:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			},
		},
		Required: []string{"refresh_token"},
	}

	spec.Components.Schemas["ErrorResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"error": {
				Type:        "string",
				Description: "Error code",
				Example:     "invalid_credentials",
			},
			"message": {
				Type:        "string",
				Description: "Error message",
				Example:     "Invalid email or password",
			},
		},
	}

	spec.Components.Schemas["SuccessResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"success": {
				Type:        "boolean",
				Description: "Success status",
				Example:     true,
			},
			"data": {
				Description: "Response data",
			},
		},
	}

	// Agent API specific schemas
	spec.Components.Schemas["UnreadConversationsResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"data": {
				Type:  "array",
				Items: &Schema{Ref: "#/components/schemas/Conversation"},
			},
		},
	}

	spec.Components.Schemas["UnreadConversationsCountResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"count": {
				Type:        "integer",
				Description: "Number of unread conversations",
				Example:     5,
			},
		},
	}

	spec.Components.Schemas["PaginatedConversationsResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"data": {
				Type:  "array",
				Items: &Schema{Ref: "#/components/schemas/Conversation"},
			},
			"total": {
				Type:        "integer",
				Description: "Total number of tickets",
				Example:     100,
			},
			"page": {
				Type:        "integer",
				Description: "Current page number",
				Example:     1,
			},
			"limit": {
				Type:        "integer",
				Description: "Items per page",
				Example:     20,
			},
		},
	}

	spec.Components.Schemas["ChangeConversationStatusRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"status": {
				Type:        "string",
				Description: "New conversation status",
				Enum:        []interface{}{"new", "wait_for_agent", "in_progress", "wait_for_user", "on_hold", "resolved", "closed", "unresolved", "spam"},
				Example:     "in_progress",
			},
		},
		Required: []string{"status"},
	}

	spec.Components.Schemas["ReplyToConversationRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"message": {
				Type:        "string",
				Description: "Reply message content",
				Example:     "Hello! I can help you with this issue.",
			},
		},
		Required: []string{"message"},
	}

	spec.Components.Schemas["AssignConversationRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"user_id": {
				Type:        "string",
				Format:      "uuid",
				Description: "User ID to assign ticket to",
				Example:     "123e4567-e89b-12d3-a456-426614174000",
			},
			"department_id": {
				Type:        "integer",
				Description: "Department ID to assign ticket to",
				Example:     1,
			},
		},
	}

	spec.Components.Schemas["ChangeConversationDepartmentRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"department_id": {
				Type:        "integer",
				Description: "New department ID",
				Example:     1,
			},
		},
		Required: []string{"department_id"},
	}

	spec.Components.Schemas["TagConversationRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"tag_ids": {
				Type: "array",
				Items: &Schema{
					Type: "integer",
				},
				Description: "Array of existing tag IDs",
				Example:     []interface{}{1, 2, 3},
			},
			"tag_names": {
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
				Description: "Array of tag names (creates if not exists)",
				Example:     []interface{}{"bug", "urgent", "new-feature"},
			},
		},
	}

	spec.Components.Schemas["TagConversationResponse"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"message": {
				Type:        "string",
				Description: "Success message",
				Example:     "Ticket tags updated successfully",
			},
			"total_tags": {
				Type:        "integer",
				Description: "Total number of tags assigned",
				Example:     3,
			},
			"created_tags": {
				Type:        "array",
				Items:       &Schema{Ref: "#/components/schemas/Tag"},
				Description: "Tags that were created during this operation",
			},
		},
	}

	spec.Components.Schemas["CreateTagRequest"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"name": {
				Type:        "string",
				Description: "Tag name",
				Example:     "urgent",
			},
		},
		Required: []string{"name"},
	}

	spec.Components.Schemas["ConversationAssignment"] = Schema{
		Type: "object",
		Properties: map[string]Schema{
			"id": {
				Type:        "integer",
				Description: "Assignment ID",
				Example:     1,
			},
			"conversation_id": {
				Type:        "integer",
				Description: "Ticket ID",
				Example:     123,
			},
			"user_id": {
				Type:        "string",
				Format:      "uuid",
				Description: "Assigned user ID",
				Example:     "123e4567-e89b-12d3-a456-426614174000",
			},
			"department_id": {
				Type:        "integer",
				Description: "Assigned department ID",
				Example:     1,
			},
		},
	}
}

// generatePaths creates OpenAPI paths for all endpoints
func generatePaths(spec *OpenAPI) {
	// Authentication endpoints
	spec.Paths["/api/auth/login"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Authentication"},
			Summary:     "User login",
			Description: "Authenticate user with email and password",
			OperationID: "loginUser",
			RequestBody: &RequestBody{
				Description: "Login credentials",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/LoginRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Login successful",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/LoginResponse"},
								},
							},
						},
					},
				},
				"401": {
					Description: "Invalid credentials",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required for login
		},
	}

	spec.Paths["/api/auth/refresh"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Authentication"},
			Summary:     "Refresh JWT token",
			Description: "Get new access token using refresh token",
			OperationID: "refreshToken",
			RequestBody: &RequestBody{
				Description: "Refresh token",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/RefreshRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Token refreshed successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/LoginResponse"},
								},
							},
						},
					},
				},
				"401": {
					Description: "Invalid refresh token",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required for refresh
		},
	}

	// System endpoints
	spec.Paths["/health"] = PathItem{
		Get: &Operation{
			Tags:        []string{"System"},
			Summary:     "Health check",
			Description: "Check if the API is healthy and running",
			OperationID: "getHealth",
			Responses: map[string]Response{
				"200": {
					Description: "API is healthy",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Type: "string", Example: "ok"},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required for health check
		},
	}

	spec.Paths["/uptime"] = PathItem{
		Get: &Operation{
			Tags:        []string{"System"},
			Summary:     "Get uptime",
			Description: "Get server uptime in seconds",
			OperationID: "getUptime",
			Responses: map[string]Response{
				"200": {
					Description: "Server uptime",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "object",
										Properties: map[string]Schema{
											"uptime": {Type: "integer", Example: 3600},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required for uptime
		},
	}

	// System endpoints
	spec.Paths["/api/system/departments"] = PathItem{
		Get: &Operation{
			Tags:        []string{"System"},
			Summary:     "Get departments",
			Description: "Get a list of all departments",
			OperationID: "getDepartments",
			Responses: map[string]Response{
				"200": {
					Description: "List of departments",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type:  "array",
										Items: &Schema{Ref: "#/components/schemas/Department"},
									},
									"meta": {
										Type: "object",
										Properties: map[string]Schema{
											"count": {Type: "integer"},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required
		},
	}

	spec.Paths["/api/system/ticket-status"] = PathItem{
		Get: &Operation{
			Tags:        []string{"System"},
			Summary:     "Get conversation statuses",
			Description: "Get a list of all available conversation statuses",
			OperationID: "getTicketStatuses",
			Responses: map[string]Response{
				"200": {
					Description: "List of conversation statuses",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "array",
										Items: &Schema{
											Type: "object",
											Properties: map[string]Schema{
												"value":       {Type: "string", Example: "new"},
												"label":       {Type: "string", Example: "New"},
												"description": {Type: "string", Example: "Newly created ticket awaiting initial review"},
											},
										},
									},
									"meta": {
										Type: "object",
										Properties: map[string]Schema{
											"count": {Type: "integer"},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required
		},
	}

	// Generate OAuth endpoints
	generateOAuthPaths(spec)

	// Generate Profile endpoints
	generateProfilePaths(spec)

	// Generate Client Ticket endpoints
	generateClientTicketPaths(spec)

	// Generate Admin Ticket endpoints
	generateAdminTicketPaths(spec)

	// Generate Agent API endpoints
	generateAgentPaths(spec)
}

// generateOAuthPaths creates OAuth-related API paths
func generateOAuthPaths(spec *OpenAPI) {
	// GET /api/auth/oauth/providers
	spec.Paths["/api/auth/oauth/providers"] = PathItem{
		Get: &Operation{
			Tags:        []string{"OAuth"},
			Summary:     "Get OAuth providers",
			Description: "Get a list of available OAuth providers",
			OperationID: "getOAuthProviders",
			Responses: map[string]Response{
				"200": {
					Description: "List of OAuth providers",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "array",
										Items: &Schema{
											Type: "object",
											Properties: map[string]Schema{
												"provider":     {Type: "string", Example: "google"},
												"name":         {Type: "string", Example: "Google"},
												"enabled":      {Type: "boolean", Example: true},
												"redirect_uri": {Type: "string", Example: "http://localhost:8000/auth/oauth/google/callback"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required
		},
	}

	// GET /api/auth/oauth/google
	spec.Paths["/api/auth/oauth/google"] = PathItem{
		Get: &Operation{
			Tags:        []string{"OAuth"},
			Summary:     "Start Google OAuth login",
			Description: "Redirects to Google OAuth consent page",
			OperationID: "googleOAuthLogin",
			Parameters: []Parameter{
				{
					Name:        "redirect_url",
					In:          "query",
					Required:    true,
					Description: "URL to redirect after OAuth completion",
					Schema:      &Schema{Type: "string", Example: "/static/login.html"},
				},
				{
					Name:        "format",
					In:          "query",
					Description: "Response format (json for API)",
					Schema:      &Schema{Type: "string", Example: "json"},
				},
			},
			Responses: map[string]Response{
				"302": {Description: "Redirect to Google OAuth"},
				"200": {
					Description: "OAuth URL (when format=json)",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "object",
										Properties: map[string]Schema{
											"auth_url": {Type: "string"},
											"state":    {Type: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required
		},
	}

	// GET /api/auth/oauth/microsoft
	spec.Paths["/api/auth/oauth/microsoft"] = PathItem{
		Get: &Operation{
			Tags:        []string{"OAuth"},
			Summary:     "Start Microsoft OAuth login",
			Description: "Redirects to Microsoft OAuth consent page",
			OperationID: "microsoftOAuthLogin",
			Parameters: []Parameter{
				{
					Name:        "redirect_url",
					In:          "query",
					Required:    true,
					Description: "URL to redirect after OAuth completion",
					Schema:      &Schema{Type: "string", Example: "/static/login.html"},
				},
				{
					Name:        "format",
					In:          "query",
					Description: "Response format (json for API)",
					Schema:      &Schema{Type: "string", Example: "json"},
				},
			},
			Responses: map[string]Response{
				"302": {Description: "Redirect to Microsoft OAuth"},
				"200": {
					Description: "OAuth URL (when format=json)",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "object",
										Properties: map[string]Schema{
											"auth_url": {Type: "string"},
											"state":    {Type: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required
		},
	}
}

// generateProfilePaths creates profile-related API paths
func generateProfilePaths(spec *OpenAPI) {
	// GET /api/auth/profile
	spec.Paths["/api/auth/profile"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Profile"},
			Summary:     "Get user profile",
			Description: "Get the profile of the currently authenticated user",
			OperationID: "getUserProfile",
			Responses: map[string]Response{
				"200": {
					Description: "User profile",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "object",
										Properties: map[string]Schema{
											"user": {Ref: "#/components/schemas/User"},
											"departments": {
												Type:  "array",
												Items: &Schema{Ref: "#/components/schemas/Department"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Put: &Operation{
			Tags:        []string{"Profile"},
			Summary:     "Update user profile",
			Description: "Update the profile of the currently authenticated user",
			OperationID: "updateUserProfile",
			RequestBody: &RequestBody{
				Description: "Profile update data",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]Schema{
								"name":         {Type: "string"},
								"last_name":    {Type: "string"},
								"display_name": {Type: "string"},
								"avatar":       {Type: "string"},
								"password":     {Type: "string"},
							},
						},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Profile updated successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/User"},
									"message": {Type: "string", Example: "Profile updated successfully"},
								},
							},
						},
					},
				},
			},
		},
	}

	// POST /api/auth/api-key
	spec.Paths["/api/auth/api-key"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Profile"},
			Summary:     "Generate API key",
			Description: "Generate a new API key for the authenticated user. This will replace any existing API key.",
			OperationID: "generateAPIKey",
			Responses: map[string]Response{
				"200": {
					Description: "API key generated successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "object",
										Properties: map[string]Schema{
											"api_key": {Type: "string", Example: "homa_1234567890abcdef..."},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Delete: &Operation{
			Tags:        []string{"Profile"},
			Summary:     "Revoke API key",
			Description: "Revoke the current API key for the authenticated user",
			OperationID: "revokeAPIKey",
			Responses: map[string]Response{
				"200": {
					Description: "API key revoked successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"message": {Type: "string", Example: "API key revoked successfully"},
								},
							},
						},
					},
				},
			},
		},
	}
}

// generateClientTicketPaths creates client ticket API paths
func generateClientTicketPaths(spec *OpenAPI) {
	// PUT /api/client/tickets
	spec.Paths["/api/client/tickets"] = PathItem{
		Put: &Operation{
			Tags:        []string{"Client Tickets"},
			Summary:     "Create a new ticket",
			Description: "Create a new ticket with optional custom attributes",
			OperationID: "createTicket",
			RequestBody: &RequestBody{
				Description: "Ticket data",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]Schema{
								"title":         {Type: "string", Example: "Login issue"},
								"client_name":   {Type: "string", Example: "John Doe"},
								"client_email":  {Type: "string", Example: "john@example.com"},
								"client_id":     {Type: "string", Format: "uuid"},
								"department_id": {Type: "integer", Example: 1},
								"status":        {Type: "string", Enum: []interface{}{"new", "wait_for_agent", "in_progress", "wait_for_user", "on_hold", "resolved", "closed", "unresolved", "spam"}},
								"priority":      {Type: "string", Enum: []interface{}{"low", "medium", "high", "urgent"}},
								"message":       {Type: "string", Example: "Initial ticket description"},
								"parameters":    {Type: "object", AdditionalProperties: true},
							},
							Required: []string{"title", "status", "priority"},
						},
					},
				},
			},
			Responses: map[string]Response{
				"201": {
					Description: "Ticket created successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "object",
										Properties: map[string]Schema{
											"id":       {Type: "integer"},
											"secret":   {Type: "string"},
											"title":    {Type: "string"},
											"status":   {Type: "string"},
											"priority": {Type: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required for client
		},
	}

	// GET /api/client/tickets/{ticket_id}/{secret}
	spec.Paths["/api/client/tickets/{ticket_id}/{secret}"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Client Tickets"},
			Summary:     "Get ticket with messages",
			Description: "Get ticket details and messages using secret authentication",
			OperationID: "getTicketWithSecret",
			Parameters: []Parameter{
				{Name: "ticket_id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
				{Name: "secret", In: "path", Required: true, Schema: &Schema{Type: "string"}},
				{Name: "offset", In: "query", Schema: &Schema{Type: "integer", Example: 0}},
				{Name: "limit", In: "query", Schema: &Schema{Type: "integer", Example: 20}},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket with messages",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type: "object",
										Properties: map[string]Schema{
											"ticket":   {Ref: "#/components/schemas/Conversation"},
											"messages": {Type: "array", Items: &Schema{Ref: "#/components/schemas/Message"}},
										},
									},
									"meta": {
										Type: "object",
										Properties: map[string]Schema{
											"total":  {Type: "integer"},
											"offset": {Type: "integer"},
											"count":  {Type: "integer"},
											"limit":  {Type: "integer"},
										},
									},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // Secret-based auth
		},
		Delete: &Operation{
			Tags:        []string{"Client Tickets"},
			Summary:     "Close ticket",
			Description: "Close ticket using secret authentication",
			OperationID: "closeTicketWithSecret",
			Parameters: []Parameter{
				{Name: "ticket_id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
				{Name: "secret", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket closed successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/Conversation"},
									"message": {Type: "string", Example: "Ticket closed successfully"},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // Secret-based auth
		},
	}

	// POST /api/client/tickets/{ticket_id}/{secret}/messages
	spec.Paths["/api/client/tickets/{ticket_id}/{secret}/messages"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Client Tickets"},
			Summary:     "Add message to ticket",
			Description: "Add a message to ticket using secret authentication",
			OperationID: "addClientMessage",
			Parameters: []Parameter{
				{Name: "ticket_id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
				{Name: "secret", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			RequestBody: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]Schema{
								"message": {Type: "string", Example: "Additional information about the issue"},
							},
							Required: []string{"message"},
						},
					},
				},
			},
			Responses: map[string]Response{
				"201": {
					Description: "Message added successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/Message"},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // Secret-based auth
		},
	}

	// PUT /api/client/upsert
	spec.Paths["/api/client/upsert"] = PathItem{
		Put: &Operation{
			Tags:        []string{"Clients"},
			Summary:     "Upsert client",
			Description: "Create or return existing client",
			OperationID: "upsertClient",
			RequestBody: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]Schema{
								"type":  {Type: "string", Enum: []interface{}{"email", "phone", "whatsapp", "slack", "telegram", "web", "chat"}},
								"value": {Type: "string"},
							},
							Required: []string{"type", "value"},
						},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Client upserted successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/Client"},
								},
							},
						},
					},
				},
			},
			Security: []map[string][]string{}, // No auth required
		},
	}
}

// generateAdminTicketPaths creates admin ticket API paths
func generateAdminTicketPaths(spec *OpenAPI) {
	// POST /api/admin/tickets/{ticket_id}/assign/user
	spec.Paths["/api/admin/tickets/{ticket_id}/assign/user"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Admin - Ticket Assignments"},
			Summary:     "Assign ticket to user",
			Description: "Assign a ticket to a specific user",
			OperationID: "assignTicketToUser",
			Parameters: []Parameter{
				{Name: "ticket_id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
			},
			RequestBody: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]Schema{
								"user_id": {Type: "string", Format: "uuid"},
							},
							Required: []string{"user_id"},
						},
					},
				},
			},
			Responses: map[string]Response{
				"201": {
					Description: "Ticket assigned successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/ConversationAssignment"},
								},
							},
						},
					},
				},
			},
		},
	}

	// POST /api/admin/tickets/{ticket_id}/assign/department
	spec.Paths["/api/admin/tickets/{ticket_id}/assign/department"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Admin - Ticket Assignments"},
			Summary:     "Assign ticket to department",
			Description: "Assign a ticket to a department",
			OperationID: "assignTicketToDepartment",
			Parameters: []Parameter{
				{Name: "ticket_id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
			},
			RequestBody: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]Schema{
								"department_id": {Type: "integer"},
							},
							Required: []string{"department_id"},
						},
					},
				},
			},
			Responses: map[string]Response{
				"201": {
					Description: "Ticket assigned successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/ConversationAssignment"},
								},
							},
						},
					},
				},
			},
		},
	}

	// DELETE /api/admin/tickets/{ticket_id}/unassign
	spec.Paths["/api/admin/tickets/{ticket_id}/unassign"] = PathItem{
		Delete: &Operation{
			Tags:        []string{"Admin - Ticket Assignments"},
			Summary:     "Unassign ticket",
			Description: "Remove all assignments from a ticket",
			OperationID: "unassignTicket",
			Parameters: []Parameter{
				{Name: "ticket_id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket unassigned successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"message": {Type: "string", Example: "Ticket unassigned successfully"},
								},
							},
						},
					},
				},
			},
		},
	}

	// GET /api/admin/tickets/{ticket_id}/assignments
	spec.Paths["/api/admin/tickets/{ticket_id}/assignments"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Admin - Ticket Assignments"},
			Summary:     "Get ticket assignments",
			Description: "Get all assignments for a specific ticket",
			OperationID: "getConversationAssignments",
			Parameters: []Parameter{
				{Name: "ticket_id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket assignments",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data": {
										Type:  "array",
										Items: &Schema{Ref: "#/components/schemas/ConversationAssignment"},
									},
									"meta": {
										Type: "object",
										Properties: map[string]Schema{
											"count": {Type: "integer"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// generateSchemaFromStruct creates an OpenAPI schema from a Go struct type
func generateSchemaFromStruct(t reflect.Type) Schema {
	return generateSchemaFromStructWithVisited(t, make(map[reflect.Type]bool))
}

// generateSchemaFromStructWithVisited creates an OpenAPI schema from a Go struct type with circular reference protection
func generateSchemaFromStructWithVisited(t reflect.Type, visited map[reflect.Type]bool) Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check for circular reference
	if visited[t] {
		return Schema{
			Type:        "object",
			Description: "Circular reference to " + t.Name(),
		}
	}

	// Mark as visited
	visited[t] = true
	defer func() { delete(visited, t) }()

	schema := Schema{
		Type:       "object",
		Properties: make(map[string]Schema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip embedded types like restify.API
		if field.Anonymous {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Skip omitempty relationship fields to avoid circular references
		if jsonTag != "" && strings.Contains(jsonTag, "omitempty") {
			continue
		}

		// Parse JSON tag
		jsonName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				jsonName = parts[0]
			}
		}

		// Generate schema for field
		fieldSchema := generateSchemaFromTypeWithVisited(field.Type, visited)

		// Add description from struct tags or field name
		if gormTag := field.Tag.Get("gorm"); gormTag != "" {
			if strings.Contains(gormTag, "not null") && !strings.Contains(gormTag, "primaryKey") {
				schema.Required = append(schema.Required, jsonName)
			}
		}

		schema.Properties[jsonName] = fieldSchema
	}

	return schema
}

// generateSchemaFromType creates an OpenAPI schema from a Go type
func generateSchemaFromType(t reflect.Type) Schema {
	return generateSchemaFromTypeWithVisited(t, make(map[reflect.Type]bool))
}

// generateSchemaFromTypeWithVisited creates an OpenAPI schema from a Go type with circular reference protection
func generateSchemaFromTypeWithVisited(t reflect.Type, visited map[reflect.Type]bool) Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t {
	case reflect.TypeOf(uuid.UUID{}):
		return Schema{
			Type:    "string",
			Format:  "uuid",
			Example: "123e4567-e89b-12d3-a456-426614174000",
		}
	case reflect.TypeOf(time.Time{}):
		return Schema{
			Type:    "string",
			Format:  "date-time",
			Example: "2025-01-01T12:00:00Z",
		}
	case reflect.TypeOf(datatypes.JSON{}):
		return Schema{
			Type:                 "object",
			AdditionalProperties: true,
			Example:              map[string]interface{}{"key": "value"},
		}
	}

	switch t.Kind() {
	case reflect.String:
		return Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Schema{Type: "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return Schema{Type: "number"}
	case reflect.Bool:
		return Schema{Type: "boolean"}
	case reflect.Slice:
		return Schema{
			Type:  "array",
			Items: &[]Schema{generateSchemaFromTypeWithVisited(t.Elem(), visited)}[0],
		}
	case reflect.Struct:
		return generateSchemaFromStructWithVisited(t, visited)
	default:
		return Schema{Type: "object"}
	}
}

// generateAgentPaths creates OpenAPI paths for agent endpoints
func generateAgentPaths(spec *OpenAPI) {
	// Agent authentication security requirement
	agentSecurity := []map[string][]string{{"bearerAuth": {}}}

	// GET /api/agent/tickets/unread - Get unread conversations
	spec.Paths["/api/agent/tickets/unread"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Get unread conversations",
			Description: "Get tickets that are unread (new, wait_for_agent, in_progress) for the authenticated agent",
			OperationID: "getUnreadTickets",
			Responses: map[string]Response{
				"200": {
					Description: "List of unread conversations",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/UnreadConversationsResponse"},
								},
							},
						},
					},
				},
				"401": {
					Description: "Unauthorized - Authentication required",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
				"403": {
					Description: "Forbidden - Only agents and administrators can access",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// GET /api/agent/tickets/unread/count - Get unread conversations count
	spec.Paths["/api/agent/tickets/unread/count"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Get unread conversations count",
			Description: "Get count of unread conversations for the authenticated agent",
			OperationID: "getUnreadTicketsCount",
			Responses: map[string]Response{
				"200": {
					Description: "Count of unread conversations",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/UnreadConversationsCountResponse"},
								},
							},
						},
					},
				},
				"401": {
					Description: "Unauthorized",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// GET /api/agent/tickets - Get paginated conversation list
	spec.Paths["/api/agent/tickets"] = PathItem{
		Get: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Get conversation list",
			Description: "Get paginated list of tickets accessible to the authenticated agent",
			OperationID: "getTicketList",
			Parameters: []Parameter{
				{
					Name:        "page",
					In:          "query",
					Description: "Page number (default: 1)",
					Schema:      &Schema{Type: "integer", Example: 1},
				},
				{
					Name:        "limit",
					In:          "query",
					Description: "Items per page (default: 20, max: 100)",
					Schema:      &Schema{Type: "integer", Example: 20},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Paginated list of tickets",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/PaginatedConversationsResponse"},
								},
							},
						},
					},
				},
				"401": {
					Description: "Unauthorized",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// PUT /api/agent/tickets/{id}/status - Change conversation status
	spec.Paths["/api/agent/tickets/{id}/status"] = PathItem{
		Put: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Change conversation status",
			Description: "Update the status of a specific ticket",
			OperationID: "changeTicketStatus",
			Parameters: []Parameter{
				{
					Name:        "id",
					In:          "path",
					Description: "Ticket ID",
					Required:    true,
					Schema:      &Schema{Type: "integer", Example: 1},
				},
			},
			RequestBody: &RequestBody{
				Description: "New conversation status",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/ChangeConversationStatusRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket status updated successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/Conversation",
							},
						},
					},
				},
				"400": {
					Description: "Invalid input",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
				"404": {
					Description: "Ticket not found or access denied",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// POST /api/agent/tickets/{id}/reply - Reply to ticket
	spec.Paths["/api/agent/tickets/{id}/reply"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Reply to ticket",
			Description: "Add a message reply to a specific ticket",
			OperationID: "replyToTicket",
			Parameters: []Parameter{
				{
					Name:        "id",
					In:          "path",
					Description: "Ticket ID",
					Required:    true,
					Schema:      &Schema{Type: "integer", Example: 1},
				},
			},
			RequestBody: &RequestBody{
				Description: "Reply message",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/ReplyToConversationRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Reply sent successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/Message",
							},
						},
					},
				},
				"400": {
					Description: "Invalid input",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
				"404": {
					Description: "Ticket not found or access denied",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// PUT /api/agent/tickets/{id}/assign - Assign ticket
	spec.Paths["/api/agent/tickets/{id}/assign"] = PathItem{
		Put: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Assign ticket",
			Description: "Assign a ticket to a user or department",
			OperationID: "assignTicket",
			Parameters: []Parameter{
				{
					Name:        "id",
					In:          "path",
					Description: "Ticket ID",
					Required:    true,
					Schema:      &Schema{Type: "integer", Example: 1},
				},
			},
			RequestBody: &RequestBody{
				Description: "Assignment details (provide either user_id or department_id)",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/AssignConversationRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket assigned successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/ConversationAssignment",
							},
						},
					},
				},
				"400": {
					Description: "Invalid input",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
				"404": {
					Description: "Ticket not found or access denied",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// PUT /api/agent/tickets/{id}/department - Change ticket department
	spec.Paths["/api/agent/tickets/{id}/department"] = PathItem{
		Put: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Change ticket department",
			Description: "Move a ticket to a different department",
			OperationID: "changeTicketDepartment",
			Parameters: []Parameter{
				{
					Name:        "id",
					In:          "path",
					Description: "Ticket ID",
					Required:    true,
					Schema:      &Schema{Type: "integer", Example: 1},
				},
			},
			RequestBody: &RequestBody{
				Description: "New department ID",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/ChangeConversationDepartmentRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket department updated successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/Conversation",
							},
						},
					},
				},
				"400": {
					Description: "Invalid input",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
				"404": {
					Description: "Ticket not found or access denied",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// PUT /api/agent/tickets/{id}/tags - Tag ticket
	spec.Paths["/api/agent/tickets/{id}/tags"] = PathItem{
		Put: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Tag ticket",
			Description: "Add/remove tags from a ticket. Can use existing tag IDs or tag names (creates if not exists)",
			OperationID: "tagTicket",
			Parameters: []Parameter{
				{
					Name:        "id",
					In:          "path",
					Description: "Ticket ID",
					Required:    true,
					Schema:      &Schema{Type: "integer", Example: 1},
				},
			},
			RequestBody: &RequestBody{
				Description: "Tag IDs and/or tag names (at least one must be provided)",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/TagConversationRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Ticket tags updated successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]Schema{
									"success": {Type: "boolean", Example: true},
									"data":    {Ref: "#/components/schemas/TagConversationResponse"},
								},
							},
						},
					},
				},
				"400": {
					Description: "Invalid input",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
				"404": {
					Description: "Ticket not found or access denied",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}

	// POST /api/agent/tags - Create tag
	spec.Paths["/api/agent/tags"] = PathItem{
		Post: &Operation{
			Tags:        []string{"Agent"},
			Summary:     "Create tag",
			Description: "Create a new tag",
			OperationID: "createTag",
			RequestBody: &RequestBody{
				Description: "Tag details",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/CreateTagRequest"},
					},
				},
			},
			Responses: map[string]Response{
				"201": {
					Description: "Tag created successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Ref: "#/components/schemas/Tag",
							},
						},
					},
				},
				"400": {
					Description: "Invalid input",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Security: agentSecurity,
		},
	}
}

// ToJSON converts the OpenAPI spec to JSON
func (spec *OpenAPI) ToJSON() ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}
