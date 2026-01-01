package auth

import (
	"strconv"
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/lib/response"
)

// List Users returns a paginated list of users
func (c Controller) ListUsers(request *evo.Request) interface{} {
	// Check if user is administrator
	user := request.User().Interface().(*User)
	if user.Type != UserTypeAdministrator {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "Only administrators can access user management", 403))
	}

	// Parse query parameters
	pageStr := request.Query("page").String()
	if pageStr == "" {
		pageStr = "1"
	}
	page, _ := strconv.Atoi(pageStr)

	pageSizeStr := request.Query("page_size").String()
	if pageSizeStr == "" {
		pageSizeStr = "10"
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)

	search := request.Query("search").String()
	userType := request.Query("type").String()
	status := request.Query("status").String()

	// Set defaults and limits
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Build query
	query := db.Model(&User{})

	// Apply search filter
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(display_name) LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	// Apply type filter
	if userType != "" && (userType == UserTypeAgent || userType == UserTypeAdministrator) {
		query = query.Where("type = ?", userType)
	}

	// Apply status filter
	if status != "" && (status == UserStatusActive || status == UserStatusBlocked) {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Calculate pagination
	offset := (page - 1) * pageSize
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// Get users
	var users []User
	if err := query.
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Remove sensitive data
	for i := range users {
		users[i].PasswordHash = nil
		users[i].APIKey = nil
	}

	return response.OK(map[string]interface{}{
		"users":       users,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// CreateUser creates a new user
func (c Controller) CreateUser(request *evo.Request) interface{} {
	// Check if user is administrator
	user := request.User().Interface().(*User)
	if user.Type != UserTypeAdministrator {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "Only administrators can create users", 403))
	}

	var req struct {
		Name        string  `json:"name"`
		LastName    string  `json:"last_name"`
		DisplayName string  `json:"display_name"`
		Email       string  `json:"email"`
		Password    string  `json:"password"`
		Type        string  `json:"type"`
		Avatar      *string `json:"avatar"`
		SecurityKey *string `json:"security_key"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Validate required fields
	if req.Name == "" || req.LastName == "" || req.Email == "" || req.Password == "" || req.Type == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Missing required fields", 400))
	}

	// Validate user type
	if req.Type != UserTypeAgent && req.Type != UserTypeAdministrator && req.Type != UserTypeBot {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user type", 400))
	}

	// Check if email already exists
	var existingUser User
	if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return response.Error(response.NewError(response.ErrorCodeConflict, "A user with this email already exists", 409))
	}

	// Set display name if not provided
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name + " " + req.LastName
	}

	// Create new user
	newUser := User{
		Name:        req.Name,
		LastName:    req.LastName,
		DisplayName: displayName,
		Email:       req.Email,
		Type:        req.Type,
		Status:      UserStatusActive,
		Avatar:      req.Avatar,
	}

	// Set security_key for bot users
	if req.Type == UserTypeBot && req.SecurityKey != nil {
		newUser.SecurityKey = req.SecurityKey
	}

	// Set password
	if err := newUser.SetPassword(req.Password); err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Save to database
	if err := db.Create(&newUser).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Remove sensitive data before returning
	newUser.PasswordHash = nil
	newUser.APIKey = nil

	return response.OKWithMessage(map[string]interface{}{
		"user": newUser,
	}, "User created successfully")
}

// GetUser retrieves a single user by ID
func (c Controller) GetUser(request *evo.Request) interface{} {
	// Check if user is administrator
	user := request.User().Interface().(*User)
	if user.Type != UserTypeAdministrator {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "Only administrators can access user management", 403))
	}

	userID := request.Param("id").String()
	if userID == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "User ID is required", 400))
	}

	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user ID format", 400))
	}

	// Find user
	var targetUser User
	if err := db.Where("id = ?", id).First(&targetUser).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "User not found", 404))
	}

	// Remove sensitive data
	targetUser.PasswordHash = nil
	targetUser.APIKey = nil

	return response.OK(targetUser)
}

// UpdateUser updates an existing user
func (c Controller) UpdateUser(request *evo.Request) interface{} {
	// Check if user is administrator
	user := request.User().Interface().(*User)
	if user.Type != UserTypeAdministrator {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "Only administrators can update users", 403))
	}

	userID := request.Param("id").String()
	if userID == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "User ID is required", 400))
	}

	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user ID format", 400))
	}

	var req struct {
		Name        *string `json:"name"`
		LastName    *string `json:"last_name"`
		DisplayName *string `json:"display_name"`
		Email       *string `json:"email"`
		Password    *string `json:"password"`
		Type        *string `json:"type"`
		Avatar      *string `json:"avatar"`
		SecurityKey *string `json:"security_key"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Find user
	var targetUser User
	if err := db.Where("id = ?", id).First(&targetUser).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "User not found", 404))
	}

	// Update fields if provided
	if req.Name != nil {
		targetUser.Name = *req.Name
	}
	if req.LastName != nil {
		targetUser.LastName = *req.LastName
	}
	if req.DisplayName != nil {
		targetUser.DisplayName = *req.DisplayName
	}
	if req.Email != nil {
		// Check if email is already taken by another user
		var existingUser User
		if err := db.Where("email = ? AND id != ?", *req.Email, id).First(&existingUser).Error; err == nil {
			return response.Error(response.NewError(response.ErrorCodeConflict, "Email is already taken by another user", 409))
		}
		targetUser.Email = *req.Email
	}
	if req.Type != nil {
		if *req.Type != UserTypeAgent && *req.Type != UserTypeAdministrator && *req.Type != UserTypeBot {
			return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user type", 400))
		}
		targetUser.Type = *req.Type
	}
	if req.Avatar != nil {
		targetUser.Avatar = req.Avatar
	}
	if req.Password != nil && *req.Password != "" {
		if err := targetUser.SetPassword(*req.Password); err != nil {
			return response.Error(response.ErrInternalError)
		}
	}
	// Update security_key for bot users
	if req.SecurityKey != nil && targetUser.Type == UserTypeBot {
		targetUser.SecurityKey = req.SecurityKey
	}

	// Save updates
	if err := db.Save(&targetUser).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Remove sensitive data before returning
	targetUser.PasswordHash = nil
	targetUser.APIKey = nil

	return response.OKWithMessage(map[string]interface{}{
		"user": targetUser,
	}, "User updated successfully")
}

