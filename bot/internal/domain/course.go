package domain

type ExerciseType string

const (
	ExerciseTheory      ExerciseType = "theory"
	ExerciseVocab       ExerciseType = "vocab"
	ExerciseChoice      ExerciseType = "choice"
	ExerciseMatch       ExerciseType = "match"
	ExerciseFillBlank   ExerciseType = "fill_blank"
	ExerciseTranslateRU ExerciseType = "translate_ru_os"
	ExerciseTranslateOS ExerciseType = "translate_os_ru"
)

type Topic struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	TitleRU     string `json:"title_ru"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type Level struct {
	ID        int64  `json:"id"`
	TopicSlug string `json:"topic_slug"`
	Slug      string `json:"slug"`
	TitleRU   string `json:"title_ru"`
	SortOrder int    `json:"sort_order"`
}

type Exercise struct {
	ID        int64                  `json:"id"`
	LevelSlug string                 `json:"level_slug"`
	Type      ExerciseType           `json:"type"`
	SortOrder int                    `json:"sort_order"`
	Data      map[string]interface{} `json:"data"`
}
