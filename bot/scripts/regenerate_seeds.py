import hashlib
import json
import math
import re
from pathlib import Path

TOPIC_SLUGS = {
    1: "cases",
    2: "family",
    3: "home",
    4: "time",
    5: "nature",
    6: "culture",
}


def split_answers(raw: str) -> list[str]:
    if not raw:
        return []
    values = {raw.strip()}
    for part in re.split(r"[;/]", raw):
        part = part.strip()
        if part:
            values.add(part)
    comma_parts = [p.strip() for p in raw.split(",") if p.strip()]
    if 1 < len(comma_parts) <= 3:
        values.update(comma_parts)
    return sorted(values)


def stable_shuffle(items: list[str], key: str) -> list[str]:
    ranked = sorted(items, key=lambda x: hashlib.md5(f"{key}:{x}".encode()).hexdigest())
    return ranked


def pick_case_options(correct: str, pool: list[str], key: str) -> list[str]:
    distractors = [c for c in pool if c != correct]
    distractors = stable_shuffle(distractors, key)
    options = [correct] + distractors[:3]
    options = stable_shuffle(options, key + ":opts")
    return options


def join_sentence(tokens: list[str]) -> str:
    return " ".join(tokens)


def sentence_variants(tokens: list[str]) -> list[str]:
    base = join_sentence(tokens)
    variants = {base}
    if tokens:
        variants.add(join_sentence([tokens[0].lower(), *tokens[1:]]))
    return sorted(variants)


def make_learn_word_seed(word_id: str, word_meta: dict) -> dict:
    lemma = word_meta.get("lemma", "")
    trans_ru = word_meta.get("trans_ru", "")
    audio = word_meta.get("audio")
    prompt = f"📚 Новое слово\n«{lemma}» — {trans_ru}"
    if audio is None:
        prompt += "\n🔊 (аудио скоро)"
    prompt += "\nНажмите «Далее», когда запомните."
    return {
        "type": "vocab",
        "data": {
            "prompt": prompt,
            "lemma": lemma,
            "trans_ru": trans_ru,
            "word_id": word_id,
            "audio": audio,
            "accepted_literals": ["далее"],
        },
    }


