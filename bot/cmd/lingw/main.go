package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"lingw/internal/adapter/postgres"
	"lingw/internal/adapter/scheduler"
	seedadapter "lingw/internal/adapter/seed"
	tgadapter "lingw/internal/adapter/telegram"
	"lingw/assets/audio"
	"lingw/internal/config"
	"lingw/internal/domain"
	courseuc "lingw/internal/usecase/course"
	leveluc "lingw/internal/usecase/level"
	reminduc "lingw/internal/usecase/reminder"
	useruc "lingw/internal/usecase/user"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	userStore := postgres.NewUserStore(pool)
	progressStore := postgres.NewProgressStore(pool)

	var courseStore domain.CourseStore
	switch cfg.ContentSource {
	case "seed":
		courseStore, err = seedadapter.NewCourseStore()
	case "postgres":
		courseStore = postgres.NewCourseStore()
	default:
		logger.Error("unsupported CONTENT_SOURCE", "value", cfg.ContentSource)
		os.Exit(1)
	}
	if err != nil {
		logger.Error("failed to init course store", "err", err)
		os.Exit(1)
	}

	var lexiconStore domain.LexiconStore
	switch cfg.LexiconSource {
	case "stub":
		lexiconStore, err = seedadapter.NewLexiconStore()
	case "postgres":
		lexiconStore = postgres.NewLexiconStore()
	default:
		logger.Error("unsupported LEXICON_SOURCE", "value", cfg.LexiconSource)
		os.Exit(1)
	}
	if err != nil {
		logger.Error("failed to init lexicon store", "err", err)
		os.Exit(1)
	}

	registerSvc := useruc.NewRegisterService(userStore)
	unlockSvc := courseuc.NewUnlockService(courseStore, progressStore)
	listSvc := courseuc.NewListService(courseStore, progressStore)
	progressSvc := courseuc.NewProgressService(courseStore, progressStore, unlockSvc)
	checkerSvc := leveluc.NewChecker(lexiconStore)
	sessionSvc := leveluc.NewSessionService(courseStore, progressStore, checkerSvc, unlockSvc)
	audioStore := audio.NewStore(cfg.AudioDir)

	tg, err := tgadapter.New(ctx, tgadapter.Deps{
		Token:    cfg.TelegramBotToken,
		Debug:    cfg.BotDebug,
		Log:      logger,
		Register: registerSvc,
		List:     listSvc,
		Unlock:   unlockSvc,
		Progress: progressSvc,
		Session:  sessionSvc,
		Lexicon:  lexiconStore,
		Audio:    audioStore,
	})
	if err != nil {
		logger.Error("failed to init telegram adapter", "err", err)
		os.Exit(1)
	}

	reminderSvc := reminduc.NewService(userStore, tg)
	reminderScheduler := scheduler.NewReminder(logger, reminderSvc, cfg.ReminderTickMinutes)
	go reminderScheduler.Start(ctx)

	if err := tg.Start(ctx); err != nil {
		logger.Error("bot stopped with error", "err", err)
		os.Exit(1)
	}
}
