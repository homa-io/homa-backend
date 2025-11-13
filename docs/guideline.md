# Development Guidelines

Technical coding standards for AI agents developing applications in this Evo v2 project.

## Core Architecture

**Modular App System**: Apps are self-contained modules under `/apps/` implementing:
```go
type App struct{}
func (a App) Register() error  // Initialize resources
func (a App) Router() error   // Register routes  
func (a App) WhenReady() error // Post-init tasks
func (a App) Name() string    // App identifier
```

**Entry Point**: Register apps in `main.go`:
```go
apps.Register(system.App{}, models.App{})
```

## App Lifecycle & Initialization

### Critical Rule: Settings Initialization Order
**IMPORTANT**: Never call `settings.Get()` or any config-dependent code in global variable initialization. The settings system is not available during global variable initialization.

**❌ WRONG - Will fail silently or cause runtime errors:**
```go
var JWTSecret = getJWTSecret() // settings.Get() called too early!
var DatabaseURL = settings.Get("DB.URL").String() // WRONG!
```

**✅ CORRECT - Initialize in Register() or WhenReady():**
```go
var JWTSecret []byte // Global declaration only

func (a App) Register() error {
    // Initialize after settings are loaded
    InitializeJWTSecret()
    return nil
}

func InitializeJWTSecret() {
    secret := settings.Get("JWT.SECRET").String() // Now settings are available
    if secret == "" {
        secret = os.Getenv("JWT_SECRET")
    }
    JWTSecret = []byte(secret)
}
```

### App Lifecycle Methods
- **Register()**: Called first, use for model registration, global variable initialization
- **Router()**: Called second, use for route registration  
- **WhenReady()**: Called last, use for post-initialization tasks

### Variable Naming & Shadowing
**CRITICAL**: Avoid variable shadowing, especially in authentication code. Variable shadowing can cause silent failures and security bugs.

**❌ WRONG - Variable shadowing causes JWT parsing to fail:**
```go
func (u *User) FromRequest(request *evo.Request) evo.UserInterface {
    token := getAuthHeader() // Original token string
    
    // This shadows the original 'token' variable!
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Now 'token' refers to the JWT object, not the string!
        return JWTSecret, nil
    })
    // JWT parsing will fail because variable is shadowed
}
```

**✅ CORRECT - Use distinct variable names:**
```go
func (u *User) FromRequest(request *evo.Request) evo.UserInterface {
    authToken := getAuthHeader() // Clear name for auth header
    
    // Use distinct name for JWT token object
    jwtToken, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return JWTSecret, nil
    })
    // Now both variables are distinct and accessible
}
```

## Model Standards

### Required Structure
```go
package models
import (
    "github.com/getevo/evo/v2/lib/db"    // REQUIRED: For database operations
    "github.com/getevo/restify"
    "github.com/google/uuid"      // For UUID fields
    "gorm.io/datatypes"          // For JSON fields  
    "gorm.io/gorm"               // For GORM hooks
    "time"
)

type ModelName struct {
    // Database fields with proper tags...
    ID        uint      `gorm:"primaryKey" json:"id"`
    Name      string    `gorm:"size:255;not null" json:"name"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    
    // Relationships...
    Children []ChildModel `gorm:"foreignKey:ParentID" json:"children,omitempty"`
    
    restify.API                  // REQUIRED: Must be at the END of struct
}
```

### Field Types & Tags

**UUID Primary Keys** (Users, Clients):
```go
ID uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`

// Add BeforeCreate hook for UUID generation
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.ID == (uuid.UUID{}) {
        u.ID = uuid.New()
    }
    return nil
}
```

**String Primary Keys** (Channels):
```go
ID string `gorm:"primaryKey;size:50" json:"id"`
```

**Auto-increment Primary Keys** (Others):
```go
ID uint `gorm:"primaryKey" json:"id"`
```

