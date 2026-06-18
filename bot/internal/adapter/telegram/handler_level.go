package telegram

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"lingw/assets/audio"
	"lingw/internal/domain"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (a *Adapter) handleLevelStart(ctx context.Context, userID, chatID int64, topicSlug, levelSlug string) {
	a.log.Info("level selected", "user_id", userID, "chat_id", chatID, "topic_slug", topicSlug, "level_slug", levelSlug)
	user, err := a.register.Ensure(ctx, userID, "")
	if err != nil {
		a.log.Error("failed to ensure user in level start", "user_id", userID, "chat_id", chatID, "level_slug", levelSlug, "err", err)
		a.sendText(ctx, chatID, "Ошибка пользователя.", nil)
		return
	}
	session, err := a.session.Start(ctx, user.ID, topicSlug, levelSlug)
	if err != nil {
		a.log.Warn("failed to start level session", "user_id", user.ID, "topic_slug", topicSlug, "level_slug", levelSlug, "err", err)
		a.sendText(ctx, chatID, "🔒 Этот уровень пока заблокирован.\nСначала завершите предыдущий уровень в теме.", nil)
		return
	}
	a.log.Info("level session started", "user_id", user.ID, "level_slug", levelSlug, "current_step", session.CurrentStep, "total_steps", session.TotalSteps)
	a.sendText(
		ctx,
		chatID,
		fmt.Sprintf("🚀 Уровень начат!\nВсего заданий: %d\n\nОтвечайте текстом в этот чат.", session.TotalSteps),
		exerciseNavKeyboard(topicSlug),
	)
	a.sendCurrentExercise(ctx, user.ID, chatID, levelSlug)
}

func (a *Adapter) handleTextAnswer(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		a.log.Warn("text answer ignored: empty message")
		return
	}
	txt := strings.TrimSpace(update.Message.Text)
	if txt == "" || strings.HasPrefix(txt, "/") {
		a.log.Info("text answer ignored", "user_id", update.Message.From.ID, "reason", "empty_or_command")
		return
	}
	a.log.Info("text answer received", "user_id", update.Message.From.ID, "raw_text", txt)
	user, err := a.register.Ensure(ctx, update.Message.From.ID, update.Message.From.Username)
	if err != nil {
		a.log.Error("failed to ensure user on text answer", "user_id", update.Message.From.ID, "err", err)
		return
	}
	parts := strings.SplitN(txt, " ", 2)
	var levelSlug string
	answer := txt
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "lvl:") {
		levelSlug = strings.TrimPrefix(parts[0], "lvl:")
		answer = parts[1]
	} else {
		active, activeErr := a.session.ActiveSession(ctx, user.ID)
		if activeErr != nil {
			a.log.Info("no active level session and no lvl prefix", "user_id", user.ID)
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
				ChatID:      update.Message.Chat.ID,
				Text:        "ℹ️ Сначала выберите уровень в меню «Учиться», а затем отвечайте на задания.",
				ReplyMarkup: mainKeyboard(),
			})
			return
		}
		levelSlug = active.LevelSlug
	}
	a.processLevelAnswer(ctx, update.Message.Chat.ID, user.ID, levelSlug, answer)
}

func (a *Adapter) handleSubmitAnswer(ctx context.Context, telegramUserID, chatID int64, answer string) {
	user, err := a.register.Ensure(ctx, telegramUserID, "")
	if err != nil {
		a.log.Error("failed to ensure user on submit callback", "user_id", telegramUserID, "err", err)
		return
	}
	active, err := a.session.ActiveSession(ctx, user.ID)
	if err != nil {
		a.sendText(ctx, chatID, "ℹ️ Сначала выберите уровень в меню «Учиться».", mainKeyboard())
		return
	}
	a.processLevelAnswer(ctx, chatID, user.ID, active.LevelSlug, answer)
}

func (a *Adapter) processLevelAnswer(ctx context.Context, chatID, userID int64, levelSlug, answer string) {
	session, correct, done, err := a.session.Submit(ctx, userID, levelSlug, answer)
	if err != nil {
		a.log.Warn("failed to submit answer", "user_id", userID, "level_slug", levelSlug, "err", err)
		a.sendText(ctx, chatID, "⚠️ Ответ не принят.\nПроверьте, что уровень начат, и попробуйте снова.", nil)
		return
	}
	a.log.Info("answer submitted", "user_id", userID, "level_slug", levelSlug, "answer", answer, "correct", correct, "done", done, "current_step", session.CurrentStep, "total_steps", session.TotalSteps)
	topicSlug := a.topicSlugForLevel(ctx, levelSlug)
	if !correct {
		a.sendText(ctx, chatID, "❌ Пока неверно.\nПопробуйте ещё раз внимательно — вы справитесь 💪", exerciseNavKeyboard(topicSlug))
		return
	}
	if done {
		a.sendLevelComplete(ctx, userID, chatID, levelSlug)
		return
	}
	a.sendText(
		ctx,
		chatID,
		"✅ Верно!\nПереходим дальше: шаг "+strconv.Itoa(session.CurrentStep+1)+" из "+strconv.Itoa(session.TotalSteps),
		exerciseNavKeyboard(topicSlug),
	)
	a.sendCurrentExercise(ctx, userID, chatID, levelSlug)
}

