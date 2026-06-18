package course

import (
	"context"
	"testing"

	"lingw/internal/domain"
	"lingw/internal/testutil"
)

func sampleCourse() *testutil.CourseStore {
	return &testutil.CourseStore{
		Topics: []domain.Topic{
			{Slug: "topic_01", SortOrder: 1},
			{Slug: "topic_02", SortOrder: 2},
		},
		Levels: map[string][]domain.Level{
			"topic_01": {
				{Slug: "topic_01_level_01", TopicSlug: "topic_01", SortOrder: 1},
				{Slug: "topic_01_level_02", TopicSlug: "topic_01", SortOrder: 2},
			},
			"topic_02": {
				{Slug: "topic_02_level_01", TopicSlug: "topic_02", SortOrder: 1},
			},
		},
	}
}

func TestEnsureTopicAvailableFirstTopic(t *testing.T) {
	t.Parallel()
	progress := testutil.NewProgressStore()
	svc := NewUnlockService(sampleCourse(), progress)
	if err := svc.EnsureTopicAvailable(context.Background(), 1, "topic_01"); err != nil {
		t.Fatalf("first topic should be open: %v", err)
	}
}

func TestEnsureTopicAvailableLockedWithoutProgress(t *testing.T) {
	t.Parallel()
	progress := testutil.NewProgressStore()
	svc := NewUnlockService(sampleCourse(), progress)
	err := svc.EnsureTopicAvailable(context.Background(), 1, "topic_02")
	if err != domain.ErrTopicLocked {
		t.Fatalf("want ErrTopicLocked, got %v", err)
	}
}

func TestEnsureTopicAvailableAfterCompletingPrevious(t *testing.T) {
	t.Parallel()
	progress := testutil.NewProgressStore()
	progress.MarkCompleted(1, "topic_01_level_01")
	progress.MarkCompleted(1, "topic_01_level_02")
	svc := NewUnlockService(sampleCourse(), progress)
	if err := svc.EnsureTopicAvailable(context.Background(), 1, "topic_02"); err != nil {
		t.Fatalf("topic 2 should unlock: %v", err)
	}
}

func TestEnsureLevelAvailableRequiresPreviousLevel(t *testing.T) {
	t.Parallel()
	progress := testutil.NewProgressStore()
	svc := NewUnlockService(sampleCourse(), progress)
	if err := svc.EnsureLevelAvailable(context.Background(), 1, "topic_01", "topic_01_level_01"); err != nil {
		t.Fatalf("first level should be open: %v", err)
	}
	err := svc.EnsureLevelAvailable(context.Background(), 1, "topic_01", "topic_01_level_02")
	if err != domain.ErrLevelLocked {
		t.Fatalf("want ErrLevelLocked, got %v", err)
	}
	progress.MarkCompleted(1, "topic_01_level_01")
	if err := svc.EnsureLevelAvailable(context.Background(), 1, "topic_01", "topic_01_level_02"); err != nil {
		t.Fatalf("second level should open: %v", err)
	}
}

func TestNextLevel(t *testing.T) {
	t.Parallel()
	progress := testutil.NewProgressStore()
	progress.MarkCompleted(1, "topic_01_level_01")
	svc := NewUnlockService(sampleCourse(), progress)
	next, ok, err := svc.NextLevel(context.Background(), 1, "topic_01", "topic_01_level_01")
	if err != nil || !ok {
		t.Fatalf("NextLevel: ok=%v err=%v", ok, err)
	}
	if next.Slug != "topic_01_level_02" {
		t.Fatalf("unexpected next: %s", next.Slug)
	}
	_, ok, err = svc.NextLevel(context.Background(), 1, "topic_01", "topic_01_level_02")
	if err != nil || ok {
		t.Fatalf("last level should have no next: ok=%v err=%v", ok, err)
	}
}
