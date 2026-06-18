package postgres

import (
	"context"
	"fmt"
	"time"

	"lingw/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) UpsertUser(ctx context.Context, telegramID int64, username string) (domain.User, error) {
	q := `
INSERT INTO users (telegram_id, username)
VALUES ($1, $2)
ON CONFLICT (telegram_id)
DO UPDATE SET username = EXCLUDED.username
RETURNING id, telegram_id, username, timezone, reminder_hour, reminders_enabled, created_at`
	var u domain.User
	if err := s.pool.QueryRow(ctx, q, telegramID, username).Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.Timezone, &u.ReminderHour, &u.RemindersEnabled, &u.CreatedAt,
	); err != nil {
		return domain.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

func (s *UserStore) GetByTelegramID(ctx context.Context, telegramID int64) (domain.User, error) {
	q := `
SELECT id, telegram_id, username, timezone, reminder_hour, reminders_enabled, last_activity_at, last_reminder_sent_at, created_at
FROM users WHERE telegram_id = $1`
	var u domain.User
	if err := s.pool.QueryRow(ctx, q, telegramID).Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.Timezone, &u.ReminderHour, &u.RemindersEnabled, &u.LastActivityAt, &u.LastReminderSentAt, &u.CreatedAt,
	); err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *UserStore) UpdateActivity(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET last_activity_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("update activity: %w", err)
	}
	return nil
}

func (s *UserStore) ListReminderCandidates(ctx context.Context) ([]domain.User, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, telegram_id, username, timezone, reminder_hour, reminders_enabled, last_activity_at, last_reminder_sent_at, created_at
FROM users
WHERE reminders_enabled = true`)
	if err != nil {
		return nil, fmt.Errorf("list reminder candidates: %w", err)
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.TelegramID, &u.Username, &u.Timezone, &u.ReminderHour, &u.RemindersEnabled, &u.LastActivityAt, &u.LastReminderSentAt, &u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reminder candidate: %w", err)
		}
		out = append(out, u)
	}
	return out, nil
}

func (s *UserStore) MarkReminderSent(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET last_reminder_sent_at = $2 WHERE id = $1`, userID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("mark reminder sent: %w", err)
	}
	return nil
}