func (a *Adapter) sendCurrentExercise(ctx context.Context, userID, chatID int64, levelSlug string) {
	session, exercise, err := a.session.CurrentExercise(ctx, userID, levelSlug)
	if err != nil {
		a.log.Warn("failed to load current exercise", "user_id", userID, "level_slug", levelSlug, "err", err)
		return
	}
	if exercise.Type == domain.ExerciseVocab {
		a.trySendExerciseVoice(ctx, chatID, exercise.Data)
	}
	prompt := a.renderExercisePrompt(ctx, exercise)
	topicSlug := a.topicSlugForLevel(ctx, levelSlug)
	advanceOnly := isAdvanceOnlyExercise(exercise)
	footer := "✍️ Введите ответ одним сообщением."
	if advanceOnly {
		footer = "Нажмите «Далее →», чтобы продолжить."
	}
	a.sendText(
		ctx,
		chatID,
		fmt.Sprintf("🧩 Задание %d/%d\n\n%s\n\n%s", session.CurrentStep+1, session.TotalSteps, prompt, footer),
		exerciseKeyboard(topicSlug, advanceOnly),
	)
}

func (a *Adapter) sendLevelComplete(ctx context.Context, userID, chatID int64, levelSlug string) {
	level, err := a.list.GetLevel(ctx, levelSlug)
	if err != nil {
		a.log.Warn("failed to resolve level on complete", "level_slug", levelSlug, "err", err)
		a.sendText(ctx, chatID, "✅ Отлично! Уровень завершён.", mainKeyboard())
		return
	}
	next, hasNext, err := a.unlock.NextLevel(ctx, userID, level.TopicSlug, levelSlug)
	if err != nil {
		a.log.Warn("failed to resolve next level", "level_slug", levelSlug, "err", err)
	}
	text := "✅ Отлично! Уровень завершён."
	if hasNext {
		text += "\n\nМожно сразу перейти к следующему уровню или вернуться в меню."
	} else {
		text += "\n\nЭто был последний уровень темы — откройте следующую тему или главное меню."
	}
	a.sendText(ctx, chatID, text, levelCompleteKeyboard(level.TopicSlug, next, hasNext))
}

func (a *Adapter) topicSlugForLevel(ctx context.Context, levelSlug string) string {
	level, err := a.list.GetLevel(ctx, levelSlug)
	if err != nil {
		return ""
	}
	return level.TopicSlug
}

func (a *Adapter) trySendExerciseVoice(ctx context.Context, chatID int64, data map[string]interface{}) {
	if a.audio == nil {
		return
	}
	ref := audio.RefFromExercise(data)
	if ref == "" {
		return
	}
	payload, err := a.audio.Load(ref)
	if err != nil {
		a.log.Info("voice file not found, skipping", "ref", ref)
		return
	}
	caption := dataStringField(data, "lemma")
	_, err = a.bot.SendVoice(ctx, &tgbot.SendVoiceParams{
		ChatID: chatID,
		Voice: &models.InputFileUpload{
			Filename: filepath.Base(ref),
			Data:     bytes.NewReader(payload),
		},
		Caption: caption,
	})
	if err != nil {
		a.log.Warn("failed to send voice", "ref", ref, "err", err)
	}
}

