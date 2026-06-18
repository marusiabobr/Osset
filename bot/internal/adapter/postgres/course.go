package postgres

import (
	"context"

	"lingw/internal/domain"
)

type CourseStore struct{}

func NewCourseStore() *CourseStore { return &CourseStore{} }

func (s *CourseStore) ListTopics(context.Context) ([]domain.Topic, error) {
	return nil, domain.ErrNotImplemented
}

func (s *CourseStore) ListLevelsByTopic(context.Context, string) ([]domain.Level, error) {
	return nil, domain.ErrNotImplemented
}

func (s *CourseStore) GetLevel(context.Context, string) (domain.Level, error) {
	return domain.Level{}, domain.ErrNotImplemented
}

func (s *CourseStore) ListExercisesByLevel(context.Context, string) ([]domain.Exercise, error) {
	return nil, domain.ErrNotImplemented
}
