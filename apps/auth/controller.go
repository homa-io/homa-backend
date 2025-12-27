package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/url"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/iesreza/homa-backend/lib/response"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hlandau/passlib"
	"golang.org/x/oauth2"
)

type Controller struct {
}

// OAuthProvider represents an OAuth provider configuration (without secrets)
type OAuthProvider struct {
	Provider    string `json:"provider"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	RedirectURI string `json:"redirect_uri,omitempty"`
	Scopes      string `json:"scopes,omitempty"`
}

// OAuthProvidersResponse defines the structure for the OAuth providers response
type OAuthProvidersResponse []OAuthProvider

// OAuth user info structures
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

type MicrosoftUserInfo struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	Photo             string `json:"photo,omitempty"`
}

// OAuth login response
type OAuthLoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}

// Department represents a department (avoiding import cycle)
type Department struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetProfileResponse defines the structure for the get profile response
type GetProfileResponse struct {
	User
	Departments []Department `json:"departments"`
}

// EditProfileRequest defines the structure for the edit profile request
type EditProfileRequest struct {
	Name        string  `json:"name"`
	LastName    string  `json:"last_name"`
	DisplayName string  `json:"display_name"`
	Avatar      *string `json:"avatar"`
	Password    *string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         *User  `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (c Controller) LoginHandler(request *evo.Request) any {
	var loginReq LoginRequest
	if err := request.BodyParser(&loginReq); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Find user by email
	var user User
	if err := db.Where("email = ?", loginReq.Email).First(&user).Error; err != nil {
		user.RecordLogin(request, false, "user_not_found")
		invalidCredentialsErr := response.NewError(response.ErrorCodeUnauthorized, "Invalid email or password", 401)
		return response.Error(invalidCredentialsErr)
	}

	// Verify password
	if !user.VerifyPassword(loginReq.Password) {
		user.RecordLogin(request, false, "invalid_password")
		invalidCredentialsErr := response.NewError(response.ErrorCodeUnauthorized, "Invalid email or password", 401)
		return response.Error(invalidCredentialsErr)
	}

	// Check if user is blocked
	if user.Status == UserStatusBlocked {
		user.RecordLogin(request, false, "account_blocked")
		blockedErr := response.NewError(response.ErrorCodeForbidden, "Your account has been blocked. Please contact an administrator.", 403)
		return response.Error(blockedErr)
	}

	// Generate tokens
	accessToken, err := user.GenerateJWT()
	if err != nil {
		user.RecordLogin(request, false, "token_generation_failed")
		return response.Error(response.ErrInternalError)
	}

	refreshToken, err := user.GenerateRefreshToken()
	if err != nil {
		user.RecordLogin(request, false, "refresh_token_generation_failed")
		return response.Error(response.ErrInternalError)
	}

	// Record successful login
	user.RecordLogin(request, true, "login_success")

	// Hide sensitive data
	user.PasswordHash = nil

	loginData := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    86400, // 24 hours
		User:         &user,
	}

	return response.OK(loginData)
}

func (c Controller) RefreshHandler(request *evo.Request) any {
	var refreshReq RefreshRequest
	if err := request.BodyParser(&refreshReq); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Parse refresh token
	token, err := jwt.ParseWithClaims(refreshReq.RefreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})

	if err != nil || !token.Valid {
		return response.Error(response.ErrInvalidToken)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return response.Error(response.ErrInvalidToken)
	}

	// Find user
	var user User
	if err := db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return response.Error(response.ErrUserNotFound)
	}

	// Generate new access token
	accessToken, err := user.GenerateJWT()
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Generate new refresh token
	newRefreshToken, err := user.GenerateRefreshToken()
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Hide sensitive data
	user.PasswordHash = nil

	refreshData := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    86400, // 24 hours
		User:         &user,
	}

	return response.OK(refreshData)
}

// GetOAuthProviders returns available OAuth providers with dynamic base URL
// @Summary Get OAuth providers
// @Description Get a list of available OAuth providers with their public configuration
// @Tags OAuth
// @Accept json
// @Produce json
// @Success 200 {array} OAuthProvider
// @Router /auth/oauth/providers [get]
func (c Controller) GetOAuthProviders(req *evo.Request) interface{} {
	var url = req.URL()
	var base = url.Scheme + "://" + url.Host + "/api/auth/oauth/"

	var providers []OAuthProvider

	// Check if Google OAuth is enabled
	if settings.Get("OAUTH.GOOGLE.ENABLED").Bool() {
		providers = append(providers, OAuthProvider{
			Provider:    OAuthProviderGoogle,
			Name:        "Google",
			Enabled:     true,
			RedirectURI: base + "google/callback",
			Scopes:      "userinfo.email,userinfo.profile",
		})
	}

	// Check if Microsoft OAuth is enabled
	if settings.Get("OAUTH.MICROSOFT.ENABLED").Bool() {
		providers = append(providers, OAuthProvider{
			Provider:    OAuthProviderMicrosoft,
			Name:        "Microsoft",
			Enabled:     true,
			RedirectURI: base + "microsoft/callback",
			Scopes:      "user.read",
		})
	}

	return response.List(providers, len(providers))
}

