package course

import (
	"context"

	"lingw/internal/domain"
)

type UnlockService struct {
	courses  domain.CourseStore
	progress domain.ProgressStore
}

func NewUnlockService(courses domain.CourseStore, progress domain.ProgressStore) *UnlockService {
	return &UnlockService{courses: courses, progress: progress}
}

func (s *UnlockService) EnsureTopicAvailable(ctx context.Context, userID int64, topicSlug string) error {
	topics, err := s.courses.ListTopics(ctx)
	if err != nil {
		return err
	}
	for i, topic := range topics {
		if topic.Slug != topicSlug {
			continue
		}
		if i == 0 {
			return nil
		}
		prevTopic := topics[i-1]
		return s.ensureAllLevelsCompleted(ctx, userID, prevTopic.Slug)
	}
	return domain.ErrNotFound
}

func (s *UnlockService) ensureAllLevelsCompleted(ctx context.Context, userID int64, topicSlug string) error {
	levels, err := s.courses.ListLevelsByTopic(ctx, topicSlug)
	if err != nil {
		return err
	}
	for _, level := range levels {
		p, err := s.progress.GetLevelProgress(ctx, userID, level.Slug)
		if err != nil {
			return err
		}
		if p.Status != domain.LevelCompleted {
			return domain.ErrTopicLocked
		}
	}
	return nil
}

func (s *UnlockService) EnsureLevelAvailable(ctx context.Context, userID int64, topicSlug, levelSlug string) error {
	if err := s.EnsureTopicAvailable(ctx, userID, topicSlug); err != nil {
		return err
	}
	levels, err := s.courses.ListLevelsByTopic(ctx, topicSlug)
	if err != nil {
		return err
	}
	for i, level := range levels {
		if level.Slug != levelSlug {
			continue
		}
		if i == 0 {
			return nil
		}
		prev := levels[i-1]
		progress, err := s.progress.GetLevelProgress(ctx, userID, prev.Slug)
		if err != nil {
			return err
		}
		if progress.Status != domain.LevelCompleted {
			return domain.ErrLevelLocked
		}
		return nil
	}
	return domain.ErrNotFound
}

func (s *UnlockService) NextLevel(ctx context.Context, userID int64, topicSlug, currentLevelSlug string) (domain.Level, bool, error) {
	levels, err := s.courses.ListLevelsByTopic(ctx, topicSlug)
	if err != nil {
		return domain.Level{}, false, err
	}
	for i, level := range levels {
		if level.Slug != currentLevelSlug {
			continue
		}
		if i+1 >= len(levels) {
			return domain.Level{}, false, nil
		}
		next := levels[i+1]
		if err := s.EnsureLevelAvailable(ctx, userID, topicSlug, next.Slug); err != nil {
			return domain.Level{}, false, nil
		}
		return next, true, nil
	}
	return domain.Level{}, false, domain.ErrNotFound
}

