package course

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"lingw/internal/domain"
	"lingw/internal/testutil"
)

func progressCourse() *testutil.CourseStore {
	return &testutil.CourseStore{
		Topics: []domain.Topic{
			{Slug: "topic_01", TitleRU: "Тема 1", SortOrder: 1},
			{Slug: "topic_02", TitleRU: "Тема 2", SortOrder: 2},
		},
		Levels: map[string][]domain.Level{
			"topic_01": {
				{Slug: "topic_01_level_01", TopicSlug: "topic_01", TitleRU: "Уровень 1", SortOrder: 1},
				{Slug: "topic_01_level_02", TopicSlug: "topic_01", TitleRU: "Уровень 2", SortOrder: 2},
			},
			"topic_02": {
				{Slug: "topic_02_level_01", TopicSlug: "topic_02", TitleRU: "Уровень 1", SortOrder: 1},
			},
		},
	}
}

func TestProgressSummaryCountsCompletedTopics(t *testing.T) {
	t.Parallel()
	progress := testutil.NewProgressStore()
	progress.MarkCompleted(1, "topic_01_level_01")
	progress.MarkCompleted(1, "topic_01_level_02")
	svc := NewProgressService(progressCourse(), progress, NewUnlockService(progressCourse(), progress))

	summary, err := svc.Summary(context.Background(), 1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.TotalLevels != 3 {
		t.Fatalf("total levels: %d", summary.TotalLevels)
	}
	if summary.CompletedLevels != 2 {
		t.Fatalf("completed levels: %d", summary.CompletedLevels)
	}
	if summary.Topics[0].State != TopicProgressCompleted {
		t.Fatalf("topic 1 state: %s", summary.Topics[0].State)
	}
	if summary.Topics[1].State != TopicProgressAvailable {
		t.Fatalf("topic 2 state: %s", summary.Topics[1].State)
	}
}

func TestProgressSummaryIncludesActiveSession(t *testing.T) {
	t.Parallel()
	progress := testutil.NewProgressStore()
	_ = progress.SaveSession(context.Background(), domain.LevelSession{
		UserID:       1,
		LevelSlug:    "topic_01_level_01",
		CurrentStep:  2,
		TotalSteps:   10,
		UpdatedAtUTC: time.Now().UTC(),
	})
	svc := NewProgressService(progressCourse(), progress, NewUnlockService(progressCourse(), progress))

	summary, err := svc.Summary(context.Background(), 1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Active == nil {
		t.Fatal("expected active session")
	}
	if summary.Active.Step != 3 || summary.Active.Total != 10 {
		t.Fatalf("active step: %+v", summary.Active)
	}
}

func TestFormatProgressMessageContainsTotals(t *testing.T) {
	t.Parallel()
	text := FormatProgressMessage(ProgressSummary{
		TotalLevels:     10,
		CompletedLevels: 3,
		Topics: []TopicProgress{
			{Topic: domain.Topic{TitleRU: "Падежи"}, Completed: 3, Total: 4, State: TopicProgressActive},
		},
	})
	if !strings.Contains(text, "3 / 10") {
		t.Fatalf("missing totals: %s", text)
	}
	if !strings.Contains(text, "Падежи") {
		t.Fatalf("missing topic title: %s", text)
	}
}

func TestProgressBar(t *testing.T) {
	t.Parallel()
	bar := progressBar(4, 8)
	if utf8.RuneCountInString(bar) != 16 {
		t.Fatalf("bar len: %d", utf8.RuneCountInString(bar))
	}
	if !strings.Contains(bar, "█") || !strings.Contains(bar, "░") {
		t.Fatalf("unexpected bar: %q", bar)
	}
}
