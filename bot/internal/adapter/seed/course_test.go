package seed

import (
	"context"
	"testing"
)

func TestNewCourseStoreLoadsEmbeddedSeeds(t *testing.T) {
	store, err := NewCourseStore()
	if err != nil {
		t.Fatalf("NewCourseStore: %v", err)
	}
	topics, err := store.ListTopics(context.Background())
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	if len(topics) < 6 {
		t.Fatalf("expected at least 6 topics, got %d", len(topics))
	}
	if topics[0].SortOrder > topics[len(topics)-1].SortOrder {
		t.Fatal("topics should be sorted by sort_order")
	}
}

func TestCourseStoreLevelsAndExercises(t *testing.T) {
	store, err := NewCourseStore()
	if err != nil {
		t.Fatalf("NewCourseStore: %v", err)
	}
	ctx := context.Background()
	topics, err := store.ListTopics(ctx)
	if err != nil || len(topics) == 0 {
		t.Fatalf("ListTopics: %v", err)
	}
	levels, err := store.ListLevelsByTopic(ctx, topics[0].Slug)
	if err != nil {
		t.Fatalf("ListLevelsByTopic: %v", err)
	}
	if len(levels) == 0 {
		t.Fatal("first topic should have levels")
	}
	exercises, err := store.ListExercisesByLevel(ctx, levels[0].Slug)
	if err != nil {
		t.Fatalf("ListExercisesByLevel: %v", err)
	}
	if len(exercises) == 0 {
		t.Fatal("level should have exercises")
	}
	for _, ex := range exercises {
		if ex.Type == "" {
			t.Fatalf("exercise missing type in %s", levels[0].Slug)
		}
		if ex.Data == nil {
			t.Fatalf("exercise %s missing data", ex.Type)
		}
		if _, ok := ex.Data["prompt"]; !ok {
			t.Fatalf("exercise %s missing prompt", ex.Type)
		}
	}
}

func TestVocabExercisesMayReferenceAudio(t *testing.T) {
	store, err := NewCourseStore()
	if err != nil {
		t.Fatalf("NewCourseStore: %v", err)
	}
	ctx := context.Background()
	levels, err := store.ListLevelsByTopic(ctx, "topic_01_cases")
	if err != nil {
		t.Fatalf("ListLevelsByTopic: %v", err)
	}
	foundAudio := false
	for _, lvl := range levels {
		exercises, err := store.ListExercisesByLevel(ctx, lvl.Slug)
		if err != nil {
			t.Fatalf("ListExercisesByLevel: %v", err)
		}
		for _, ex := range exercises {
			if ex.Type != "vocab" {
				continue
			}
			if audio, ok := ex.Data["audio"].(string); ok && audio != "" {
				foundAudio = true
			}
		}
	}
	if !foundAudio {
		t.Fatal("expected at least one vocab exercise with audio in topic 1")
	}
}