// generateState generates a random state for OAuth
func (c Controller) generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateStateWithRedirect generates state parameter with embedded redirect URL
func (c Controller) generateStateWithRedirect(redirectURL string) string {
	stateData := map[string]string{
		"random":       c.generateState(),
		"redirect_url": redirectURL,
	}

	jsonData, err := json.Marshal(stateData)
	if err != nil {
		log.Error("Failed to marshal state data:", err)
		return c.generateState() // Fallback to simple state
	}

	return base64.URLEncoding.EncodeToString(jsonData)
}

// extractRedirectFromState extracts redirect URL from state parameter
func (c Controller) extractRedirectFromState(state string) string {
	decodedData, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		log.Error("Failed to decode state:", err)
		return "/static/login.html" // Fallback
	}

	var stateData map[string]string
	if err := json.Unmarshal(decodedData, &stateData); err != nil {
		log.Error("Failed to unmarshal state data:", err)
		return "/static/login.html" // Fallback
	}

	if redirectURL, ok := stateData["redirect_url"]; ok {
		return redirectURL
	}

	return "/static/login.html" // Fallback
}

// GoogleOAuthLogin initiates Google OAuth login
// @Summary Start Google OAuth login
// @Description Redirects to Google OAuth consent page
// @Tags OAuth
// @Accept json
// @Produce json
// @Param redirect_url query string true "URL to redirect after OAuth completion"
// @Router /auth/oauth/google [get]
func (c Controller) GoogleOAuthLogin(req *evo.Request) interface{} {
	// Check if redirect URL is provided
	redirectURL := req.Query("redirect_url").String()
	if redirectURL == "" {
		missingRedirectErr := response.NewError(response.ErrorCodeMissingRequired, "redirect_url parameter is required", 400)
		return response.Error(missingRedirectErr)
	}

	if GoogleOAuthConfig.ClientID == "" {
		oauthNotConfiguredErr := response.NewError(response.ErrorCodeInternalError, "Google OAuth is not configured", 503)
		return response.Error(oauthNotConfiguredErr)
	}

	// Update redirect URL with dynamic base
	var url = req.URL()
	var base = url.Scheme + "://" + url.Host + "/api/auth/oauth/"
	GoogleOAuthConfig.RedirectURL = base + "google/callback"

	// Encode redirect URL in state parameter
	state := c.generateStateWithRedirect(redirectURL)
	authURL := GoogleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	// Check if this is an API request (JSON content type or API endpoint)
	if req.Query("format").String() == "json" || req.Get("Content-Type").String() == "application/json" {
		return response.OK(map[string]string{
			"auth_url": authURL,
			"state":    state,
		})
	}
	return req.Redirect(authURL)
}

// GoogleOAuthCallback handles Google OAuth callback
// @Summary Handle Google OAuth callback
// @Description Handles the callback from Google OAuth and logs in the user
// @Tags OAuth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 200 {object} OAuthLoginResponse
// @Router /auth/oauth/google/callback [get]
func (c Controller) GoogleOAuthCallback(req *evo.Request) interface{} {
	code := req.Query("code").String()
	state := req.Query("state").String()

	// Extract redirect URL from state parameter
	redirectURL := c.extractRedirectFromState(state)

	if code == "" {
		return req.Redirect(redirectURL + "?oauth=error&message=Authorization%20code%20is%20required")
	}

	// Exchange code for token
	token, err := GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Error("Failed to exchange token:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20exchange%20authorization%20code")
	}

	// Get user info from Google
	client := GoogleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Error("Failed to get user info:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20get%20user%20information")
	}
	defer resp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Error("Failed to decode user info:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20decode%20user%20information")
	}

	// Find user by email
	var user User
	if err := db.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
		return req.Redirect(redirectURL + "?oauth=error&message=User%20not%20found.%20Only%20existing%20users%20can%20log%20in%20via%20OAuth.")
	}

	// OAuth login successful - user found by email match

	// Update avatar if not already set or if OAuth provides a new one
	if userInfo.Picture != "" && (user.Avatar == nil || *user.Avatar == "") {
		user.Avatar = &userInfo.Picture
		db.Save(&user)
	}

	// Generate JWT tokens
	accessToken, err := user.GenerateJWT()
	if err != nil {
		log.Error("Failed to generate JWT:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20generate%20access%20token")
	}

	refreshToken, err := user.GenerateRefreshToken()
	if err != nil {
		log.Error("Failed to generate refresh token:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20generate%20refresh%20token")
	}

	// Record login
	user.RecordLogin(req, true, "oauth_google")

	// Prepare safe user data (no sensitive info)
	safeUser := User{
		UserID:      user.UserID,
		Name:        user.Name,
		LastName:    user.LastName,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Email:       user.Email,
		Type:        user.Type,
	}

	response := OAuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    24 * 60 * 60, // 24 hours
		User:         safeUser,
	}

	// Redirect to specified URL with response data
	return req.Redirect(redirectURL + "?oauth=success&data=" + encodeResponseData(response))
}