def to_seed_exercises(ex: dict, words_by_id: dict[str, dict], block_id: int) -> list[dict]:
    ex_type = ex.get("type")
    word_meta = words_by_id.get(ex.get("word_id", ""), {})

    if ex_type == "rule_card":
        case = ex.get("case", "")
        ending = ex.get("ending", "")
        function = ex.get("function", "правило")
        example = ex.get("example", "")
        example_form = ex.get("example_form", "")
        prompt = f"📖 Падеж {case} ({ending}): {function}."
        if example_form and example:
            prompt += f"\nПример: «{example_form}» в «{example}»."
        prompt += "\nНажмите «Далее», когда будете готовы."
        return [
            {
                "type": "theory",
                "data": {
                    "prompt": prompt,
                    "accepted_literals": ["далее"],
                    "case": case,
                    "ending": ending,
                },
            }
        ]

    if ex_type == "learn_word":
        lemma = ex.get("lemma", "")
        trans_ru = ex.get("trans_ru", "")
        audio = ex.get("audio")
        prompt = f"📚 Новое слово\n«{lemma}» — {trans_ru}"
        if audio is None:
            prompt += "\n🔊 (аудио скоро)"
        prompt += "\nНажмите «Далее», когда запомните."
        return [
            {
                "type": "vocab",
                "data": {
                    "prompt": prompt,
                    "lemma": lemma,
                    "trans_ru": trans_ru,
                    "word_id": ex.get("word_id", ""),
                    "audio": audio,
                    "accepted_literals": ["далее"],
                },
            }
        ]

    if ex_type == "translate_word":
        word = ex.get("word", "")
        answer = ex.get("answer", "")
        literals = split_answers(answer)
        prompt = f"Переведите на русский: «{word}»."
        if word_meta.get("pos") == "VERB":
            prompt = f"Переведите глагол на русский: «{word}»."
        data = {
            "prompt": prompt,
            "accepted_literals": literals,
            "word_id": ex.get("word_id", ""),
            "word": word,
        }
        pool = word_meta.get("distractor_pool") or []
        if pool:
            data["distractor_pool"] = pool
        out = [{"type": "translate_os_ru", "data": data}]
        word_id = ex.get("word_id", "")
        if block_id == 1 and word_id and word_meta:
            return [make_learn_word_seed(word_id, word_meta), *out]
        return out

    if ex_type == "choose_case":
        form = ex.get("form") or ex.get("word", "")
        answer = ex.get("answer", "")
        options = ex.get("options") or []
        answer_ru = ex.get("answer_ru", "")
        lemma = ex.get("lemma", "")
        prompt = f"Какой падеж у формы «{form}»?"
        if lemma:
            prompt = f"Слово «{lemma}». {prompt}"
        if answer_ru:
            prompt += f"\nПодсказка: {answer_ru}."
        prompt += "\nВведите падеж."
        return [
            {
                "type": "choice",
                "data": {
                    "prompt": prompt,
                    "accepted_literals": [answer],
                    "options": options,
                    "word_id": ex.get("word_id", ""),
                    "form": form,
                },
            }
        ]

    if ex_type == "choose_form":
        lemma = ex.get("lemma", "")
        prompt_form = ex.get("prompt_form", lemma)
        target_case = ex.get("target_case", "")
        target_case_ru = ex.get("target_case_ru", "")
        answer = ex.get("answer", "")
        options = ex.get("options") or []
        prompt = (
            f"Слово «{lemma}» (словарная форма: «{prompt_form}»).\n"
            f"Нужен падеж {target_case}"
        )
        if target_case_ru:
            prompt += f" ({target_case_ru})"
        prompt += ".\nВыберите правильную форму и введите её."
        return [
            {
                "type": "choice",
                "data": {
                    "prompt": prompt,
                    "accepted_literals": [answer],
                    "options": options,
                    "word_id": ex.get("word_id", ""),
                    "target_case": target_case,
                },
            }
        ]

    if ex_type == "match_case":
        lemma = ex.get("lemma", "")
        pairs = ex.get("pairs") or []
        cases = sorted({p["case"] for p in pairs})
        out = []
        for idx, pair in enumerate(pairs):
            form = pair["form"]
            correct = pair["case"]
            options = pick_case_options(correct, cases, f"{lemma}:{form}:{idx}")
            prompt = (
                f"Парадигма «{lemma}»: укажите падеж формы «{form}».\n"
                "Введите падеж."
            )
            out.append(
                {
                    "type": "choice",
                    "data": {
                        "prompt": prompt,
                        "accepted_literals": [correct],
                        "options": options,
                        "word_id": ex.get("word_id", ""),
                        "form": form,
                        "match_case": True,
                        "match_index": idx,
                        "match_total": len(pairs),
                    },
                }
            )
        return out

    if ex_type == "assemble_sentence":
        ru = ex.get("ru", "")
        answer = ex.get("answer") or []
        scrambled = ex.get("scrambled") or []
        scrambled_line = " · ".join(scrambled)
        literals = sentence_variants(answer)
        prompt = (
            f"Соберите предложение на осетинском.\n"
            f"Перевод: «{ru}»\n"
            f"Слова (в произвольном порядке): {scrambled_line}\n"
            "Введите предложение через пробел."
        )
        data = {
            "prompt": prompt,
            "accepted_literals": literals,
            "ru": ru,
            "answer": answer,
            "scrambled": scrambled,
            "contains_lemmas": ex.get("contains_lemmas") or [],
            "audio": ex.get("audio"),
            "sentence_id": ex.get("id", ""),
        }
        if ex.get("tokens"):
            data["tokens"] = ex["tokens"]
        return [{"type": "translate_ru_os", "data": data}]

    if ex_type == "mixed_review":
        note = ex.get("note", "Повторение пройденных слов.")
        return [
            {
                "type": "theory",
                "data": {
                    "prompt": f"Итоговое повторение: {note} Нажмите «Далее», чтобы продолжить.",
                    "accepted_literals": ["далее"],
                },
            }
        ]

    return [
        {
            "type": "theory",
            "data": {
                "prompt": f"Справка: неподдержанный тип задания «{ex_type}». Нажмите «Далее».",
                "accepted_literals": ["далее"],
            },
        }
    ]


def collect_words(block: dict) -> list[dict]:
    items = list(block.get("words") or [])
    items.extend(block.get("verbs") or [])
    return items


