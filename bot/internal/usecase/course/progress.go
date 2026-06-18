package course

import (
	"context"
	"fmt"
	"strings"

	"lingw/internal/domain"
)

type TopicProgressState string

const (
	TopicProgressLocked    TopicProgressState = "locked"
	TopicProgressAvailable TopicProgressState = "available"
	TopicProgressActive    TopicProgressState = "active"
	TopicProgressCompleted TopicProgressState = "completed"
)

type TopicProgress struct {
	Topic     domain.Topic
	Completed int
	Total     int
	State     TopicProgressState
}

type ActiveProgress struct {
	LevelTitle string
	TopicTitle string
	Step       int
	Total      int
}

type ProgressSummary struct {
	TotalLevels      int
	CompletedLevels  int
	InProgressLevels int
	Topics           []TopicProgress
	Active           *ActiveProgress
}

type ProgressService struct {
	courses  domain.CourseStore
	progress domain.ProgressStore
	unlock   *UnlockService
}

func NewProgressService(courses domain.CourseStore, progress domain.ProgressStore, unlock *UnlockService) *ProgressService {
	return &ProgressService{courses: courses, progress: progress, unlock: unlock}
}

func (s *ProgressService) Summary(ctx context.Context, userID int64) (ProgressSummary, error) {
	topics, err := s.courses.ListTopics(ctx)
	if err != nil {
		return ProgressSummary{}, err
	}
	summary := ProgressSummary{Topics: make([]TopicProgress, 0, len(topics))}
	for _, topic := range topics {
		tp, err := s.topicProgress(ctx, userID, topic)
		if err != nil {
			return ProgressSummary{}, err
		}
		summary.Topics = append(summary.Topics, tp)
		summary.TotalLevels += tp.Total
		summary.CompletedLevels += tp.Completed
	}
	summary.InProgressLevels = s.countInProgressLevels(ctx, userID, topics)
	if session, err := s.progress.GetActiveSession(ctx, userID); err == nil {
		level, err := s.courses.GetLevel(ctx, session.LevelSlug)
		if err == nil {
			topicTitle := level.TopicSlug
			for _, topic := range topics {
				if topic.Slug == level.TopicSlug {
					topicTitle = topic.TitleRU
					break
				}
			}
			summary.Active = &ActiveProgress{
				LevelTitle: level.TitleRU,
				TopicTitle: topicTitle,
				Step:       session.CurrentStep + 1,
				Total:      session.TotalSteps,
			}
		}
	}
	return summary, nil
}

func (s *ProgressService) topicProgress(ctx context.Context, userID int64, topic domain.Topic) (TopicProgress, error) {
	levels, err := s.courses.ListLevelsByTopic(ctx, topic.Slug)
	if err != nil {
		return TopicProgress{}, err
	}
	tp := TopicProgress{
		Topic: topic,
		Total: len(levels),
		State: TopicProgressLocked,
	}
	if err := s.unlock.EnsureTopicAvailable(ctx, userID, topic.Slug); err != nil {
		return tp, nil
	}
	for _, level := range levels {
		p, err := s.progress.GetLevelProgress(ctx, userID, level.Slug)
		if err != nil {
			return TopicProgress{}, err
		}
		if p.Status == domain.LevelCompleted {
			tp.Completed++
		}
	}
	switch {
	case tp.Total > 0 && tp.Completed >= tp.Total:
		tp.State = TopicProgressCompleted
	case tp.Completed > 0:
		tp.State = TopicProgressActive
	default:
		tp.State = TopicProgressAvailable
	}
	return tp, nil
}

func (s *ProgressService) countInProgressLevels(ctx context.Context, userID int64, topics []domain.Topic) int {
	count := 0
	for _, topic := range topics {
		levels, err := s.courses.ListLevelsByTopic(ctx, topic.Slug)
		if err != nil {
			continue
		}
		for _, level := range levels {
			p, err := s.progress.GetLevelProgress(ctx, userID, level.Slug)
			if err != nil {
				continue
			}
			if p.Status == domain.LevelInProgress {
				count++
			}
		}
	}
	return count
}

func FormatProgressMessage(summary ProgressSummary) string {
	var b strings.Builder
	b.WriteString("📈 Мой прогресс\n\n")

	if summary.TotalLevels == 0 {
		b.WriteString("Курс пока пуст. Загляните позже.")
		return b.String()
	}

	percent := summary.CompletedLevels * 100 / summary.TotalLevels
	b.WriteString(fmt.Sprintf("Общий итог: %d / %d уровней (%d%%)\n", summary.CompletedLevels, summary.TotalLevels, percent))
	b.WriteString(progressBar(summary.CompletedLevels, summary.TotalLevels))
	b.WriteByte('\n')

	if summary.Active != nil {
		b.WriteString(fmt.Sprintf(
			"\n▶️ Сейчас: %s\n   %s — задание %d из %d\n",
			summary.Active.TopicTitle,
			summary.Active.LevelTitle,
			summary.Active.Step,
			summary.Active.Total,
		))
	} else if summary.CompletedLevels < summary.TotalLevels {
		b.WriteString("\n💡 Продолжите обучение в разделе «Учиться».\n")
	} else {
		b.WriteString("\n🏆 Весь курс пройден! Отличная работа.\n")
	}

	b.WriteString("\nТемы:\n")
	for _, topic := range summary.Topics {
		b.WriteString(fmt.Sprintf("%s %s — %d/%d\n",
			topicStateIcon(topic.State),
			topic.Topic.TitleRU,
			topic.Completed,
			topic.Total,
		))
	}

	if summary.InProgressLevels > 0 {
		b.WriteString(fmt.Sprintf("\n🟡 Уровней в процессе: %d", summary.InProgressLevels))
	}
	return b.String()
}

func topicStateIcon(state TopicProgressState) string {
	switch state {
	case TopicProgressCompleted:
		return "✅"
	case TopicProgressActive:
		return "📖"
	case TopicProgressAvailable:
		return "▶️"
	default:
		return "🔒"
	}
}

func progressBar(completed, total int) string {
	const width = 16
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	filled := completed * width / total
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
