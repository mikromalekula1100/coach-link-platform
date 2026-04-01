package model

import (
	"encoding/json"
	"time"
)

// ──────────────────────────────────────────────
// Database models (sqlx)
// ──────────────────────────────────────────────

type Notification struct {
	ID        string          `db:"id"`
	UserID    string          `db:"user_id"`
	Type      string          `db:"type"`
	Title     string          `db:"title"`
	Body      *string         `db:"body"`
	Data      json.RawMessage `db:"data"`
	IsRead    bool            `db:"is_read"`
	CreatedAt time.Time       `db:"created_at"`
}

type DeviceToken struct {
	ID         string    `db:"id"`
	UserID     string    `db:"user_id"`
	FCMToken   string    `db:"fcm_token"`
	DeviceInfo *string   `db:"device_info"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// ──────────────────────────────────────────────
// DTOs – JSON response bodies
// ──────────────────────────────────────────────

type NotificationResponse struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      *string                `json:"body,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	IsRead    bool                   `json:"is_read"`
	CreatedAt time.Time              `json:"created_at"`
}

type NotificationsListResponse struct {
	Items       []NotificationResponse `json:"items"`
	Pagination  Pagination             `json:"pagination"`
	UnreadCount int                    `json:"unread_count"`
}

// ──────────────────────────────────────────────
// DTOs – JSON request bodies
// ──────────────────────────────────────────────

type RegisterDeviceTokenRequest struct {
	FCMToken   string  `json:"fcm_token" validate:"required"`
	DeviceInfo *string `json:"device_info,omitempty"`
}

// ──────────────────────────────────────────────
// Pagination
// ──────────────────────────────────────────────

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// ──────────────────────────────────────────────
// Error response
// ──────────────────────────────────────────────

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ──────────────────────────────────────────────
// Conversion helpers
// ──────────────────────────────────────────────

func ToNotificationResponse(n *Notification) NotificationResponse {
	resp := NotificationResponse{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}

	if len(n.Data) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(n.Data, &data); err == nil {
			resp.Data = data
		}
	}

	return resp
}
