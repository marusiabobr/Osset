package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TelegramBotToken    string
	DatabaseURL         string
	ContentSource       string
	LexiconSource       string
	BotDebug            bool
	HealthPort          string
	ReminderTickMinutes int
	AudioDir            string
}

func Load() (Config, error) {
	cfg := Config{
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		ContentSource:       envOr("CONTENT_SOURCE", "seed"),
		LexiconSource:       envOr("LEXICON_SOURCE", "stub"),
		BotDebug:            envOr("BOT_DEBUG", "false") == "true",
		HealthPort:          envOr("HEALTH_PORT", "8080"),
		ReminderTickMinutes: 30,
		AudioDir:            os.Getenv("AUDIO_DIR"),
	}
	if tickRaw := os.Getenv("REMINDER_TICK_MINUTES"); tickRaw != "" {
		tick, err := strconv.Atoi(tickRaw)
		if err != nil || tick <= 0 {
			return Config{}, fmt.Errorf("invalid REMINDER_TICK_MINUTES: %q", tickRaw)
		}
		cfg.ReminderTickMinutes = tick
	}
	if cfg.TelegramBotToken == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
