package level

import (
	"context"
	"fmt"
	"time"

	"lingw/internal/domain"
	"lingw/internal/usecase/course"
)

type SessionService struct {
	courseStore domain.CourseStore
	progress    domain.ProgressStore
	checker     *Checker
	unlock      *course.UnlockService
}

func NewSessionService(
	courseStore domain.CourseStore,
	progress domain.ProgressStore,
	checker *Checker,
	unlock *course.UnlockService,
) *SessionService {
	return &SessionService{
		courseStore: courseStore,
		progress:    progress,
		checker:     checker,
		unlock:      unlock,
	}
}

func (s *SessionService) Start(ctx context.Context, userID int64, topicSlug, levelSlug string) (domain.LevelSession, error) {
	if err := s.unlock.EnsureLevelAvailable(ctx, userID, topicSlug, levelSlug); err != nil {
		return domain.LevelSession{}, err
	}
	exercises, err := s.courseStore.ListExercisesByLevel(ctx, levelSlug)
	if err != nil {
		return domain.LevelSession{}, err
	}
	session := domain.LevelSession{
		UserID:       userID,
		LevelSlug:    levelSlug,
		CurrentStep:  0,
		TotalSteps:   len(exercises),
		UpdatedAtUTC: time.Now().UTC(),
	}
	if err := s.progress.SetLevelStatus(ctx, userID, levelSlug, domain.LevelInProgress); err != nil {
		return domain.LevelSession{}, err
	}
	if err := s.progress.SaveSession(ctx, session); err != nil {
		return domain.LevelSession{}, err
	}
	return session, nil
}

func (s *SessionService) Submit(ctx context.Context, userID int64, levelSlug, answer string) (domain.LevelSession, bool, bool, error) {
	session, err := s.progress.GetSession(ctx, userID, levelSlug)
	if err != nil {
		return domain.LevelSession{}, false, false, err
	}
	exercises, err := s.courseStore.ListExercisesByLevel(ctx, levelSlug)
	if err != nil {
		return domain.LevelSession{}, false, false, err
	}
	if session.CurrentStep >= len(exercises) {
		return session, true, true, nil
	}
	ex, ok := exerciseAt(exercises, session.CurrentStep, userID, levelSlug)
	if !ok {
		return domain.LevelSession{}, false, false, domain.ErrNotFound
	}
	okAnswer, err := s.checker.Check(ctx, ex, answer)
	if err != nil {
		return domain.LevelSession{}, false, false, err
	}
	if err := s.progress.RecordAttempt(ctx, domain.ExerciseAttempt{
		UserID:      userID,
		LevelSlug:   levelSlug,
		ExercisePos: session.CurrentStep,
		Answer:      answer,
		IsCorrect:   okAnswer,
		AttemptedAt: time.Now().UTC(),
	}); err != nil {
		return domain.LevelSession{}, false, false, err
	}
	if !okAnswer {
		return session, false, false, nil
	}
	session.CurrentStep++
	session.UpdatedAtUTC = time.Now().UTC()
	if session.CurrentStep >= len(exercises) {
		if err := s.progress.SetLevelStatus(ctx, userID, levelSlug, domain.LevelCompleted); err != nil {
			return domain.LevelSession{}, true, false, fmt.Errorf("complete level: %w", err)
		}
		if err := s.progress.DeleteSession(ctx, userID, levelSlug); err != nil {
			return domain.LevelSession{}, true, false, err
		}
		return session, true, true, nil
	}
	if err := s.progress.SaveSession(ctx, session); err != nil {
		return domain.LevelSession{}, true, false, err
	}
	return session, true, false, nil
}

func (s *SessionService) CurrentExercise(ctx context.Context, userID int64, levelSlug string) (domain.LevelSession, domain.Exercise, error) {
	session, err := s.progress.GetSession(ctx, userID, levelSlug)
	if err != nil {
		return domain.LevelSession{}, domain.Exercise{}, err
	}
	exercises, err := s.courseStore.ListExercisesByLevel(ctx, levelSlug)
	if err != nil {
		return domain.LevelSession{}, domain.Exercise{}, err
	}
	if session.CurrentStep < 0 || session.CurrentStep >= len(exercises) {
		return domain.LevelSession{}, domain.Exercise{}, domain.ErrNotFound
	}
	ex, ok := exerciseAt(exercises, session.CurrentStep, userID, levelSlug)
	if !ok {
		return domain.LevelSession{}, domain.Exercise{}, domain.ErrNotFound
	}
	return session, ex, nil
}

func (s *SessionService) ActiveSession(ctx context.Context, userID int64) (domain.LevelSession, error) {
	return s.progress.GetActiveSession(ctx, userID)
}
