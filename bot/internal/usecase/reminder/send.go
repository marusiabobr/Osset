package reminder

import (
	"context"
	"time"

	"lingw/internal/domain"
)

type Sender interface {
	SendReminder(ctx context.Context, telegramID int64, text string) error
}

type Service struct {
	users  domain.UserStore
	sender Sender
}

func NewService(users domain.UserStore, sender Sender) *Service {
	return &Service{users: users, sender: sender}
}

func (s *Service) Run(ctx context.Context) error {
	candidates, err := s.users.ListReminderCandidates(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, user := range candidates {
		if !user.RemindersEnabled || user.ReminderHour != now.Hour() {
			continue
		}
		if user.LastActivityAt != nil && sameUTCDate(*user.LastActivityAt, now) {
			continue
		}
		if user.LastReminderSentAt != nil && sameUTCDate(*user.LastReminderSentAt, now) {
			continue
		}
		if err := s.sender.SendReminder(ctx, user.TelegramID, "Пора продолжить обучение в Lingw."); err != nil {
			continue
		}
		_ = s.users.MarkReminderSent(ctx, user.ID)
	}
	return nil
}

func sameUTCDate(a, b time.Time) bool {
	aa := a.UTC()
	bb := b.UTC()
	return aa.Year() == bb.Year() && aa.Month() == bb.Month() && aa.Day() == bb.Day()
}
