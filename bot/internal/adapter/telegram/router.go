package telegram

import (
	"context"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (a *Adapter) registerRoutes() {
	a.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypePrefix, a.handleStart)
	a.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/help", tgbot.MatchTypePrefix, a.handleHelp)
	a.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypePrefix, a.handleTextAnswer)
	a.bot.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "", tgbot.MatchTypePrefix, a.handleCallback)
}

func (a *Adapter) handleCallback(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		a.log.Warn("callback update is empty")
		return
	}
	data := update.CallbackQuery.Data
	userID := update.CallbackQuery.From.ID
	chatID := userID
	if update.CallbackQuery.Message.Message != nil {
		chatID = update.CallbackQuery.Message.Message.Chat.ID
	}
	a.log.Info("callback received", "user_id", userID, "chat_id", chatID, "data", data)

	_, ackErr := b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	if ackErr != nil {
		a.log.Warn("failed to ack callback", "user_id", userID, "err", ackErr)
	}

	switch {
	case data == "menu:main":
		a.sendMainMenu(ctx, chatID)
	case data == "menu:learn":
		a.sendTopics(ctx, userID, chatID)
	case data == "menu:progress":
		a.sendProgress(ctx, userID, chatID)
	case data == "menu:settings":
		a.sendSettings(ctx, chatID)
	case strings.HasPrefix(data, "topic:"):
		a.handleTopic(ctx, userID, chatID, strings.TrimPrefix(data, "topic:"))
	case strings.HasPrefix(data, "submit:"):
		a.handleSubmitAnswer(ctx, userID, chatID, strings.TrimPrefix(data, "submit:"))
	case strings.HasPrefix(data, "level:"):
		parts := strings.SplitN(strings.TrimPrefix(data, "level:"), ":", 2)
		if len(parts) != 2 {
			a.log.Warn("invalid level callback payload", "user_id", userID, "data", data)
			return
		}
		a.handleLevelStart(ctx, userID, chatID, parts[0], parts[1])
	default:
		a.log.Warn("unknown callback command", "user_id", userID, "data", data)
		_, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Команда не распознана.",
		})
		if err != nil {
			a.log.Error("failed to send unknown command message", "chat_id", chatID, "err", err)
		}
	}
}