func dataStringField(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func isAdvanceOnlyExercise(ex domain.Exercise) bool {
	if ex.Type != domain.ExerciseTheory && ex.Type != domain.ExerciseVocab {
		return false
	}
	literals := literalsFromExerciseData(ex.Data, "accepted_literals")
	if len(literals) == 0 {
		return true
	}
	for _, lit := range literals {
		if strings.ToLower(strings.TrimSpace(lit)) != "далее" {
			return false
		}
	}
	return true
}

func (a *Adapter) renderExercisePrompt(ctx context.Context, exercise domain.Exercise) string {
	refs := refsFromData(exercise.Data, "accepted_refs")
	if p, ok := exercise.Data["prompt"].(string); ok && strings.TrimSpace(p) != "" {
		return a.withOptions(ctx, exercise.Data, p, refs)
	}
	switch exercise.Type {
	case domain.ExerciseTheory:
		if textRef, ok := exercise.Data["text_ref"].(string); ok && textRef != "" {
			if lex, err := a.lexicon.Resolve(ctx, textRef); err == nil && lex.RU != "" {
				return a.withOptions(ctx, exercise.Data, "Теория: "+lex.RU, refs)
			}
		}
		return a.withOptions(ctx, exercise.Data, "📖 Теория: ознакомьтесь с правилом и нажмите «Далее».", refs)
	case domain.ExerciseVocab:
		return a.withOptions(ctx, exercise.Data, "🌐 Перевод RU -> OS\nВведите осетинское слово для: "+a.firstRU(ctx, refs), refs)
	case domain.ExerciseChoice:
		return a.withOptions(ctx, exercise.Data, "🎯 Выберите правильный вариант и введите его текст:", refs)
	case domain.ExerciseFillBlank:
		return a.withOptions(ctx, exercise.Data, "🧱 Заполните пропуск нужной формой слова для: "+a.firstRU(ctx, refs), refs)
	case domain.ExerciseTranslateRU:
		return a.withOptions(ctx, exercise.Data, "🌐 Переведите на осетинский:\n"+a.firstRU(ctx, refs), refs)
	case domain.ExerciseTranslateOS:
		return a.withOptions(ctx, exercise.Data, "🌐 Переведите на русский:\n"+a.firstOS(ctx, refs), refs)
	case domain.ExerciseMatch:
		return a.withOptions(ctx, exercise.Data, "🔗 Сопоставьте форму и значение. Введите правильный вариант для:\n"+a.firstOS(ctx, refs), refs)
	default:
		return "Введите ответ текстом."
	}
}

func (a *Adapter) firstRU(ctx context.Context, refs []string) string {
	for _, ref := range refs {
		if lex, err := a.lexicon.Resolve(ctx, ref); err == nil && lex.RU != "" {
			return lex.RU
		}
	}
	return "заглушка"
}

func (a *Adapter) firstOS(ctx context.Context, refs []string) string {
	for _, ref := range refs {
		if lex, err := a.lexicon.Resolve(ctx, ref); err == nil && lex.OS != "" {
			return lex.OS
		}
	}
	return "заглушка"
}

func refsFromData(data map[string]interface{}, key string) []string {
	raw, ok := data[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, fmt.Sprintf("%v", v))
	}
	return out
}

func literalsFromExerciseData(data map[string]interface{}, key string) []string {
	raw, ok := data[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, fmt.Sprintf("%v", v))
	}
	return out
}

func (a *Adapter) optionLines(ctx context.Context, data map[string]interface{}, refs []string) []string {
	lits := literalsFromExerciseData(data, "options")
	if len(lits) > 0 {
		if shouldShuffleOptions(data) {
			return stableShuffleStrings(lits, optionShuffleSeed(data))
		}
		return lits
	}
	optRefs := refsFromData(data, "option_refs")
	if len(optRefs) == 0 {
		optRefs = refs
	}
	lines := make([]string, 0, len(optRefs))
	for _, r := range optRefs {
		if lex, err := a.lexicon.Resolve(ctx, r); err == nil {
			if lex.RU != "" && lex.OS != "" {
				lines = append(lines, fmt.Sprintf("%s — %s", lex.OS, lex.RU))
				continue
			}
			if lex.RU != "" {
				lines = append(lines, lex.RU)
				continue
			}
			if lex.OS != "" {
				lines = append(lines, lex.OS)
				continue
			}
		}
		lines = append(lines, r)
	}
	return lines
}

func (a *Adapter) withOptions(ctx context.Context, data map[string]interface{}, prompt string, refs []string) string {
	if lines := a.optionLines(ctx, data, refs); len(lines) > 0 {
		return prompt + "\nВарианты:\n- " + strings.Join(lines, "\n- ")
	}
	return prompt
}

func shouldShuffleOptions(data map[string]interface{}) bool {
	if _, ok := data["target_case"].(string); ok {
		return true
	}
	if matchCase, ok := data["match_case"].(bool); ok && matchCase {
		return true
	}
	if prompt, ok := data["prompt"].(string); ok {
		lower := strings.ToLower(prompt)
		return strings.Contains(lower, "падеж") || strings.Contains(lower, "парадигма") || strings.Contains(lower, "форму")
	}
	return false
}

func optionShuffleSeed(data map[string]interface{}) string {
	return strings.Join([]string{
		fmt.Sprintf("%v", data["prompt"]),
		fmt.Sprintf("%v", data["form"]),
		fmt.Sprintf("%v", data["word_id"]),
		fmt.Sprintf("%v", data["target_case"]),
	}, "|")
}

func stableShuffleStrings(items []string, seed string) []string {
	if len(items) <= 1 {
		return items
	}
	out := append([]string(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		hi := md5.Sum([]byte(seed + ":" + out[i]))
		hj := md5.Sum([]byte(seed + ":" + out[j]))
		return bytes.Compare(hi[:], hj[:]) < 0
	})
	return out
}
