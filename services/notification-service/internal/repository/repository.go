package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/coach-link/platform/services/notification-service/internal/model"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ──────────────────────────────────────────────
// Notifications
// ──────────────────────────────────────────────

func (r *Repository) CreateNotification(ctx context.Context, n *model.Notification) error {
	const q = `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	var dataBytes []byte
	if n.Data != nil {
		dataBytes = n.Data
	}

	return r.db.QueryRowContext(ctx, q,
		n.UserID,
		n.Type,
		n.Title,
		n.Body,
		dataBytes,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *Repository) GetNotifications(ctx context.Context, userID string, isRead *bool, page, pageSize int) ([]model.Notification, int, error) {
	offset := (page - 1) * pageSize

	var args []interface{}
	args = append(args, userID)
	argIdx := 2

	where := "WHERE user_id = $1"
	if isRead != nil {
		where += fmt.Sprintf(" AND is_read = $%d", argIdx)
		args = append(args, *isRead)
		argIdx++
	}

	// Count total
	var total int
	countQ := "SELECT COUNT(*) FROM notifications " + where
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}

	// Fetch page
	dataQ := fmt.Sprintf(
		`SELECT id, user_id, type, title, body, data, is_read, created_at
		 FROM notifications %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, pageSize, offset)

	var notifications []model.Notification
	if err := r.db.SelectContext(ctx, &notifications, dataQ, args...); err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *Repository) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	const q = `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`
	if err := r.db.GetContext(ctx, &count, q, userID); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID, notificationID string) (*model.Notification, error) {
	const q = `
		UPDATE notifications
		SET is_read = true
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, type, title, body, data, is_read, created_at`

	var n model.Notification
	if err := r.db.GetContext(ctx, &n, q, notificationID, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &n, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	const q = `UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`
	_, err := r.db.ExecContext(ctx, q, userID)
	return err
}

// ──────────────────────────────────────────────
// Device Tokens
// ──────────────────────────────────────────────

func (r *Repository) UpsertDeviceToken(ctx context.Context, userID, fcmToken, deviceInfo string) error {
	const q = `
		INSERT INTO device_tokens (user_id, fcm_token, device_info)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, fcm_token) DO UPDATE
		SET device_info = EXCLUDED.device_info,
		    updated_at = NOW()`

	var di *string
	if deviceInfo != "" {
		di = &deviceInfo
	}

	_, err := r.db.ExecContext(ctx, q, userID, fcmToken, di)
	return err
}

// GetDeviceTokensByUserID returns all FCM tokens registered for the given user.
func (r *Repository) GetDeviceTokensByUserID(ctx context.Context, userID string) ([]string, error) {
	const q = `SELECT fcm_token FROM device_tokens WHERE user_id = $1`
	var tokens []string
	if err := r.db.SelectContext(ctx, &tokens, q, userID); err != nil {
		return nil, err
	}
	return tokens, nil
}

// ──────────────────────────────────────────────
// Sentinel errors
// ──────────────────────────────────────────────

type repoError string

func (e repoError) Error() string { return string(e) }

const (
	ErrNotFound repoError = "not found"
)