// MicrosoftOAuthLogin initiates Microsoft OAuth login
// @Summary Start Microsoft OAuth login
// @Description Redirects to Microsoft OAuth consent page
// @Tags OAuth
// @Accept json
// @Produce json
// @Param redirect_url query string true "URL to redirect after OAuth completion"
// @Router /auth/oauth/microsoft [get]
func (c Controller) MicrosoftOAuthLogin(req *evo.Request) interface{} {
	// Check if redirect URL is provided
	redirectURL := req.Query("redirect_url").String()
	if redirectURL == "" {
		missingRedirectErr := response.NewError(response.ErrorCodeMissingRequired, "redirect_url parameter is required", 400)
		return response.Error(missingRedirectErr)
	}

	if MicrosoftOAuthConfig.ClientID == "" {
		oauthNotConfiguredErr := response.NewError(response.ErrorCodeInternalError, "Microsoft OAuth is not configured", 503)
		return response.Error(oauthNotConfiguredErr)
	}

	// Update redirect URL with dynamic base
	var url = req.URL()
	var base = url.Scheme + "://" + url.Host + "/api/auth/oauth/"
	MicrosoftOAuthConfig.RedirectURL = base + "microsoft/callback"

	// Encode redirect URL in state parameter
	state := c.generateStateWithRedirect(redirectURL)
	authURL := MicrosoftOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	// Check if this is an API request (JSON content type or API endpoint)
	if req.Query("format").String() == "json" || req.Get("Content-Type").String() == "application/json" {
		return response.OK(map[string]string{
			"auth_url": authURL,
			"state":    state,
		})
	}
	return req.Redirect(authURL)
}

// MicrosoftOAuthCallback handles Microsoft OAuth callback
// @Summary Handle Microsoft OAuth callback
// @Description Handles the callback from Microsoft OAuth and logs in the user
// @Tags OAuth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 200 {object} OAuthLoginResponse
// @Router /auth/oauth/microsoft/callback [get]
func (c Controller) MicrosoftOAuthCallback(req *evo.Request) interface{} {
	code := req.Query("code").String()
	state := req.Query("state").String()

	// Extract redirect URL from state parameter
	redirectURL := c.extractRedirectFromState(state)

	if code == "" {
		return req.Redirect(redirectURL + "?oauth=error&message=Authorization%20code%20is%20required")
	}

	// Exchange code for token
	token, err := MicrosoftOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Error("Failed to exchange token:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20exchange%20authorization%20code")
	}

	// Get user info from Microsoft Graph
	client := MicrosoftOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		log.Error("Failed to get user info:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20get%20user%20information")
	}
	defer resp.Body.Close()

	var userInfo MicrosoftUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Error("Failed to decode user info:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20decode%20user%20information")
	}

	// Use mail or userPrincipalName as email
	email := userInfo.Mail
	if email == "" {
		email = userInfo.UserPrincipalName
	}

	// Find user by email
	var user User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return req.Redirect(redirectURL + "?oauth=error&message=User%20not%20found.%20Only%20existing%20users%20can%20log%20in%20via%20OAuth.")
	}

	// OAuth login successful - user found by email match

	// Try to get Microsoft profile photo
	if user.Avatar == nil || *user.Avatar == "" {
		photoResp, err := client.Get("https://graph.microsoft.com/v1.0/me/photo/$value")
		if err == nil && photoResp.StatusCode == 200 {
			// Microsoft photo is binary data, we'd need to upload it somewhere
			// For now, we'll skip this and use a placeholder or let user upload manually
			defer photoResp.Body.Close()
		}
	}

	// Generate JWT tokens
	accessToken, err := user.GenerateJWT()
	if err != nil {
		log.Error("Failed to generate JWT:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20generate%20access%20token")
	}

	refreshToken, err := user.GenerateRefreshToken()
	if err != nil {
		log.Error("Failed to generate refresh token:", err)
		return req.Redirect(redirectURL + "?oauth=error&message=Failed%20to%20generate%20refresh%20token")
	}

	// Record login
	user.RecordLogin(req, true, "oauth_microsoft")

	// Prepare safe user data (no sensitive info)
	safeUser := User{
		UserID:      user.UserID,
		Name:        user.Name,
		LastName:    user.LastName,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Email:       user.Email,
		Type:        user.Type,
	}

	res := OAuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    24 * 60 * 60, // 24 hours
		User:         safeUser,
	}

	// Redirect to specified URL with response data
	return req.Redirect(redirectURL + "?oauth=success&data=" + encodeResponseData(res))
}

