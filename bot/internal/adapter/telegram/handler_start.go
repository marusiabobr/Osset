package telegram

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"lingw/internal/usecase/course"
)

func (a *Adapter) handleStart(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		a.log.Warn("start ignored: empty message")
		return
	}
	a.log.Info("start command received", "user_id", update.Message.From.ID, "username", update.Message.From.Username)
	if _, err := a.register.Ensure(ctx, update.Message.From.ID, update.Message.From.Username); err != nil {
		a.log.Error("failed to register user on start", "user_id", update.Message.From.ID, "err", err)
		_, sendErr := b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "⚠️ Не удалось зарегистрировать пользователя. Попробуйте снова через пару секунд.",
		})
		if sendErr != nil {
			a.log.Error("failed to send start error", "chat_id", update.Message.Chat.ID, "err", sendErr)
		}
		return
	}
	a.sendText(
		ctx,
		update.Message.Chat.ID,
		`👋 Добро пожаловать в Lingw!

Здесь вы изучаете осетинский язык по шагам: теория -> практика -> закрепление.

Что дальше:
1) Нажмите «Учиться»
2) Выберите тему
3) Откройте первый доступный уровень

💡 Подробная инструкция: /help`,
		mainKeyboard(),
	)
}

func (a *Adapter) handleHelp(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	helpText := "📘 Как пользоваться ботом Lingw\n\n" +
		"1) Нажмите «Учиться» и выберите тему.\n" +
		"2) Открывайте уровни по порядку: новый уровень доступен после завершения предыдущего.\n" +
		"3) В каждом уровне читайте теорию и отвечайте на задания текстом.\n" +
		"4) Для теоретических карточек нажмите кнопку «Далее →».\n" +
		"5) Для переводов вводите слово/фразу максимально точно.\n\n" +
		"✅ Если ответ верный — бот сразу покажет следующий шаг.\n" +
		"🔁 Если ответ неверный — можно пробовать сколько угодно раз.\n\n" +
		"Команды:\n" +
		"/start — начать заново и открыть меню\n" +
		"/help — открыть эту инструкцию\n\n" +
		"Удачи в изучении! 🚀"
	_, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        helpText,
		ReplyMarkup: mainKeyboard(),
	})
	if err != nil {
		a.log.Error("failed to send help", "chat_id", update.Message.Chat.ID, "err", err)
	}
}

func (a *Adapter) sendMainMenu(ctx context.Context, chatID int64) {
	a.sendText(ctx, chatID, "🏠 Главное меню\nВыберите, что хотите сделать:", mainKeyboard())
}

func (a *Adapter) sendProgress(ctx context.Context, telegramUserID, chatID int64) {
	user, err := a.register.Ensure(ctx, telegramUserID, "")
	if err != nil {
		a.log.Error("failed to ensure user for progress", "user_id", telegramUserID, "err", err)
		a.sendText(ctx, chatID, "Ошибка загрузки пользователя.", mainKeyboard())
		return
	}
	summary, err := a.progress.Summary(ctx, user.ID)
	if err != nil {
		a.log.Error("failed to build progress summary", "user_id", user.ID, "err", err)
		a.sendText(ctx, chatID, "Не удалось загрузить прогресс. Попробуйте позже.", mainKeyboard())
		return
	}
	a.sendText(ctx, chatID, course.FormatProgressMessage(summary), mainKeyboard())
}

func (a *Adapter) sendSettings(ctx context.Context, chatID int64) {
	a.sendText(
		ctx,
		chatID,
		`⚙️ Настройки

• Напоминания: включены ежедневно
• Часовой пояс по умолчанию: Europe/Moscow

В следующих версиях появится гибкая настройка времени уведомлений.`,
		mainKeyboard(),
	)
}

func (a *Adapter) sendTopics(ctx context.Context, userID, chatID int64) {
	a.log.Info("open topics menu", "user_id", userID, "chat_id", chatID)
	u, err := a.register.Ensure(ctx, userID, "")
	if err != nil {
		a.log.Error("failed to ensure user in topics", "user_id", userID, "chat_id", chatID, "err", err)
		a.sendText(ctx, chatID, "Ошибка загрузки пользователя.", mainKeyboard())
		return
	}
	_ = u
	topics, err := a.list.Topics(ctx)
	if err != nil {
		a.log.Error("failed to load topics", "user_id", userID, "chat_id", chatID, "err", err)
		a.sendText(ctx, chatID, "Ошибка загрузки тем.", mainKeyboard())
		return
	}
	a.log.Info("topics loaded", "user_id", userID, "chat_id", chatID, "count", len(topics))
	a.sendText(
		ctx,
		chatID,
		`🗂 Выберите тему обучения

🔒 — тема пока закрыта
▶️ — доступна к прохождению
✅ — уже пройдена`,
		topicsKeyboard(topics),
	)
}

func (a *Adapter) handleTopic(ctx context.Context, userID, chatID int64, topicSlug string) {
	a.log.Info("topic selected", "user_id", userID, "chat_id", chatID, "topic_slug", topicSlug)
	user, err := a.register.Ensure(ctx, userID, "")
	if err != nil {
		a.log.Error("failed to ensure user in topic", "user_id", userID, "chat_id", chatID, "topic_slug", topicSlug, "err", err)
		a.sendText(ctx, chatID, "Ошибка пользователя.", mainKeyboard())
		return
	}
	if err := a.unlock.EnsureTopicAvailable(ctx, user.ID, topicSlug); err != nil {
		a.log.Warn("topic is locked or unavailable", "user_id", user.ID, "chat_id", chatID, "topic_slug", topicSlug, "err", err)
		a.sendText(
			ctx,
			chatID,
			fmt.Sprintf("🔒 Тема «%s» пока заблокирована.\nСначала завершите предыдущую тему.", topicSlug),
			mainKeyboard(),
		)
		return
	}
	levels, err := a.list.LevelsByTopic(ctx, user.ID, topicSlug)
	if err != nil {
		a.log.Error("failed to load levels for topic", "user_id", user.ID, "topic_slug", topicSlug, "err", err)
		a.sendText(ctx, chatID, "Ошибка загрузки уровней.", mainKeyboard())
		return
	}
	a.log.Info("levels loaded", "user_id", user.ID, "chat_id", chatID, "topic_slug", topicSlug, "count", len(levels))
	a.sendText(
		ctx,
		chatID,
		"📚 Уровни темы\n\nВыберите уровень. Рекомендуется идти строго по порядку.",
		levelsKeyboard(topicSlug, levels),
	)
}
