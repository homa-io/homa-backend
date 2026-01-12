package models

import (
	"github.com/getevo/evo/v2/lib/db"
	"github.com/google/uuid"
)

// GetUserDepartmentIDs returns all department IDs for a user
// This is used for access control across multiple controllers
func GetUserDepartmentIDs(userID uuid.UUID) ([]uint, error) {
	var departmentIDs []uint
	err := db.Model(&UserDepartment{}).
		Where("user_id = ?", userID).
		Pluck("department_id", &departmentIDs).Error
	if err != nil {
		return nil, err
	}
	return departmentIDs, nil
}

// HasDepartmentAccess checks if a user has access to a specific department
func HasDepartmentAccess(userID uuid.UUID, departmentID uint) bool {
	departmentIDs, err := GetUserDepartmentIDs(userID)
	if err != nil {
		return false
	}
	for _, id := range departmentIDs {
		if id == departmentID {
			return true
		}
	}
	return false
}

// HasConversationAccess checks if a user has access to a specific conversation
// A user has access if:
// - The conversation's department is in their departments, OR
// - The conversation is assigned to them
func HasConversationAccess(userID uuid.UUID, conversationID uint) bool {
	departmentIDs, err := GetUserDepartmentIDs(userID)
	if err != nil {
		return false
	}

	var count int64
	err = db.Model(&Conversation{}).
		Where("id = ? AND (department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?))",
			conversationID, departmentIDs, userID).
		Count(&count).Error

	return err == nil && count > 0
}

// Note: For applying department-based filtering, use:
// departmentIDs, _ := GetUserDepartmentIDs(userID)
// query.Where("department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)", departmentIDs, userID)

// BuildConversationAccessCondition returns the SQL condition for conversation access
// Useful for building complex queries with access control
func BuildConversationAccessCondition(userID uuid.UUID) (string, []interface{}) {
	departmentIDs, err := GetUserDepartmentIDs(userID)
	if err != nil {
		return "1=0", nil // Deny access if can't get departments
	}

	condition := "department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)"
	args := []interface{}{departmentIDs, userID}
	return condition, args
}
