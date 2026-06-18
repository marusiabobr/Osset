# Аудио в Lingw

## Именование

Один файл на лексему:

```
assets/audio/{word_id}.ogg
```

Примеры: `b1_01.ogg` (аз), `t2_01.ogg` (лӕг).

## Объявление в курсе

В `course_from_glossed3.json`:

```json
{
  "id": "t2_01",
  "lemma": "лӕг",
  "trans_ru": "мужчина",
  "audio": "t2_01.ogg"
}
```

Для тем 2+ также в упражнении `learn_word`:

```json
{
  "type": "learn_word",
  "word_id": "t2_01",
  "audio": "t2_01.ogg"
}
```

Блок 1: карточка `learn_word` вставляется скриптом автоматически перед `translate_word` и берёт `audio` из `words[]`.

## Формат для Telegram

`SendVoice` ожидает **OGG Opus**, моно, 48 kHz. Если файл записан как Vorbis:

```bash
ffmpeg -i input.ogg -c:a libopus -b:a 32k -ar 48000 -ac 1 output.ogg
```

## Локальная разработка

В `.env` можно задать внешнюю папку (перекрывает embedded):

```
AUDIO_DIR=C:\path\to\audio
```

Иначе файлы вшиваются при сборке из `assets/audio/` (`go:embed`).
