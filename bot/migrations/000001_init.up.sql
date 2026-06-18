CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    timezone TEXT NOT NULL DEFAULT 'Europe/Moscow',
    reminder_hour INT NOT NULL DEFAULT 19,
    reminders_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_activity_at TIMESTAMPTZ NULL,
    last_reminder_sent_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS topics (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    title_ru TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL
);

CREATE TABLE IF NOT EXISTS levels (
    id BIGSERIAL PRIMARY KEY,
    topic_slug TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    title_ru TEXT NOT NULL,
    sort_order INT NOT NULL,
    focus TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS exercises (
    id BIGSERIAL PRIMARY KEY,
    level_slug TEXT NOT NULL,
    type TEXT NOT NULL,
    sort_order INT NOT NULL,
    data JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS lexemes (
    id BIGSERIAL PRIMARY KEY,
    ref TEXT UNIQUE NOT NULL,
    topic_id BIGINT NULL,
    pos TEXT NOT NULL,
    dialect TEXT NOT NULL DEFAULT 'iron'
);

CREATE TABLE IF NOT EXISTS word_forms (
    id BIGSERIAL PRIMARY KEY,
    lexeme_id BIGINT NOT NULL REFERENCES lexemes(id) ON DELETE CASCADE,
    grammatical_case TEXT NOT NULL,
    grammatical_number TEXT NOT NULL,
    text_os TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS translations (
    id BIGSERIAL PRIMARY KEY,
    lexeme_id BIGINT NOT NULL REFERENCES lexemes(id) ON DELETE CASCADE,
    lang TEXT NOT NULL,
    text TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_level_progress (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level_slug TEXT NOT NULL,
    status TEXT NOT NULL,
    completed_at TIMESTAMPTZ NULL,
    attempts_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, level_slug)
);

CREATE TABLE IF NOT EXISTS user_exercise_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level_slug TEXT NOT NULL,
    exercise_pos INT NOT NULL,
    answer TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS user_level_sessions (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level_slug TEXT NOT NULL,
    current_step INT NOT NULL,
    total_steps INT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, level_slug)
);