**Foreign Keys**:
```go
// UUID foreign keys
UserID   uuid.UUID `gorm:"type:char(36);not null;index;fk:users" json:"user_id"`
ClientID uuid.UUID `gorm:"type:char(36);not null;index;fk:clients" json:"client_id"`

// Regular foreign keys  
ChannelID string `gorm:"size:50;not null;index;fk:channels" json:"channel_id"`
TicketID  uint   `gorm:"not null;index;fk:tickets" json:"ticket_id"`

// Optional foreign keys (use pointers)
DepartmentID *uint      `gorm:"index;fk:departments" json:"department_id"`
UserID       *uuid.UUID `gorm:"type:char(36);index;fk:users" json:"user_id"`
```

**Standard Fields**:
```go
Name      string         `gorm:"size:255;not null" json:"name"`
Email     string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
Data      datatypes.JSON `gorm:"type:json" json:"data"`
Enabled   bool           `gorm:"default:1" json:"enabled"`        // MySQL: 1=true, 0=false
CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
```

**Enums with Database Constraints**:
```go
// User type constants
const (
    UserTypeAgent         = "agent"
    UserTypeAdministrator = "administrator"
)

// Ticket status constants  
const (
    TicketStatusNew          = "new"
    TicketStatusWaitForAgent = "wait_for_agent"
    TicketStatusInProgress   = "in_progress"
    TicketStatusWaitForUser  = "wait_for_user"
    TicketStatusOnHold       = "on_hold"
    TicketStatusResolved     = "resolved"
    TicketStatusClosed       = "closed"
    TicketStatusUnresolved   = "unresolved"
    TicketStatusSpam         = "spam"
)

// Ticket priority constants
const (
    TicketPriorityLow    = "low"
    TicketPriorityMedium = "medium"
    TicketPriorityHigh   = "high"
    TicketPriorityUrgent = "urgent"
)

// External ID type constants
const (
    ExternalIDTypeEmail     = "email"
    ExternalIDTypePhone     = "phone"
    ExternalIDTypeWhatsapp  = "whatsapp"
    ExternalIDTypeSlack     = "slack"
    ExternalIDTypeTelegram  = "telegram"
    ExternalIDTypeWeb       = "web"
    ExternalIDTypeChat      = "chat"
)

// OAuth provider constants
const (
    OAuthProviderGoogle    = "google"
    OAuthProviderGithub    = "github"
    OAuthProviderGitLab    = "gitlab"
    OAuthProviderMicrosoft = "microsoft"
)

// Fields with enum constraints
Type     string `gorm:"size:50;not null;check:type IN ('agent','administrator')" json:"type"`
Status   string `gorm:"size:50;not null;check:status IN ('new','wait_for_agent','in_progress','wait_for_user','on_hold','resolved','closed','unresolved','spam')" json:"status"`
Priority string `gorm:"size:50;not null;check:priority IN ('low','medium','high','urgent')" json:"priority"`
Provider string `gorm:"size:50;not null;check:provider IN ('google','github','gitlab','microsoft')" json:"provider"`
```

### GORM Hooks

**UUID Generation** (Required for Users and Clients):
```go
// Add BeforeCreate hook for automatic UUID generation
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.ID == (uuid.UUID{}) {
        u.ID = uuid.New()
    }
    return nil
}

func (c *Client) BeforeCreate(tx *gorm.DB) error {
    if c.ID == (uuid.UUID{}) {
        c.ID = uuid.New()
    }
    return nil
}
```

### Relationships

**One-to-Many**:
```go
// Parent side
Children []ChildModel `gorm:"foreignKey:ParentID;references:ID" json:"children,omitempty"`

// Child side  
ParentID uint        `gorm:"not null;index;fk:parents" json:"parent_id"`
Parent   ParentModel `gorm:"foreignKey:ParentID;references:ID" json:"parent,omitempty"`
```

