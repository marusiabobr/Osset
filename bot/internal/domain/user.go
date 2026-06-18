package domain

import "time"

type User struct {
	ID                 int64
	TelegramID         int64
	Username           string
	Timezone           string
	ReminderHour       int
	RemindersEnabled   bool
	LastActivityAt     *time.Time
	LastReminderSentAt *time.Time
	CreatedAt          time.Time
}
