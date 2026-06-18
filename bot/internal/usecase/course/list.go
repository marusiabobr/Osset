package course

import (
	"context"

	"lingw/internal/domain"
)

type ListService struct {
	courses  domain.CourseStore
	progress domain.ProgressStore
}

func NewListService(courses domain.CourseStore, progress domain.ProgressStore) *ListService {
	return &ListService{courses: courses, progress: progress}
}

type LevelWithStatus struct {
	Level  domain.Level
	Status domain.LevelStatus
}

func (s *ListService) Topics(ctx context.Context) ([]domain.Topic, error) {
	return s.courses.ListTopics(ctx)
}

func (s *ListService) LevelsByTopic(ctx context.Context, userID int64, topicSlug string) ([]LevelWithStatus, error) {
	levels, err := s.courses.ListLevelsByTopic(ctx, topicSlug)
	if err != nil {
		return nil, err
	}
	result := make([]LevelWithStatus, 0, len(levels))
	for _, level := range levels {
		p, err := s.progress.GetLevelProgress(ctx, userID, level.Slug)
		if err != nil {
			return nil, err
		}
		result = append(result, LevelWithStatus{Level: level, Status: p.Status})
	}
	return result, nil
}

func (s *ListService) GetLevel(ctx context.Context, levelSlug string) (domain.Level, error) {
	return s.courses.GetLevel(ctx, levelSlug)
}