def add_word_to_lexicon(lexicon: dict, word: dict) -> None:
    wid = word.get("id", "")
    if not wid:
        return
    lexicon[f"{wid}_lemma"] = {"os": word.get("lemma", ""), "ru": word.get("trans_ru", "")}
    for p in word.get("paradigm") or []:
        case = p.get("case", "Case")
        lexicon[f"{wid}_{case}"] = {
            "os": p.get("form", ""),
            "ru": f"{word.get('trans_ru', '')} ({case})",
        }
    for f in word.get("forms") or []:
        tense = f.get("tense", "Form")
        lexicon[f"{wid}_{tense}"] = {
            "os": f.get("form", ""),
            "ru": f"{word.get('trans_ru', '')} ({tense})",
        }
    for d in word.get("distractor_pool") or []:
        d_lemma = d.get("lemma", "")
        if d_lemma:
            lexicon.setdefault(
                f"pool_{d_lemma}",
                {"os": d_lemma, "ru": d.get("trans_ru", "")},
            )


def chunk_exercises(items: list[dict], target_levels: int = 4) -> list[list[dict]]:
    if not items:
        return [
            [
                {
                    "type": "theory",
                    "data": {
                        "prompt": "Блок пока пуст. Нажмите «Далее».",
                        "accepted_literals": ["далее"],
                    },
                }
            ]
        ]
    levels_count = max(1, min(5, target_levels))
    if len(items) > 50:
        levels_count = 5
    chunk = max(1, math.ceil(len(items) / levels_count))
    result = [items[i : i + chunk] for i in range(0, len(items), chunk)]
    if len(result) > 5:
        merged = result[:4]
        tail = []
        for part in result[4:]:
            tail.extend(part)
        merged.append(tail)
        result = merged
    return result


def main() -> None:
    root = Path(__file__).resolve().parents[1]
    src = json.loads((root / "course_from_glossed3.json").read_text(encoding="utf-8"))
    blocks = src.get("blocks", [])

    topics = []
    levels = {}
    lexicon = {"cmd_next": {"os": "далее", "ru": "далее"}}

    for block in blocks:
        block_id = block["id"]
        slug_suffix = TOPIC_SLUGS.get(block_id, f"block{block_id}")
        topic_slug = f"topic_{block_id:02d}_{slug_suffix}"
        title_os = block.get("title_os") or ""
        topics.append(
            {
                "id": block_id,
                "slug": topic_slug,
                "title_ru": block.get("title", f"Блок {block_id}"),
                "description": title_os,
                "sort_order": block_id,
            }
        )

        words = collect_words(block)
        words_by_id = {w["id"]: w for w in words if w.get("id")}
        for w in words:
            add_word_to_lexicon(lexicon, w)

        seed_exercises: list[dict] = []
        for ex in block.get("exercises", []):
            seed_exercises.extend(to_seed_exercises(ex, words_by_id, block_id))

        chunks = chunk_exercises(seed_exercises, target_levels=4)
        for idx, chunk in enumerate(chunks, 1):
            level_slug = f"topic_{block_id:02d}_level_{idx:02d}"
            prepared = []
            for ex_idx, ex in enumerate(chunk, 1):
                item = dict(ex)
                item["sort_order"] = ex_idx
                prepared.append(item)
            levels[level_slug] = {
                "slug": level_slug,
                "topic_slug": topic_slug,
                "title_ru": f"{block.get('title', 'Блок')} — уровень {idx}",
                "sort_order": idx,
                "exercises": prepared,
            }

    levels_dir = root / "seeds" / "levels"
    for p in levels_dir.glob("*.json"):
        p.unlink()
    for slug, payload in sorted(levels.items()):
        (levels_dir / f"{slug}.json").write_text(
            json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )

    (root / "seeds" / "topics.json").write_text(
        json.dumps(sorted(topics, key=lambda t: t["sort_order"]), ensure_ascii=False, indent=2)
        + "\n",
        encoding="utf-8",
    )
    (root / "seeds" / "lexicon_stub.json").write_text(
        json.dumps(lexicon, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    print(
        f"Wrote {len(topics)} topics, {len(levels)} levels and {len(lexicon)} lexicon refs "
        f"from course_from_glossed3.json"
    )


if __name__ == "__main__":
    main()