**Many-to-Many**:
```go
// Both sides specify full relationship
Tags []Tag `gorm:"many2many:ticket_tags;foreignKey:ID;joinForeignKey:TicketID;references:ID;joinReferences:TagID" json:"tags,omitempty"`

// Junction table (always create explicit model)
type TicketTag struct {
    TicketID uint `gorm:"primaryKey;fk:tickets" json:"ticket_id"`
    TagID    uint `gorm:"primaryKey;fk:tags" json:"tag_id"`
    
    Ticket Ticket `gorm:"foreignKey:TicketID;references:ID" json:"ticket,omitempty"`
    Tag    Tag    `gorm:"foreignKey:TagID;references:ID" json:"tag,omitempty"`
    
    restify.API
}
```

### Database Operations

**GORM Access**: Use `db` package for all database operations:
```go
import "github.com/getevo/evo/v2/lib/db"

// Database operations
var user User
db.Where("email = ?", email).First(&user)
db.Create(&user)
db.Save(&user)

// Associations
db.Model(&user).Association("Departments").Find(&departments)
```

### Model Registration
Add ALL models to `/apps/models/app.go`:
```go
func (a App) Register() error {
    db.UseModel(User{})
    db.UseModel(Channel{})
    // ... all models including junction tables
    return nil
}
```

## API Standards

**Request Structure Naming Convention**:
Every API endpoint that accepts a request body (PUT, POST, DELETE, etc.) must have a corresponding struct named `{NameOfAPI}Request`. This convention enables:
- Clear API documentation through struct definition
- Consistent validation patterns
- Type safety for request parsing

Examples:
- `LoginRequest` for login API
- `CreateUserRequest` for user creation API
- `UpdateTicketRequest` for ticket updates
- `DeleteClientRequest` for client deletion

**Request Body Parsing**:
```go
// Define request struct with validation tags
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=6"`
}

func (c Controller) LoginHandler(request *evo.Request) any {
    var loginReq LoginRequest
    if err := request.BodyParser(&loginReq); err != nil {
        request.Status(400)
        return map[string]any{
            "error":   "invalid_request",
            "message": "Invalid request format",
        }
    }
    
    // Optional: Add validation
    if err := validate.Struct(&loginReq); err != nil {
        request.Status(400)
        return map[string]any{
            "error":   "validation_failed",
            "message": err.Error(),
        }
    }
    
    // Process request...
}
```

**Response Handlers**:
```go
func HandlerName(request *evo.Request) any {
    return map[string]any{
        "status": "success",
        "data": responseData,
    }
}
```

**Response Structure Patterns**:
For list/array responses, use direct array types instead of wrapper structs to save JSON space and simplify parsing:

```go
// ❌ WRONG - Inefficient wrapper structure
type UsersListResponse struct {
    Users []User `json:"users"`
}

// ✅ CORRECT - Direct array type
type UsersListResponse []User

// ❌ WRONG - Wrapper with unnecessary nesting
type DepartmentListResponse struct {
    Departments []models.Department `json:"departments"`
}

// ✅ CORRECT - Clean direct array
type DepartmentListResponse []models.Department
```

**Benefits of Direct Array Types**:
- Smaller JSON payload (no wrapper object)
- Easier client-side parsing
- More RESTful response structure
- Less memory usage
- Cleaner API documentation

**Custom Output Responses**:
Use `github.com/getevo/evo/v2/lib/outcome` for custom output instead of manual content-type setting:

```go
import "github.com/getevo/evo/v2/lib/outcome"

// Instead of:
// request.Set("Content-Type", "application/json")
// return string(jsonData)

// Use outcome library methods:
func HandlerName(request *evo.Request) any {
    // For JSON responses
    return outcome.Json(jsonData)
    
    // For other response types
    return outcome.Html(htmlData)
    return outcome.Text(textData)
    return outcome.Xml(xmlData)
}

