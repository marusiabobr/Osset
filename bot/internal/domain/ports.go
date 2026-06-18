package domain

import "context"

type CourseStore interface {
	ListTopics(ctx context.Context) ([]Topic, error)
	ListLevelsByTopic(ctx context.Context, topicSlug string) ([]Level, error)
	GetLevel(ctx context.Context, levelSlug string) (Level, error)
	ListExercisesByLevel(ctx context.Context, levelSlug string) ([]Exercise, error)
}

type LexiconStore interface {
	Resolve(ctx context.Context, ref string) (LexemeDisplay, error)
	ResolveForm(ctx context.Context, ref string) (WordFormDisplay, error)
	AcceptedAnswers(ctx context.Context, refs []string) ([]string, error)
}

type UserStore interface {
	UpsertUser(ctx context.Context, telegramID int64, username string) (User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (User, error)
	UpdateActivity(ctx context.Context, userID int64) error
	ListReminderCandidates(ctx context.Context) ([]User, error)
	MarkReminderSent(ctx context.Context, userID int64) error
}

type ProgressStore interface {
	GetLevelProgress(ctx context.Context, userID int64, levelSlug string) (LevelProgress, error)
	SetLevelStatus(ctx context.Context, userID int64, levelSlug string, status LevelStatus) error
	RecordAttempt(ctx context.Context, attempt ExerciseAttempt) error
	SaveSession(ctx context.Context, session LevelSession) error
	GetSession(ctx context.Context, userID int64, levelSlug string) (LevelSession, error)
	GetActiveSession(ctx context.Context, userID int64) (LevelSession, error)
	DeleteSession(ctx context.Context, userID int64, levelSlug string) error
	ListCompletedLevelsByTopic(ctx context.Context, userID int64, topicSlug string) ([]string, error)
}
