package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/generic"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/getevo/restify"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hlandau/passlib"
	"gorm.io/gorm"
)

// User type constants
const (
	UserTypeAgent         = "agent"
	UserTypeAdministrator = "administrator"
)

// OAuth provider constants (for API responses only)
const (
	OAuthProviderGoogle    = "google"
	OAuthProviderMicrosoft = "microsoft"
)

// JWT configuration
var JWTSecret []byte

// InitializeJWTSecret should be called during app initialization (Register or WhenReady)
func InitializeJWTSecret() {
	// Try to get from config/env
	secret := settings.Get("JWT.SECRET").String()
	if secret == "" {
		// Fallback to environment variable
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		// Development fallback - should be changed in production
		log.Warning("JWT_SECRET not set, using development key. Change this in production!")
		secret = "your-secret-key-change-this-in-production"
	}
	JWTSecret = []byte(secret)
	log.Debug("JWT secret initialized successfully")
}

// JWT Claims structure
type Claims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Departments []string `json:"departments"`
	jwt.RegisteredClaims
}

type User struct {
	UserID       uuid.UUID `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	Name         string    `gorm:"column:name;size:255;not null" json:"name"`
	LastName     string    `gorm:"column:last_name;size:255;not null" json:"last_name"`
	DisplayName  string    `gorm:"column:display_name;size:255" json:"display_name"`
	Avatar       *string   `gorm:"column:avatar;size:500" json:"avatar"`
	Email        string    `gorm:"column:email;size:255;uniqueIndex;not null" json:"email"`
	PasswordHash *string   `gorm:"column:password_hash;size:255" json:"password_hash,omitempty"`
	APIKey       *string   `gorm:"column:api_key;size:255;uniqueIndex" json:"api_key,omitempty"`
	Type         string    `gorm:"column:type;size:50;not null;check:type IN ('agent','administrator')" json:"type"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	LoginHistory []UserLoginHistory `gorm:"foreignKey:UserID;references:UserID" json:"login_history,omitempty"`

	restify.API
}

type UserLoginHistory struct {
	ID        uint      `gorm:"column:id;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"column:user_id;type:char(36);not null;index;fk:users" json:"user_id"`
	IPAddress string    `gorm:"column:ip_address;size:45;not null" json:"ip_address"`
	UserAgent string    `gorm:"column:user_agent;size:500" json:"user_agent"`
	LoginAt   time.Time `gorm:"column:login_at;autoCreateTime" json:"login_at"`
	Success   bool      `gorm:"column:success;not null" json:"success"`
	Reason    string    `gorm:"column:reason;size:255" json:"reason"`

	// Relationships
	User User `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`

	restify.API
}

// BeforeCreate hook to generate UUID for User
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.UserID = uuid.New()
	return nil
}

// Evo UserInterface implementation
func (u *User) GetFirstName() string {
	return u.Name
}

func (u *User) GetLastName() string {
	return u.LastName
}

func (u *User) GetFullName() string {
	return u.DisplayName
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) UUID() string {
	return u.UserID.String()
}

func (u *User) ID() uint64 {
	// Convert UUID to uint64 for compatibility
	return uint64(u.UserID.ID())
}

func (u *User) Interface() interface{} {
	return u
}

func (u *User) Anonymous() bool {
	return u.UserID == uuid.Nil
}

func (u *User) HasPermission(permission string) bool {
	return u.Type == UserTypeAdministrator
}

func (u *User) Attributes() evo.Attributes {
	var m evo.Attributes
	generic.Parse(u).Cast(&m)
	return m
}

// FromRequest extracts user from JWT token in request
func (u *User) FromRequest(request *evo.Request) evo.UserInterface {
	authToken, ok := GetAuthToken(request)
	if !ok || authToken == "" {
		return u
	}

	// Handle API Key authentication
	if strings.HasPrefix(authToken, "APIKey") {
		apikey := strings.TrimSpace(authToken[6:])
		if apikey != "" {
			var user User
			if err := db.Where("api_key = ?", apikey).First(&user).Error; err != nil {
				log.Debug("API key not found:", err)
				return u
			}
			if !user.Anonymous() {
				return &user
			}
		}
		return u
	}

	// Handle JWT Bearer token authentication
	if !strings.HasPrefix(authToken, "Bearer ") {
		return u
	}

	tokenString := strings.TrimPrefix(authToken, "Bearer ")

	// Clean up the token string - remove any trailing data
	tokenString = strings.TrimSpace(tokenString)
	if idx := strings.Index(tokenString, ","); idx != -1 {
		tokenString = tokenString[:idx]
	}
	if idx := strings.Index(tokenString, "\""); idx != -1 {
		tokenString = tokenString[:idx]
	}

	// Debug: Check if secret is properly initialized
	if len(JWTSecret) == 0 {
		log.Error("JWT secret is not initialized!")
		return u
	}

	jwtToken, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	})

	if err != nil {
		log.Debug("JWT parsing error:", err)
		return u
	}

	if !jwtToken.Valid {
		log.Debug("JWT token is not valid")
		return u
	}

	claims, ok := jwtToken.Claims.(*Claims)
	if !ok {
		log.Debug("JWT claims parsing failed")
		return u
	}

	// Find user in database
	var user User
	if err := db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		log.Debug("User not found for claims:", claims.UserID)
		return u
	}

	return &user
}

// Password and JWT utilities
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
	_, err := passlib.Verify(password, *u.PasswordHash)
	return err == nil
}

// GenerateAPIKey creates a new API key for the user
func (u *User) GenerateAPIKey() (string, error) {
	// Generate a random 32-byte API key
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to hex string and add prefix for easy identification
	apiKey := "homa_" + hex.EncodeToString(bytes)
	u.APIKey = &apiKey

	return apiKey, nil
}

// ClearAPIKey removes the API key from the user
func (u *User) ClearAPIKey() {
	u.APIKey = nil
}

func (u *User) GenerateJWT() (string, error) {
	// Load departments for this user - will need to import from models package later
	var departments []interface{} // Placeholder for now

	departmentNames := make([]string, len(departments))
	// for i, dept := range departments {
	//     departmentNames[i] = dept.Name
	// }

	claims := Claims{
		UserID:      u.UserID.String(),
		Email:       u.Email,
		Name:        u.GetFullName(),
		Type:        u.Type,
		Departments: departmentNames,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

func (u *User) GenerateRefreshToken() (string, error) {
	claims := Claims{
		UserID: u.UserID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// Login creates a login history record
func (u *User) RecordLogin(request *evo.Request, success bool, reason string) {
	ip := request.IP()
	if ip == "" {
		ip = "unknown"
	}

	userAgent := request.Header("User-Agent")
	if userAgent == "" {
		userAgent = "unknown"
	}

	history := UserLoginHistory{
		UserID:    u.UserID,
		IPAddress: ip,
		UserAgent: userAgent,
		Success:   success,
		Reason:    reason,
	}

	db.Create(&history)
}

func (UserLoginHistory) TableName() string {
	return "user_login_history"
}

// GetAuthToken retrieves the authentication token from the request.
// It first tries to get the token from the "Authorization" header of the request.
// If the token is not found in the header, it then tries to get it from the "Authorization" cookie of the request.
// If the token length is less than 7 characters, it returns an empty token and false as the second return value.
// Otherwise, it removes the "Bearer " prefix from the token and returns it along with true as the second return value.
func GetAuthToken(request *evo.Request) (string, bool) {
	var token = request.Header("X-Authorization")
	if token == "" {
		token = request.Header("Authorization")
	}
	if token == "" {
		token = request.Cookie("Authorization")
	}
	return token, true
}
