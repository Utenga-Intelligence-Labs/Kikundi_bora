package services

import (
	"kikundibora/database"
	"kikundibora/models"
)

// NotifyUser sends a notification to a specific user.
func NotifyUser(userID string, notifType models.NotificationType, title, message string) {
	notif := models.Notification{
		UserID:  userID,
		Type:    notifType,
		Title:   title,
		Message: message,
	}
	database.DB.Create(&notif)
}

// NotifyUsers sends a notification to multiple users.
func NotifyUsers(userIDs []string, notifType models.NotificationType, title, message string) {
	for _, uid := range userIDs {
		NotifyUser(uid, notifType, title, message)
	}
}

// NotifyRole sends a notification to all users with a specific role, optionally excluding one user.
func NotifyRole(role models.Role, notifType models.NotificationType, title, message string, excludeUserID string) {
	var users []models.User
	query := database.DB.Where("role = ? AND status = ? AND deleted_at IS NULL", role, models.UserStatusActive)
	if excludeUserID != "" {
		query = query.Where("id != ?", excludeUserID)
	}
	query.Find(&users)
	for _, u := range users {
		NotifyUser(u.ID, notifType, title, message)
	}
}