// GetProfile returns the current user's profile
// @Summary Get user profile
// @Description Get the profile of the currently authenticated user.
// @Tags Profile
// @Accept json
// @Produce json
// @Success 200 {object} GetProfileResponse
// @Router /auth/profile [get]
// @Security Bearer
func (c Controller) GetProfile(req *evo.Request) interface{} {
	var user = req.User().(*User)
	var departments []Department
	// Query departments directly to avoid import cycle
	db.Raw("SELECT d.id, d.name, d.description FROM departments d JOIN user_departments ud ON d.id = ud.department_id WHERE ud.user_id = ?", user.UserID).Scan(&departments)

	profileData := GetProfileResponse{
		User:        *user,
		Departments: departments,
	}

	return response.OK(profileData)
}

// EditProfile updates the current user's profile
// @Summary Update user profile
// @Description Update the profile of the currently authenticated user.
// @Tags Profile
// @Accept json
// @Produce json
// @Param body body EditProfileRequest true "User profile data"
// @Success 200 {object} User
// @Router /auth/profile [put]
// @Security Bearer
func (c Controller) EditProfile(req *evo.Request) interface{} {
	var user = req.User().(*User)
	var params EditProfileRequest
	if err := req.BodyParser(&params); err != nil {
		return err
	}

	user.Name = params.Name
	user.LastName = params.LastName
	user.DisplayName = params.DisplayName
	user.Avatar = params.Avatar

	if params.Password != nil && *params.Password != "" {
		hash, err := passlib.Hash(*params.Password)
		if err != nil {
			log.Error(err)
			return err
		}
		user.PasswordHash = &hash
	}

	if err := db.Save(&user).Error; err != nil {
		log.Error(err)
		return response.Error(response.ErrDatabaseError)
	}

	return response.OKWithMessage(user, "Profile updated successfully")
}

// GenerateAPIKey generates a new API key for the authenticated user
// @Summary Generate API key
// @Description Generate a new API key for the authenticated user. This will replace any existing API key.
// @Tags Profile
// @Accept json
// @Produce json
// @Success 200 {object} object{api_key=string}
// @Router /auth/api-key [post]
// @Security Bearer
func (c Controller) GenerateAPIKey(req *evo.Request) interface{} {
	user := req.User().(*User)

	apiKey, err := user.GenerateAPIKey()
	if err != nil {
		log.Error("Failed to generate API key:", err)
		return response.Error(response.ErrInternalError)
	}

	if err := db.Save(user).Error; err != nil {
		log.Error("Failed to save API key:", err)
		return response.Error(response.ErrDatabaseError)
	}

	return response.OK(map[string]string{
		"api_key": apiKey,
	})
}

// RevokeAPIKey removes the API key from the authenticated user
// @Summary Revoke API key
// @Description Revoke the current API key for the authenticated user
// @Tags Profile
// @Accept json
// @Produce json
// @Success 200 {object} object{message=string}
// @Router /auth/api-key [delete]
// @Security Bearer
func (c Controller) RevokeAPIKey(req *evo.Request) interface{} {
	user := req.User().(*User)

	user.ClearAPIKey()

	if err := db.Save(user).Error; err != nil {
		log.Error("Failed to revoke API key:", err)
		return response.Error(response.ErrDatabaseError)
	}

	return response.Message("API key revoked successfully")
}

// encodeResponseData encodes response data for URL parameter
func encodeResponseData(response OAuthLoginResponse) string {
	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Error("Failed to marshal OAuth response:", err)
		return ""
	}
	return url.QueryEscape(base64.URLEncoding.EncodeToString(jsonData))
}
