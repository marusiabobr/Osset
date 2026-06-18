package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"lingw/internal/domain"
	"lingw/seeds"
)

type CourseStore struct {
	topics      []domain.Topic
	levelsByTop map[string][]domain.Level
	exByLevel   map[string][]domain.Exercise
}

type levelFile struct {
	Slug      string            `json:"slug"`
	TopicSlug string            `json:"topic_slug"`
	TitleRU   string            `json:"title_ru"`
	SortOrder int               `json:"sort_order"`
	Exercises []exercisePayload `json:"exercises"`
}

type exercisePayload struct {
	Type      domain.ExerciseType    `json:"type"`
	SortOrder int                    `json:"sort_order"`
	Data      map[string]interface{} `json:"data"`
}

func NewCourseStore() (*CourseStore, error) {
	var topics []domain.Topic
	rawTopics, err := fs.ReadFile(seeds.Files, "topics.json")
	if err != nil {
		return nil, fmt.Errorf("read topics: %w", err)
	}
	if err := json.Unmarshal(rawTopics, &topics); err != nil {
		return nil, fmt.Errorf("decode topics: %w", err)
	}
	levelsByTop := map[string][]domain.Level{}
	exByLevel := map[string][]domain.Exercise{}
	err = fs.WalkDir(seeds.Files, "levels", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || path.Ext(p) != ".json" {
			return nil
		}
		content, err := fs.ReadFile(seeds.Files, p)
		if err != nil {
			return err
		}
		var lf levelFile
		if err := json.Unmarshal(content, &lf); err != nil {
			return fmt.Errorf("decode level %s: %w", p, err)
		}
		level := domain.Level{
			Slug:      lf.Slug,
			TopicSlug: lf.TopicSlug,
			TitleRU:   lf.TitleRU,
			SortOrder: lf.SortOrder,
		}
		levelsByTop[level.TopicSlug] = append(levelsByTop[level.TopicSlug], level)
		exercises := make([]domain.Exercise, 0, len(lf.Exercises))
		for idx, ex := range lf.Exercises {
			sortOrder := ex.SortOrder
			if sortOrder == 0 {
				sortOrder = idx + 1
			}
			exercises = append(exercises, domain.Exercise{
				LevelSlug: lf.Slug,
				Type:      ex.Type,
				SortOrder: sortOrder,
				Data:      ex.Data,
			})
		}
		exByLevel[lf.Slug] = exercises
		return nil
	})
	if err != nil {
		return nil, err
	}
	for slug := range levelsByTop {
		sort.Slice(levelsByTop[slug], func(i, j int) bool {
			return levelsByTop[slug][i].SortOrder < levelsByTop[slug][j].SortOrder
		})
	}
	sort.Slice(topics, func(i, j int) bool { return topics[i].SortOrder < topics[j].SortOrder })
	return &CourseStore{topics: topics, levelsByTop: levelsByTop, exByLevel: exByLevel}, nil
}

func (s *CourseStore) ListTopics(context.Context) ([]domain.Topic, error) {
	return append([]domain.Topic(nil), s.topics...), nil
}

func (s *CourseStore) ListLevelsByTopic(_ context.Context, topicSlug string) ([]domain.Level, error) {
	levels, ok := s.levelsByTop[topicSlug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return append([]domain.Level(nil), levels...), nil
}

func (s *CourseStore) GetLevel(ctx context.Context, levelSlug string) (domain.Level, error) {
	topics, err := s.ListTopics(ctx)
	if err != nil {
		return domain.Level{}, err
	}
	for _, t := range topics {
		levels, err := s.ListLevelsByTopic(ctx, t.Slug)
		if err != nil {
			continue
		}
		for _, l := range levels {
			if l.Slug == levelSlug {
				return l, nil
			}
		}
	}
	return domain.Level{}, domain.ErrNotFound
}

func (s *CourseStore) ListExercisesByLevel(_ context.Context, levelSlug string) ([]domain.Exercise, error) {
	ex, ok := s.exByLevel[levelSlug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return append([]domain.Exercise(nil), ex...), nil
}
