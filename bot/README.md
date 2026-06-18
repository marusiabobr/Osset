# Lingw — Telegram-бот для изучения осетинского

Бот ведёт пользователя по темам и уровням: теория падежей, новые слова с озвучкой, перевод, выбор формы, сборка предложений. Прогресс и разблокировка хранятся в PostgreSQL; учебный контент — во встроенных seeds.

## Возможности

- 6 тем, 29 уровней (контент из глоссированного корпуса)
- Последовательное открытие тем и уровней
- Бесконечные попытки на задание
- Озвучка слов (`.ogg`, поле `audio` в курсе)
- Ежедневные напоминания
- Кнопки навигации: «Далее», «К уровням», «Следующий уровень»

## Стек

| Компонент | Технология |
|-----------|------------|
| Бот | Go, [go-telegram/bot](https://github.com/go-telegram/bot) |
| БД | PostgreSQL, [pgx](https://github.com/jackc/pgx) |
| Миграции | golang-migrate (Docker) |
| Контент | JSON → `seeds/` (`scripts/regenerate_seeds.py`) |

## Структура репозитория

```
cmd/lingw/                 — точка входа
internal/domain/           — модели и порты
internal/usecase/          — unlock, session, checker, order
internal/adapter/telegram/ — handlers, клавиатуры, голос
internal/adapter/seed/     — чтение embedded seeds
internal/adapter/postgres/ — пользователи и прогресс
assets/audio/              — озвучка (embed + AUDIO_DIR)
seeds/                     — topics, levels, lexicon_stub
course_from_glossed3.json    — исходник курса
scripts/regenerate_seeds.py  — генератор seeds
docs/                      — архитектура, комментарии к курсу, аудио
tests/                     — pytest для скрипта генерации
```

Подробнее: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [docs/COURSE.md](docs/COURSE.md).

## Быстрый старт

```bash
cp .env.example .env
# Укажите TELEGRAM_BOT_TOKEN в .env

docker compose up -d --build
docker compose logs -f bot
```

Миграции применяются сервисом `migrate` при первом запуске.

### Локально без Docker

```bash
# PostgreSQL должен быть доступен по DATABASE_URL
make migrate-up
make run
```

## Переменные окружения

| Переменная | Описание |
|------------|----------|
| `TELEGRAM_BOT_TOKEN` | Токен бота |
| `DATABASE_URL` | PostgreSQL connection string |
| `CONTENT_SOURCE` | `seed` (по умолчанию) или `postgres` |
| `LEXICON_SOURCE` | `stub` или `postgres` |
| `AUDIO_DIR` | Внешняя папка с `.ogg` (опционально) |
| `REMINDER_TICK_MINUTES` | Интервал проверки напоминаний |

## Обновление контента

1. Редактировать `course_from_glossed3.json` (см. комментарии в [docs/COURSE.md](docs/COURSE.md)).
2. Добавить аудио: `assets/audio/{word_id}.ogg` ([docs/AUDIO.md](docs/AUDIO.md)).
3. Сгенерировать seeds:

```bash
python scripts/regenerate_seeds.py
```

4. Пересобрать бота: `docker compose up -d --build bot`

## Тесты

```bash
# Go (юнит-тесты use case, seeds, audio)
make test

# Python (хелперы regenerate_seeds)
pip install -r requirements-dev.txt
pytest tests/ -q
```

## Типы заданий в боте

- `theory` — карточка правила
- `vocab` — новое слово (+ голос при наличии `audio`)
- `choice` — падеж или форма
- `translate_os_ru` — перевод на русский
- `translate_ru_os` — сборка предложения

## Лицензия и данные

Тексты извлечены из глоссированного корпуса; поля `needs_validation: true` помечают строки, требующие проверки носителем языка.
