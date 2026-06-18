// Package testutil provides in-memory fakes for unit tests.
// It is not used in production code paths.
package testutil

import (
	"context"
	"fmt"
	"sync"

	"lingw/internal/domain"
)

// CourseStore is an in-memory CourseStore for tests.
type CourseStore struct {
	Topics  []domain.Topic
	Levels  map[string][]domain.Level
	Exercises map[string][]domain.Exercise
}

func (s *CourseStore) ListTopics(_ context.Context) ([]domain.Topic, error) {
	return s.Topics, nil
}

func (s *CourseStore) ListLevelsByTopic(_ context.Context, topicSlug string) ([]domain.Level, error) {
	return s.Levels[topicSlug], nil
}

func (s *CourseStore) GetLevel(_ context.Context, levelSlug string) (domain.Level, error) {
	for _, levels := range s.Levels {
		for _, l := range levels {
			if l.Slug == levelSlug {
				return l, nil
			}
		}
	}
	return domain.Level{}, domain.ErrNotFound
}

func (s *CourseStore) ListExercisesByLevel(_ context.Context, levelSlug string) ([]domain.Exercise, error) {
	return s.Exercises[levelSlug], nil
}

// ProgressStore is an in-memory ProgressStore for tests.
type ProgressStore struct {
	mu        sync.Mutex
	byLevel   map[string]domain.LevelProgress
	completed map[string]bool
	sessions  map[string]domain.LevelSession
}

func NewProgressStore() *ProgressStore {
	return &ProgressStore{
		byLevel:   make(map[string]domain.LevelProgress),
		completed: make(map[string]bool),
		sessions:  make(map[string]domain.LevelSession),
	}
}

func (s *ProgressStore) key(userID int64, levelSlug string) string {
	return fmt.Sprintf("%d:%s", userID, levelSlug)
}

func (s *ProgressStore) GetLevelProgress(_ context.Context, userID int64, levelSlug string) (domain.LevelProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(userID, levelSlug)
	if p, ok := s.byLevel[k]; ok {
		return p, nil
	}
	status := domain.LevelAvailable
	if s.completed[k] {
		status = domain.LevelCompleted
	}
	return domain.LevelProgress{UserID: userID, LevelSlug: levelSlug, Status: status}, nil
}

func (s *ProgressStore) SetLevelStatus(_ context.Context, userID int64, levelSlug string, status domain.LevelStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(userID, levelSlug)
	p := s.byLevel[k]
	p.UserID = userID
	p.LevelSlug = levelSlug
	p.Status = status
	s.byLevel[k] = p
	if status == domain.LevelCompleted {
		s.completed[k] = true
	}
	return nil
}

func (s *ProgressStore) RecordAttempt(_ context.Context, _ domain.ExerciseAttempt) error { return nil }
func (s *ProgressStore) SaveSession(_ context.Context, session domain.LevelSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[s.key(session.UserID, session.LevelSlug)] = session
	return nil
}
func (s *ProgressStore) GetSession(_ context.Context, userID int64, levelSlug string) (domain.LevelSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[s.key(userID, levelSlug)]
	if !ok {
		return domain.LevelSession{}, domain.ErrNotFound
	}
	return session, nil
}
func (s *ProgressStore) GetActiveSession(_ context.Context, userID int64) (domain.LevelSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest domain.LevelSession
	var found bool
	for _, session := range s.sessions {
		if session.UserID != userID {
			continue
		}
		if !found || session.UpdatedAtUTC.After(latest.UpdatedAtUTC) {
			latest = session
			found = true
		}
	}
	if !found {
		return domain.LevelSession{}, domain.ErrNotFound
	}
	return latest, nil
}
func (s *ProgressStore) DeleteSession(_ context.Context, userID int64, levelSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, s.key(userID, levelSlug))
	return nil
}
func (s *ProgressStore) ListCompletedLevelsByTopic(_ context.Context, _ int64, _ string) ([]string, error) {
	return nil, nil
}

func (s *ProgressStore) MarkCompleted(userID int64, levelSlug string) {
	_ = s.SetLevelStatus(context.Background(), userID, levelSlug, domain.LevelCompleted)
}

// StubLexicon resolves refs to themselves for literal-based checks.
type StubLexicon struct{}

func (StubLexicon) Resolve(_ context.Context, ref string) (domain.LexemeDisplay, error) {
	return domain.LexemeDisplay{OS: ref, RU: ref}, nil
}

func (StubLexicon) ResolveForm(_ context.Context, ref string) (domain.WordFormDisplay, error) {
	return domain.WordFormDisplay{OS: ref, RU: ref}, nil
}

func (StubLexicon) AcceptedAnswers(_ context.Context, refs []string) ([]string, error) {
	return refs, nil
}