// For custom content types, use outcome.Response struct:
func OpenAPISpecHandler(request *evo.Request) any {
    spec := GenerateOpenAPI()
    
    jsonData, err := spec.ToJSON()
    if err != nil {
        request.Status(500)
        return map[string]any{
            "error":   "internal_error",
            "message": "Failed to generate OpenAPI specification",
        }
    }

    return outcome.Response{
        ContentType: "application/json",
        Data:        jsonData,
    }
}
```

**Error Responses**:
```go
request.WriteJSON(evo.Map{
    "status": "error", 
    "error": "Error message",
})
```

## Generic Values and Type Handling

The Evo framework uses `generic.Value` for request parameters and query strings, which requires explicit type conversion.

**Query Parameter Handling**:
```go
import "github.com/getevo/evo/v2"

func HandlerName(request *evo.Request) any {
    // Query parameters return generic.Value, must convert to specific types
    id := request.Query("id").String()           // Convert to string
    page := request.Query("page").Int()          // Convert to int
    limit := request.Query("limit").Int64()      // Convert to int64
    enabled := request.Query("enabled").Bool()   // Convert to bool
    
    // Check if query parameter exists
    if request.Query("optional").Exists() {
        optional := request.Query("optional").String()
        // Process optional parameter
    }
    
    // Default values
    pageSize := request.Query("page_size").IntOr(10)  // Default to 10 if not provided
    search := request.Query("search").StringOr("")    // Default to empty string
}
```

**URL Parameter Handling**:
```go
func HandlerName(request *evo.Request) any {
    // URL parameters (from routes like /users/:id) also return generic.Value
    userID := request.Param("id").String()
    
    // For UUID conversion
    uuid, err := uuid.Parse(request.Param("id").String())
    if err != nil {
        request.Status(400)
        return map[string]any{
            "error":   "invalid_uuid",
            "message": "Invalid UUID format",
        }
    }
}
```

**Form Data and Headers**:
```go
func HandlerName(request *evo.Request) any {
    // Form values also return generic.Value
    email := request.FormValue("email").String()
    age := request.FormValue("age").IntOr(0)
    
    // Headers return generic.Value as well
    authHeader := request.Header("Authorization").String()
    contentType := request.Header("Content-Type").StringOr("application/json")
}
```

**Available Conversion Methods**:
```go
// String conversions
.String()           // Convert to string
.StringOr(default)  // Convert to string with default value

// Integer conversions  
.Int()              // Convert to int
.IntOr(default)     // Convert to int with default value
.Int64()            // Convert to int64
.Int64Or(default)   // Convert to int64 with default value

// Boolean conversions
.Bool()             // Convert to bool
.BoolOr(default)    // Convert to bool with default value

// Float conversions
.Float32()          // Convert to float32
.Float64()          // Convert to float64

