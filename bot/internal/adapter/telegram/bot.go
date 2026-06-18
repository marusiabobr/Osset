package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"lingw/assets/audio"
	"lingw/internal/domain"
	"lingw/internal/usecase/course"
	"lingw/internal/usecase/level"
	"lingw/internal/usecase/user"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Adapter struct {
	bot      *tgbot.Bot
	log      *slog.Logger
	register *user.RegisterService
	list     *course.ListService
	unlock   *course.UnlockService
	progress *course.ProgressService
	session  *level.SessionService
	lexicon  domain.LexiconStore
	audio    *audio.Store
}

type Deps struct {
	Token    string
	Debug    bool
	Log      *slog.Logger
	Register *user.RegisterService
	List     *course.ListService
	Unlock   *course.UnlockService
	Progress *course.ProgressService
	Session  *level.SessionService
	Lexicon  domain.LexiconStore
	Audio    *audio.Store
}

func New(ctx context.Context, deps Deps) (*Adapter, error) {
	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(func(ctx context.Context, b *tgbot.Bot, update *models.Update) {
			_ = ctx
			_ = b
			_ = update
		}),
	}
	if deps.Debug {
		opts = append(opts, tgbot.WithDebug())
	}
	b, err := tgbot.New(deps.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	a := &Adapter{
		bot:      b,
		log:      deps.Log,
		register: deps.Register,
		list:     deps.List,
		unlock:   deps.Unlock,
		progress: deps.Progress,
		session:  deps.Session,
		lexicon:  deps.Lexicon,
		audio:    deps.Audio,
	}
	a.registerRoutes()
	_ = ctx
	return a, nil
}

func (a *Adapter) Start(ctx context.Context) error {
	a.log.Info("telegram bot started")
	a.bot.Start(ctx)
	return nil
}

func (a *Adapter) SendReminder(ctx context.Context, telegramID int64, text string) error {
	_, err := a.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: telegramID, Text: text})
	return err
}

func (a *Adapter) sendText(ctx context.Context, chatID int64, text string, kb *models.InlineKeyboardMarkup) {
	params := &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	_, err := a.bot.SendMessage(ctx, params)
	if err != nil {
		a.log.Error("failed to send message", "chat_id", chatID, "text", text, "err", err)
	}
}