// DeleteUser deletes a user
func (c Controller) DeleteUser(request *evo.Request) interface{} {
	// Check if user is administrator
	user := request.User().Interface().(*User)
	if user.Type != UserTypeAdministrator {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "Only administrators can delete users", 403))
	}

	userID := request.Param("id").String()
	if userID == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "User ID is required", 400))
	}

	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user ID format", 400))
	}

	// Prevent self-deletion
	if id == user.UserID {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "You cannot delete your own account", 400))
	}

	// Find user
	var targetUser User
	if err := db.Where("id = ?", id).First(&targetUser).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "User not found", 404))
	}

	// Delete user
	if err := db.Delete(&targetUser).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	return response.OKWithMessage(nil, "User deleted successfully")
}

// BlockUser blocks a user from accessing the system
func (c Controller) BlockUser(request *evo.Request) interface{} {
	// Check if user is administrator
	user := request.User().Interface().(*User)
	if user.Type != UserTypeAdministrator {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "Only administrators can block users", 403))
	}

	userID := request.Param("id").String()
	if userID == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "User ID is required", 400))
	}

	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user ID format", 400))
	}

	// Prevent self-blocking
	if id == user.UserID {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "You cannot block your own account", 400))
	}

	// Find user
	var targetUser User
	if err := db.Where("id = ?", id).First(&targetUser).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "User not found", 404))
	}

	// Update status
	targetUser.Status = UserStatusBlocked
	if err := db.Save(&targetUser).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	return response.OKWithMessage(map[string]interface{}{
		"id":     targetUser.UserID,
		"email":  targetUser.Email,
		"status": targetUser.Status,
	}, "User blocked successfully")
}

// UnblockUser unblocks a user
func (c Controller) UnblockUser(request *evo.Request) interface{} {
	// Check if user is administrator
	user := request.User().Interface().(*User)
	if user.Type != UserTypeAdministrator {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "Only administrators can unblock users", 403))
	}

	userID := request.Param("id").String()
	if userID == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "User ID is required", 400))
	}

	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user ID format", 400))
	}

	// Find user
	var targetUser User
	if err := db.Where("id = ?", id).First(&targetUser).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "User not found", 404))
	}

	// Update status
	targetUser.Status = UserStatusActive
	if err := db.Save(&targetUser).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	return response.OKWithMessage(map[string]interface{}{
		"id":     targetUser.UserID,
		"email":  targetUser.Email,
		"status": targetUser.Status,
	}, "User unblocked successfully")
}