// Utility methods
.Exists()           // Check if value exists
.IsEmpty()          // Check if value is empty
```

**Common Patterns**:
```go
func UserHandler(request *evo.Request) any {
    // Safe ID parsing with validation
    userID := request.Param("id").String()
    if userID == "" {
        request.Status(400)
        return map[string]any{
            "error":   "missing_id",
            "message": "User ID is required",
        }
    }
    
    // Pagination with defaults
    page := request.Query("page").IntOr(1)
    limit := request.Query("limit").IntOr(10)
    
    // Optional filtering
    var filters map[string]interface{}
    if request.Query("search").Exists() {
        filters["search"] = request.Query("search").String()
    }
    
    // Boolean flags
    includeDeleted := request.Query("include_deleted").BoolOr(false)
}
```

## Evo v2 Framework Controller Patterns

All API handlers must follow the Evo v2 framework patterns for proper request handling and response formatting.

### Required Controller Method Signature

**CRITICAL**: All controller methods MUST use this exact signature:
```go
func (c Controller) MethodName(request *evo.Request) any
```

**❌ WRONG - Do NOT use these patterns:**
```go
func (c Controller) MethodName(ctx evo.Context) error      // Wrong - evo.Context doesn't exist
func (c Controller) MethodName(c *fiber.Ctx) error        // Wrong - not using Evo patterns
func (c Controller) MethodName(w http.ResponseWriter, r *http.Request) // Wrong - not using Evo
```

**✅ CORRECT - Use this pattern:**
```go
func (c Controller) MethodName(request *evo.Request) any {
    // Handler implementation
    return responseData // Must return any type
}
```

### Request Data Access Patterns

**Request Body Parsing**:
```go
func (c Controller) CreateUser(request *evo.Request) any {
    var req CreateUserRequest
    if err := request.BodyParser(&req); err != nil {
        return response.Error(response.ErrInvalidInput)
    }
    // Process request...
}
```

**URL Parameters** (from routes like `/users/:id`):
```go
func (c Controller) GetUser(request *evo.Request) any {
    userID := request.Params("id")
    // Note: request.Params() returns string, not generic.Value
}
```

**Query Parameters** (from URL like `?search=value&page=1`):
```go
func (c Controller) ListUsers(request *evo.Request) any {
    search := request.Query("search").String()      // Convert to string
    page := request.Query("page").IntOr(1)          // Convert to int with default
    enabled := request.Query("enabled").Bool()      // Convert to bool
}
```

**User Authentication**:
```go
func (c Controller) ProtectedEndpoint(request *evo.Request) any {
    // Get authenticated user (type assertion required)
    user := request.User().(*auth.User)
    
    // Always check if user is authenticated before type assertion
    if user.Anonymous() {
        return response.Error(response.ErrUnauthorized)
    }
    
    // Safe to use user now
    userType := user.Type
}
```

### Response Patterns

**Success Responses**:
```go
func (c Controller) GetData(request *evo.Request) any {
    // Simple data response
    return data
    
    // Or use response library for standardized format
    return response.OK(data)
    return response.Created(newData)
    return response.OKWithMeta(data, &response.Meta{Page: 1, Total: 100})
}
```

**Error Responses**:
```go
func (c Controller) FailingEndpoint(request *evo.Request) any {
    // Use centralized error handling
    return response.Error(response.ErrUnauthorized)
    return response.Error(response.ErrInvalidInput)
    return response.Error(response.ErrTicketNotFound)
    
    // Custom errors
    return response.BadRequest(request, "Custom error message")
    return response.NotFound(request, "Resource not found")
    return response.InternalError(request, "Something went wrong")
}
```

### Pagination Integration

**Using pagination library**:
```go
import "github.com/getevo/pagination"

func (c Controller) ListTickets(request *evo.Request) any {
    var tickets []models.Ticket
    query := db.DB.Model(&models.Ticket{})
    
    // Apply filters, search, etc. to query
    
    // CRITICAL: Use request directly, not request.Request()
    p, err := pagination.New(query, request, &tickets, pagination.Options{
        MaxSize: 100,
    })
    if err != nil {
        return response.InternalError(request, "Failed to fetch tickets")
    }
    
    return response.OKWithMeta(tickets, &response.Meta{
        Page:       p.Page,
        Limit:      p.Size,
        Total:      p.Total,
        TotalPages: p.TotalPage,
    })
}
```

### Middleware Patterns

**Middleware function signature**:
```go
func MiddlewareName(request *evo.Request) any {
    // Middleware logic
    
    // Continue to next handler
    return request.Next()
    
    // Or stop with error
    return response.Error(response.ErrUnauthorized)
}
```

**Authentication middleware example**:
```go
func AdminAuthMiddleware(request *evo.Request) any {
    user := request.User().(*auth.User)
    
    if user.Anonymous() {
        return response.Error(response.ErrUnauthorized)
    }
    
    if user.Type != auth.UserTypeAdministrator {
        return response.Error(response.ErrForbidden)
    }
    
    return request.Next()
}
```

### Route Registration with Middleware

```go
func (a App) Router() error {
    var controller Controller
    
    // Routes without middleware
    evo.Get("/api/public", controller.PublicHandler)
    evo.Post("/api/auth/login", controller.LoginHandler)
    
    // Routes with middleware
    adminGroup := evo.Group("/api/admin")
    adminGroup.Use(AdminAuthMiddleware)
    adminGroup.Get("/tickets", controller.ListTickets)
    adminGroup.Post("/users", controller.CreateUser)
    
    return nil
}
```

### Controller Structure

```go
package appname

