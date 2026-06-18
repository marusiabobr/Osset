package scheduler

import (
	"context"
	"log/slog"
	"time"

	reminduc "lingw/internal/usecase/reminder"
)

type Reminder struct {
	logger   *slog.Logger
	service  *reminduc.Service
	interval time.Duration
}

func NewReminder(logger *slog.Logger, service *reminduc.Service, intervalMinutes int) *Reminder {
	return &Reminder{
		logger:   logger,
		service:  service,
		interval: time.Duration(intervalMinutes) * time.Minute,
	}
}

func (r *Reminder) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.service.Run(ctx); err != nil {
				r.logger.Error("reminder run failed", "err", err)
			}
		}
	}
}
