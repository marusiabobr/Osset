package domain

import "time"

type LevelStatus string

const (
	LevelLocked     LevelStatus = "locked"
	LevelAvailable  LevelStatus = "available"
	LevelInProgress LevelStatus = "in_progress"
	LevelCompleted  LevelStatus = "completed"
)

type LevelProgress struct {
	UserID      int64
	LevelSlug   string
	Status      LevelStatus
	CompletedAt *time.Time
	Attempts    int
}

type LevelSession struct {
	UserID       int64
	LevelSlug    string
	CurrentStep  int
	TotalSteps   int
	UpdatedAtUTC time.Time
}

type ExerciseAttempt struct {
	UserID      int64
	LevelSlug   string
	ExercisePos int
	Answer      string
	IsCorrect   bool
	AttemptedAt time.Time
}