import (
    "github.com/getevo/evo/v2"
    "github.com/getevo/evo/v2/lib/db"
    "github.com/getevo/homa/apps/auth"
    "github.com/getevo/homa/apps/models"
    "github.com/getevo/homa/lib/response"
)

type Controller struct {
    // Add any shared dependencies here if needed
}

// All handler methods should be methods of Controller
func (c Controller) HandlerName(request *evo.Request) any {
    // Handler implementation
    return response.OK(responseData)
}

// Example: Create user handler
func (c Controller) CreateUser(request *evo.Request) any {
    var req CreateUserRequest
    if err := request.BodyParser(&req); err != nil {
        return response.Error(response.ErrInvalidInput)
    }
    
    // Validate user is admin
    user := request.User().(*auth.User)
    if user.Anonymous() {
        return response.Error(response.ErrUnauthorized)
    }
    
    // Process creation logic...
    newUser := models.User{
        Name:  req.Name,
        Email: req.Email,
        Type:  req.Type,
    }
    
    if err := db.DB.Create(&newUser).Error; err != nil {
        return response.InternalError(request, "Failed to create user")
    }
    
    return response.Created(newUser)
}
```

**Route Registration**:
```go
func (a App) Router() error {
    var controller Controller
    
    // Register routes using controller methods
    evo.Get("/endpoint", controller.HandlerMethod)
    evo.Post("/auth/login", controller.LoginHandler)
    evo.Get("/auth/profile", controller.GetProfile)
    
    return nil
}
```

**Benefits of Controller Pattern**:
- Organized code structure
- Easy to find related handlers
- Consistent naming conventions
- Better code reusability
- Easier testing and maintenance

**IMPORTANT RULE**: ALL API functions must be methods of the Controller struct. Do NOT create separate files like `public_apis.go`, `handlers.go`, or individual handler files. Everything should be organized within `controller.go` as Controller methods.

**Authentication Checks**: Before calling `request.User().(*User)` or similar type assertions, ALWAYS check if the user is logged in using `!user.Anonymous()` to prevent panics:

```go
func (c Controller) HandlerName(request *evo.Request) interface{} {
    // ALWAYS check if user is logged in first
    if request.User().Anonymous() {
        return response.Error(response.ErrUnauthorized)
    }
    
    // Safe to type assert now
    var user = request.User().Interface().(*auth.User)
    
    // Continue with handler logic...
}
```

**Centralized Error Handling**: Use the centralized error library (`/lib/errors/`) for consistent error responses:

```go
import "github.com/getevo/homa/lib/response"

func (c Controller) HandlerName(request *evo.Request) interface{} {
    // Use predefined errors
    return response.Error(response.ErrUnauthorized)
    return response.Error(response.ErrInvalidInput)
    return response.Error(response.ErrTicketNotFound)
    
    // Use error constructors for dynamic messages
    return response.Error(response.ErrUserDepartments())
    return response.Error(response.ErrCreateTagWithName(tagName))
    
    // Custom errors
    customErr := response.NewError(
        response.ErrorCodeValidationError,
        "Custom validation failed",
        http.StatusBadRequest,
    )
    return response.Error(customErr)
}
```

**Error Library Benefits**:
- Consistent error codes and messages across the application
- Standardized HTTP status codes
- Centralized error management
- Automatic `outcome.Response` formatting
- Support for error details and context

## File Organization

**Models by Domain**:
- `auth.go` - User, UserOAuthAccount  
- `clients.go` - Client, ClientExternalID
- `channels.go` - Channel
- `tickets.go` - Ticket, Message, Tag
- `relations.go` - Junction tables

## Critical Rules

1. **Always embed `restify.API` in all models - MUST be at the END of struct definition**
2. **Use UUID for Users and Clients, string for Channels**  
3. **Include `fk:table_name` in all foreign key tags**
4. **Add `foreignKey` and `references` to all relationships**
5. **Use snake_case for JSON tags**
6. **Create enum constants with database CHECK constraints**
7. **Use pointers for nullable fields**
8. **Register ALL models in models app**
9. **Group imports: stdlib, third-party, local**
10. **Follow Go naming conventions for constants**

## User Authentication System

**Framework Integration**: User model implements `evo.UserInterface` for request context integration.

**JWT Authentication**:
```go
// JWT Claims structure
type Claims struct {
    UserID      string   `json:"user_id"`
    Email       string   `json:"email"`
    Name        string   `json:"name"`
    Type        string   `json:"type"`
    Departments []string `json:"departments"`
    jwt.RegisteredClaims
}

