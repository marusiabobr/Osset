package telegram

import (
	"fmt"

	"lingw/internal/domain"
	"lingw/internal/usecase/course"

	"github.com/go-telegram/bot/models"
)

func mainKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "📚 Учиться", CallbackData: "menu:learn"}},
			{{Text: "📈 Мой прогресс", CallbackData: "menu:progress"}},
			{{Text: "⚙️ Настройки", CallbackData: "menu:settings"}},
		},
	}
}

func topicsKeyboard(topics []domain.Topic) *models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, len(topics)+1)
	for _, topic := range topics {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: topic.TitleRU, CallbackData: "topic:" + topic.Slug},
		})
	}
	rows = append(rows, []models.InlineKeyboardButton{{Text: "⬅️ Назад", CallbackData: "menu:main"}})
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func levelsKeyboard(topicSlug string, levels []course.LevelWithStatus) *models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, len(levels)+1)
	for _, level := range levels {
		label := fmt.Sprintf("%s %s", statusIcon(level.Status), level.Level.TitleRU)
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: label, CallbackData: fmt.Sprintf("level:%s:%s", topicSlug, level.Level.Slug)},
		})
	}
	rows = append(rows, []models.InlineKeyboardButton{{Text: "📚 К темам", CallbackData: "menu:learn"}})
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func statusIcon(status domain.LevelStatus) string {
	switch status {
	case domain.LevelCompleted:
		return "✅"
	case domain.LevelLocked:
		return "🔒"
	case domain.LevelInProgress:
		return "🟡"
	default:
		return "▶️"
	}
}

func exerciseNavKeyboard(topicSlug string) *models.InlineKeyboardMarkup {
	return exerciseKeyboard(topicSlug, false)
}

func exerciseKeyboard(topicSlug string, withAdvance bool) *models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, 3)
	if withAdvance {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "Далее →", CallbackData: "submit:далее"},
		})
	}
	rows = append(rows,
		[]models.InlineKeyboardButton{{Text: "📚 К уровням темы", CallbackData: "topic:" + topicSlug}},
		[]models.InlineKeyboardButton{
			{Text: "🗂 К темам", CallbackData: "menu:learn"},
			{Text: "🏠 Меню", CallbackData: "menu:main"},
		},
	)
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func levelCompleteKeyboard(topicSlug string, nextLevel domain.Level, hasNext bool) *models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, 3)
	if hasNext {
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         "▶️ Следующий уровень",
				CallbackData: fmt.Sprintf("level:%s:%s", topicSlug, nextLevel.Slug),
			},
		})
	}
	rows = append(rows,
		[]models.InlineKeyboardButton{{Text: "📚 К уровням темы", CallbackData: "topic:" + topicSlug}},
		[]models.InlineKeyboardButton{
			{Text: "🗂 К темам", CallbackData: "menu:learn"},
			{Text: "🏠 Меню", CallbackData: "menu:main"},
		},
	)
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}
