# Архитектура Lingw

Telegram-бот для изучения осетинского языка. Код организован по слоям: домен → use case → адаптеры.

## Слои

```
cmd/lingw          — сборка зависимостей, запуск бота
internal/domain    — сущности (Topic, Level, Exercise) и интерфейсы хранилищ
internal/usecase   — бизнес-логика без Telegram и SQL
internal/adapter   — реализации портов (postgres, seed, telegram, scheduler)
seeds/             — встроенный учебный контент (go:embed)
assets/audio/      — озвучка слов (go:embed + опционально AUDIO_DIR)
```

## Поток данных урока

1. Пользователь выбирает тему → `course.ListService` + `course.UnlockService` проверяют доступ.
2. Старт уровня → `level.SessionService.Start` создаёт сессию, статус `in_progress`.
3. На каждом шаге → `level.SessionService.CurrentExercise` возвращает задание с учётом порядка (`level.order`).
4. Ответ → `level.Checker` сравнивает с `accepted_literals` или лексиконом.
5. Завершение → статус `completed`, разблокировка следующего уровня/темы.

## Типы упражнений

| Тип в seeds | Назначение |
|-------------|------------|
| `theory` | Карточка правила, кнопка «Далее» |
| `vocab` | Новое слово + опционально голосовое |
| `choice` | Выбор падежа или формы |
| `translate_os_ru` | Перевод с осетинского |
| `translate_ru_os` | Сборка предложения |

## Контент

- **Источник правды:** `course_from_glossed3.json`
- **Генерация:** `python scripts/regenerate_seeds.py` → `seeds/`
- **Рантайм:** `internal/adapter/seed` читает embedded JSON

Переключение на PostgreSQL для контента предусмотрено (`CONTENT_SOURCE=postgres`), но адаптер пока заготовка.

## Прогресс пользователя

Хранится в PostgreSQL: статусы уровней, сессии, попытки. Разблокировка тем опирается на `GetLevelProgress` по каждому уровню предыдущей темы — не на JOIN с пустой таблицей `levels`.

## Порядок заданий в уровне

- Теория и словарь — в порядке файла.
- Практика по падежам (`choice`) — перемешивается детерминированно per user + level (MD5-seed), чтобы у разных учеников был разный порядок, но у одного — стабильный.

## Аудио

`assets/audio/store.go` отдаёт `.ogg` по полю `data.audio` в упражнении `vocab`. Telegram `SendVoice` требует **Opus** внутри OGG; Vorbis может не играть на мобильных клиентах.