// Set user interface in main.go
evo.SetUserInterface(&models.User{})
```

**Password Security**:
```go
// Use passlib for secure password hashing
func (u *User) SetPassword(password string) error {
    hash, err := passlib.Hash(password)
    if err != nil {
        return err
    }
    u.PasswordHash = &hash
    return nil
}

func (u *User) VerifyPassword(password string) bool {
    if u.PasswordHash == nil {
        return false
    }
    return passlib.Verify(password, *u.PasswordHash) == nil
}
```

**Authentication Endpoints**:
- `POST /auth/login` - User login with email/password
- `POST /auth/refresh` - JWT token refresh

**Login History**: Automatic tracking via `UserLoginHistory` model with IP, User-Agent, success status.

**User Types**: `agent` (regular user), `administrator` (full access)

## CLI Argument Parsing

Use `args.Get()` from `github.com/getevo/evo/v2/lib/args` for command-line argument parsing:

```go
import "github.com/getevo/evo/v2/lib/args"

// Reading arguments
email := args.Get("-email")
password := args.Get("-password")
name := args.Get("-name")
lastname := args.Get("-lastname")

// Check if argument exists (returns non-empty string if present)
if args.Get("-create-admin") != "" {
    // Argument was provided
}

// Check for boolean flags
if args.Exists("--migration-do") {
    // Flag was provided
}
```

**Usage Pattern Example**:
```bash
# Command line usage
./homa -create-admin -email admin@example.com -password secret123 -name Admin -lastname User

# In code
func CreateAdminUser() {
    email := args.Get("-email")
    password := args.Get("-password")
    name := args.Get("-name")
    lastName := args.Get("-lastname")
    
    if email == "" || password == "" || name == "" {
        fmt.Println("Usage: ./homa -create-admin -email admin@example.com -password secret123 -name Admin -lastname User")
        os.Exit(1)
    }
    // Process arguments...
}
```

## Commands

```bash
# Run with specific config file
go run main.go -c config.dev.yml

# Run with migration
go run main.go --migration-do

# Run with dev config and migration
go run main.go -c config.dev.yml --migration-do

# Run with Swagger UI enabled
go run main.go --swagger

# Run with both migration and Swagger UI
go run main.go --migration-do --swagger

# Create admin user via CLI
./homa -create-admin -email admin@example.com -password secret123 -name Admin -lastname User

# Create admin user with dev config
go run main.go -c config.dev.yml --create-admin -email admin@example.com -password secret123 -name Admin -lastname User

# Standard commands
go run main.go
go build -o homa main.go
go mod tidy
```

## API Documentation

When running with `--swagger` flag, the application provides:

- **Swagger UI**: Available at `http://localhost:8000/swagger`
- **OpenAPI Specification**: Available at `http://localhost:8000/swagger/openapi.json`

The Swagger UI provides:
- Interactive API documentation for all endpoints
- Request/response examples with proper schemas
- Authentication testing with JWT Bearer tokens
- Automatic documentation for all restify-enabled models
- Complete CRUD operations documentation for all entities

**Features**:
- Auto-generated OpenAPI 3.0 specification
- JWT Bearer authentication integration
- Complete model schemas with validation rules
- Interactive testing interface
- RESTful endpoint documentation for all models