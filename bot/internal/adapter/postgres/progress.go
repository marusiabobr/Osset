package postgres

import (
	"context"
	"fmt"
	"strings"

	"lingw/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProgressStore struct {
	pool *pgxpool.Pool
}

func NewProgressStore(pool *pgxpool.Pool) *ProgressStore {
	return &ProgressStore{pool: pool}
}

func (s *ProgressStore) GetLevelProgress(ctx context.Context, userID int64, levelSlug string) (domain.LevelProgress, error) {
	const q = `
SELECT user_id, level_slug, status, completed_at, attempts_count
FROM user_level_progress
WHERE user_id = $1 AND level_slug = $2`
	var p domain.LevelProgress
	err := s.pool.QueryRow(ctx, q, userID, levelSlug).Scan(&p.UserID, &p.LevelSlug, &p.Status, &p.CompletedAt, &p.Attempts)
	if err == nil {
		return p, nil
	}
	return domain.LevelProgress{UserID: userID, LevelSlug: levelSlug, Status: domain.LevelLocked}, nil
}

func (s *ProgressStore) SetLevelStatus(ctx context.Context, userID int64, levelSlug string, status domain.LevelStatus) error {
	const q = `
INSERT INTO user_level_progress (user_id, level_slug, status, completed_at, attempts_count)
VALUES ($1, $2, $3, CASE WHEN $3 = 'completed' THEN NOW() ELSE NULL END, 0)
ON CONFLICT (user_id, level_slug)
DO UPDATE SET status = EXCLUDED.status,
              completed_at = CASE WHEN EXCLUDED.status = 'completed' THEN NOW() ELSE user_level_progress.completed_at END`
	if _, err := s.pool.Exec(ctx, q, userID, levelSlug, status); err != nil {
		return fmt.Errorf("set level status: %w", err)
	}
	return nil
}

func (s *ProgressStore) RecordAttempt(ctx context.Context, attempt domain.ExerciseAttempt) error {
	const qAttempt = `
INSERT INTO user_exercise_attempts (user_id, level_slug, exercise_pos, answer, is_correct, attempted_at)
VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := s.pool.Exec(ctx, qAttempt, attempt.UserID, attempt.LevelSlug, attempt.ExercisePos, attempt.Answer, attempt.IsCorrect, attempt.AttemptedAt); err != nil {
		return fmt.Errorf("record attempt: %w", err)
	}
	const qInc = `
INSERT INTO user_level_progress (user_id, level_slug, status, attempts_count)
VALUES ($1, $2, 'in_progress', 1)
ON CONFLICT (user_id, level_slug)
DO UPDATE SET attempts_count = user_level_progress.attempts_count + 1`
	if _, err := s.pool.Exec(ctx, qInc, attempt.UserID, attempt.LevelSlug); err != nil {
		return fmt.Errorf("inc attempts: %w", err)
	}
	return nil
}

func (s *ProgressStore) SaveSession(ctx context.Context, session domain.LevelSession) error {
	const q = `
INSERT INTO user_level_sessions (user_id, level_slug, current_step, total_steps, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, level_slug)
DO UPDATE SET current_step = EXCLUDED.current_step,
              total_steps = EXCLUDED.total_steps,
              updated_at = EXCLUDED.updated_at`
	if _, err := s.pool.Exec(ctx, q, session.UserID, session.LevelSlug, session.CurrentStep, session.TotalSteps, session.UpdatedAtUTC); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

func (s *ProgressStore) GetSession(ctx context.Context, userID int64, levelSlug string) (domain.LevelSession, error) {
	const q = `
SELECT user_id, level_slug, current_step, total_steps, updated_at
FROM user_level_sessions
WHERE user_id = $1 AND level_slug = $2`
	var session domain.LevelSession
	if err := s.pool.QueryRow(ctx, q, userID, levelSlug).Scan(
		&session.UserID, &session.LevelSlug, &session.CurrentStep, &session.TotalSteps, &session.UpdatedAtUTC,
	); err != nil {
		return domain.LevelSession{}, domain.ErrNotFound
	}
	return session, nil
}

func (s *ProgressStore) GetActiveSession(ctx context.Context, userID int64) (domain.LevelSession, error) {
	const q = `
SELECT user_id, level_slug, current_step, total_steps, updated_at
FROM user_level_sessions
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT 1`
	var session domain.LevelSession
	if err := s.pool.QueryRow(ctx, q, userID).Scan(
		&session.UserID, &session.LevelSlug, &session.CurrentStep, &session.TotalSteps, &session.UpdatedAtUTC,
	); err != nil {
		return domain.LevelSession{}, domain.ErrNotFound
	}
	return session, nil
}

func (s *ProgressStore) DeleteSession(ctx context.Context, userID int64, levelSlug string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM user_level_sessions WHERE user_id = $1 AND level_slug = $2`, userID, levelSlug)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func topicLevelLikePattern(topicSlug string) string {
	parts := strings.SplitN(topicSlug, "_", 3)
	if len(parts) >= 2 {
		return parts[0] + "_" + parts[1] + "_%"
	}
	return topicSlug + "_%"
}

func (s *ProgressStore) ListCompletedLevelsByTopic(ctx context.Context, userID int64, topicSlug string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT level_slug
FROM user_level_progress
WHERE user_id = $1 AND status = 'completed' AND level_slug LIKE $2`, userID, topicLevelLikePattern(topicSlug))
	if err != nil {
		return nil, fmt.Errorf("list completed levels: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	return out, nil
}
