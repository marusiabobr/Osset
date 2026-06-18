"""
Превращает glossed.json (ручные глоссы устного корпуса) в json-файл с заданиями для бота
"""
import os
import argparse
import json
import random
import re
from collections import Counter, defaultdict

_DIR = os.path.dirname(os.path.abspath(__file__))

_parser = argparse.ArgumentParser(
    description="glossed.json + вводный блок -> JSON-курс для бота",
)
_parser.add_argument("--glossed", default=os.path.join(_DIR, "..", "parsing", "glossed.json"), help="глоссированный корпус")
_parser.add_argument("--block1", default=os.path.join(_DIR, "block1_words_cases.json"), help="вводный блок")
_parser.add_argument("--output", default="course_from_glossed.json", help="куда сохранить курс")
_args, _ = _parser.parse_known_args()

SRC = _args.glossed
BLOCK1 = _args.block1
OUT = _args.output

# стандартные осетинские падежи: окончание + на какие вопросы отвечает
CASE_ENDING = {
    "Nom": "—", "Gen": "-ы", "Dat": "-ӕн", "All": "-мӕ", "Abl": "-ӕй",
    "Ine": "-ы", "Sup": "-ыл", "Com": "-имӕ", "Equ": "-ау",
}
CASE_FUNC = {
    "Nom": "словарная форма, подлежащее: «кто? что?»",
    "Gen": "принадлежность: «чей? чего?»",
    "Dat": "адресат: «кому?»",
    "All": "направление: «к, в (куда)»",
    "Abl": "источник/средство: «от, из, чем?»",
    "Ine": "местонахождение: «в, внутри (где?)»",
    "Sup": "на поверхности: «на (ком/чём)»",
    "Com": "совместность: «с (кем)»",
    "Equ": "уподобление: «как, подобно»",
}
CASE_FUNC_SHORT = {
    "Nom": "словарная форма, подлежащее", "Gen": "принадлежность",
    "Dat": "адресат «кому?»", "All": "направление «куда?»",
    "Abl": "источник «откуда/чем?»", "Ine": "местонахождение «где?»",
    "Sup": "на поверхности «на чём?»", "Com": "совместность «с кем?»",
    "Equ": "уподобление «как?»",
}
CASE_ORDER = ["Nom", "Gen", "Dat", "All", "Abl", "Ine", "Sup", "Equ", "Com"]

# у нас в разных файлах æ/Æ записана по-разному
# приводим к одному виду
def norm(t):
    if not t:
        return t
    return t.replace("\u00e6", "\u04d5").replace("\u00c6", "\u04d4")


# кириллица + ӕ/Ӕ, без дефиса
OSSETIC = re.compile("^[а-яёА-ЯЁ\u04d4\u04d5]+$")
TAG = re.compile(r'^[A-Z][A-Z0-9.]*$')

def ru_part(gloss):
    """Русская часть глоссы: выкидываем латинские теги (PRS.3SG, NOM, NEG...) и числа"""
    return ' '.join(
        w for w in gloss.split()
        if not TAG.match(w) and not re.search(r'[A-Za-z0-9.=+]', w)
    ).strip()


def clean_lemma(lemma):
    """Проверить, что лемма пригодна для словаря"""
    return (bool(lemma) and bool(OSSETIC.match(lemma)) and len(lemma) >= 2 and not lemma.isupper())


# загрузка корпуса
with open(SRC, encoding="utf-8") as f:
    corpus = json.load(f)

# нормализуем латинскую æ, делаем кириллическую во всем корпусе
for s in corpus:
    s["ossetic"] = norm(s["ossetic"])
    s["translation_ru"] = norm(s["translation_ru"])
    for t in s["tokens"]:
        t["form"] = norm(t.get("form", ""))
        t["lemma"] = norm(t.get("lemma", ""))
        t["gloss"] = norm(t.get("gloss", ""))

# noun_forms[lemma][case] = Counter((form, is_plural))
# насчитываем частотности
noun_forms = defaultdict(lambda: defaultdict(Counter))
noun_ru = defaultdict(Counter)
verb_ru = defaultdict(Counter)
verb_forms = defaultdict(lambda: defaultdict(Counter))
verb_freq = Counter()

for s in corpus:
    for t in s["tokens"]:
        lem = t.get("lemma", "")
        if not clean_lemma(lem):
            continue
        rp = ru_part(t.get("gloss", ""))
        if not rp:
            continue
        if "Case" in t:
            plural = (t.get("Number") == "Plur")
            noun_forms[lem][t["Case"]][(t["form"], plural)] += 1
            noun_ru[lem][rp] += 1
        if "Tense" in t:
            verb_freq[lem] += 1
            verb_ru[lem][rp] += 1
            verb_forms[lem][t["Tense"]][t["form"]] += 1


def best_singular_form(lemma, case):
    """Предпочитаем форму ед. числа, самую частотную"""
    c = noun_forms[lemma][case]
    sing = [(f, n) for (f, pl), n in c.items() if not pl]
    pool = sing if sing else [(f, n) for (f, pl), n in c.items()]
    if not pool:
        return None
    pool.sort(key=lambda x: -x[1])
    return pool[0][0]


def build_paradigm(lemma):
    para = []
    for case in CASE_ORDER:
        if case == "Nom":
            form = lemma
        else:
            if case not in noun_forms[lemma]:
                continue
            form = best_singular_form(lemma, case)
            if not form:
                continue
        para.append({
            "case": case,
            "form": form,
            "ending": CASE_ENDING[case],
            "function": CASE_FUNC[case],
        })
    return para


# берем несколько тем - несколько разделом для дуолинго
THEMES = [
    {
        "id": 2, "title": "Семья и люди", "title_os": None,
        "nouns": [
            ("лӕг", "мужчина; человек"), ("сывӕллон", "ребёнок"),
            ("лӕппу", "мальчик"), ("чызг", "девочка, дочь"),
            ("мад", "мать"), ("фыд", "отец"),
        ],
        "verbs": [],
    },
    {
        "id": 3, "title": "Дом и быт", "title_os": None,
        "nouns": [
            ("хӕдзар", "дом"), ("дуар", "дверь"), ("фынг", "стол (накрытый)"),
            ("кӕрт", "двор"), ("чиныг", "книга"), ("тӕбӕгъ", "тарелка"),
        ],
        "verbs": [],
    },
    {
        "id": 4, "title": "Время и день", "title_os": None,
        "nouns": [
            ("бон", "день"), ("къуыри", "неделя"), ("сахат", "час"),
            ("райсом", "утро"), ("ӕхсӕв", "ночь"), ("заман", "время, эпоха"),
        ],
        "verbs": [("зон", "знать"), ("дзур", "говорить"), ("уарз", "любить")],
    },
    {
        "id": 5, "title": "Село и природа", "title_os": None,
        "nouns": [
            ("хъӕу", "село"), ("хох", "гора"), ("дон", "вода; река"),
            ("хъӕд", "лес"), ("бӕлас", "дерево"), ("фӕндаг", "дорога, путь"),
        ],
        "verbs": [("цӕр", "жить"), ("кус", "работать"), ("кӕс", "смотреть")],
        "sentences": True,
    },
    {
        "id": 6, "title": "Язык и обычай", "title_os": None,
        "nouns": [
            ("ныхас", "слово"), ("ӕвзаг", "язык"), ("ӕгъдау", "обычай"),
            ("куывд", "пир, праздник"), ("хабар", "новость, известие"), ("ном", "имя"),
        ],
        "verbs": [("хон", "называть, звать"), ("хъус", "слушать"), ("уыд", "быть")],
        "sentences": True,
    },
]


def rnd(seed):
    r = random.Random()
    r.seed(seed)
    return r

# глобальный пул всех изучаемых слов (для distractor_pool)
# distractor_pool, чтобы путать пользователя при выборе корректного перевода
ALL_TAUGHT = []
for th in THEMES:
    for lem, tr in th["nouns"]:
        ALL_TAUGHT.append({"lemma": lem, "trans_ru": tr})


def distractor_pool(self_lemma, seed):
    pool = [w for w in ALL_TAUGHT if w["lemma"] != self_lemma]
    rnd(seed).shuffle(pool)
    return pool[:12]


def case_options(correct, seed):
    others = [c for c in CASE_ENDING if c != correct]
    rnd(seed).shuffle(others)
    opts = [correct] + others[:3]
    rnd(seed + 1).shuffle(opts)
    return opts


# сбор предложений для assemble_sentence
LEAD_FILLER = {"йер", "мӕнӕ", "гъе", "гъей", "о", "ӕй"}

def sentence_score(s, taught_lemmas):
    toks = s["tokens"]
    forms = [t["form"] for t in toks]
    lemmas = [t.get("lemma", "") for t in toks]
    score = 0
    score += sum(1 for lm in lemmas if lm in taught_lemmas) * 3 # содержит выученные слова
    score -= max(0, len(toks) - 6) # короче - лучше
    return score


def collect_sentences(taught_lemmas, need, used_texts, seed=7):
    cand = []
    for s in corpus:
        toks = s["tokens"]
        n = len(toks)
        if not (4 <= n <= 8):
            continue
        tr = s["translation_ru"].strip()
        if not tr or len(tr) > 70:
            continue
        if re.search(r'\(\*|\.\.\.|\?\?\?|:', tr):
            continue
        if "ӕ" not in s["ossetic"] and "Ӕ" not in s["ossetic"]:
            continue
        bad = False
        for t in toks:
            g = t.get("gloss", "")
            lem = t.get("lemma", "")
            if not g.strip():
                bad = True
                break
            if "=" in lem:
                bad = True
                break
            if "HES" in g:
                bad = True
                break
            if t["form"] == lem and lem[:1].isupper(): # имена собственные
                bad = True
                break
        if bad:
            continue
        if toks[0]["form"].lower() in LEAD_FILLER:
            continue
        if s["ossetic"] in used_texts:
            continue
        cand.append(s)
    cand.sort(key=lambda s: -sentence_score(s, taught_lemmas))
    chosen = cand[:need]
    out = []
    for i, s in enumerate(chosen):
        used_texts.add(s["ossetic"])
        answer = [t["form"] for t in s["tokens"]]
        scrambled = answer[:]
        rnd(seed + i).shuffle(scrambled)
        if scrambled == answer and len(answer) > 1:
            scrambled = answer[::-1]
        out.append({
            "answer": answer,
            "scrambled": scrambled,
            "ru": s["translation_ru"].strip(),
            "tokens": [
                {"form": t["form"], "lemma": t.get("lemma", ""), "gloss": t.get("gloss", "")}
                for t in s["tokens"]
            ],
            "contains_lemmas": sorted({t.get("lemma", "") for t in s["tokens"]} & taught_lemmas),
        })
    return out


# сборка блоков
taught_so_far = set()
out_blocks = []
USED_TEXTS = set()

# вводный блок — копируем block1 как есть (собирали раньше, много в ручную)
with open(BLOCK1, encoding="utf-8") as f:
    b1 = json.load(f)
intro = b1["block"]
out_blocks.append(intro)
for w in intro.get("words", []):
    taught_so_far.add(w["lemma"])

for th in THEMES:
    bid = th["id"]
    words = []
    verbs = []
    exercises = []

    # существительные: слово + парадигма
    wi = 0
    for lemma, tr in th["nouns"]:
        para = build_paradigm(lemma)
        if len(para) < 3:
            # на всякий случай - пропускаем слишком бедные на падежи
            print(f"  [skip noun {lemma}: только {len(para)} падежей]")
            continue
        wi += 1
        wid = f"t{bid}_{wi:02d}"
        words.append({
            "id": wid,
            "lemma": lemma,
            "pos": "NOUN",
            "trans_ru": tr,
            "audio": None,
            "needs_validation": True,
            "paradigm": para,
            "distractor_pool": distractor_pool(lemma, hash(wid)),
        })

    # глаголы (начинаем давать с темы 3): только лексика + формы по временам
    vi = 0
    for lemma, tr in th.get("verbs", []):
        vi += 1
        vid = f"t{bid}_v{vi:02d}"
        forms = []
        for tense in ["Pres", "Past", "Fut"]:
            if tense in verb_forms.get(lemma, {}):
                form = verb_forms[lemma][tense].most_common(1)[0][0]
                forms.append({"tense": tense, "form": form})
        verbs.append({
            "id": vid,
            "lemma": lemma,
            "pos": "VERB",
            "trans_ru": tr,
            "audio": None,
            "needs_validation": True,
            "forms": forms,
        })

    # Упраждения
    # 1) карточки слов (показ + перевод + слот аудио) - существительные, затем глаголы
    for w in words:
        exercises.append({
            "type": "learn_word", "word_id": w["id"],
            "lemma": w["lemma"], "trans_ru": w["trans_ru"], "audio": None,
        })
    for v in verbs:
        exercises.append({
            "type": "learn_word", "word_id": v["id"],
            "lemma": v["lemma"], "trans_ru": v["trans_ru"], "audio": None,
        })

    # 2) translate_word
    for w in words:
        exercises.append({
            "type": "translate_word", "word_id": w["id"],
            "word": w["lemma"], "answer": w["trans_ru"],
        })
    for v in verbs:
        exercises.append({
            "type": "translate_word", "word_id": v["id"],
            "word": v["lemma"], "answer": v["trans_ru"],
        })

    # 3) + 4) падежи: для каждого слова - choose_case по косвенным падежам, затем choose_form
    for w in words:
        para = {p["case"]: p for p in w["paradigm"]}
        oblique = [c for c in CASE_ORDER if c in para and c != "Nom"]
        # choose_case: дана форма -> назвать падеж
        for c in oblique:
            seed = hash(w["id"] + c)
            exercises.append({
                "type": "choose_case",
                "word_id": w["id"],
                "lemma": w["lemma"],
                "form": para[c]["form"],
                "answer": c,
                "answer_ru": CASE_FUNC_SHORT[c],
                "options": case_options(c, seed),
            })
        # choose_form: дана словарная форма + нужный падеж -> выбрать форму из вариантов
        all_forms = [p["form"] for p in w["paradigm"]]
        for c in oblique:
            target = para[c]["form"]
            wrong = [f for f in all_forms if f != target]
            rnd(hash(w["id"] + c + "f")).shuffle(wrong)
            opts = [target] + wrong[:3]
            rnd(hash(w["id"] + c + "o")).shuffle(opts)
            if len(opts) < 2:
                continue
            exercises.append({
                "type": "choose_form",
                "word_id": w["id"],
                "lemma": w["lemma"],
                "prompt_form": w["lemma"],
                "target_case": c,
                "target_case_ru": CASE_FUNC_SHORT[c],
                "answer": target,
                "options": opts,
            })

        # 5) match_case: вся парадигма
        exercises.append({
            "type": "match_case",
            "word_id": w["id"],
            "lemma": w["lemma"],
            "pairs": [{"form": p["form"], "case": p["case"]} for p in w["paradigm"]],
        })

    # 6) assemble_sentence
    taught_lemmas_now = taught_so_far | {w["lemma"] for w in words} | {v["lemma"] for v in verbs}
    if th.get("sentences"):
        sents = collect_sentences(taught_lemmas_now, need=8, used_texts=USED_TEXTS)
        for i, s in enumerate(sents, 1):
            exercises.append({
                "type": "assemble_sentence",
                "id": f"t{bid}_s{i:02d}",
                "ru": s["ru"],
                "answer": s["answer"],
                "scrambled": s["scrambled"],
                "audio": None,
                "tokens": s["tokens"],
                "contains_lemmas": s["contains_lemmas"],
                "needs_validation": True,
            })

    block = {
        "id": bid,
        "title": th["title"],
        "title_os": th["title_os"],
        "title_needs_validation": True,
        "words": words,
    }
    if verbs:
        block["verbs"] = verbs
    block["exercises"] = exercises
    out_blocks.append(block)

    for w in words:
        taught_so_far.add(w["lemma"])
    for v in verbs:
        taught_so_far.add(v["lemma"])


meta = {
    "course": "Осетинский в формате Duolingo — курс из глоссированного корпуса",
    "source": "Устный корпус осетинского (ossetic-studies.org), ручные "
              "поморфемные глоссы + перевод ru; вводный блок",
    "structure": "блок 1 — вводный (падежи на слове «аз»); далее тематические блоки: "
                 "слова → падежи → (с темы 4) сборка предложений",
    "case_inventory": [
      "Именительный",
      "Родительный",
      "Дательный",
      "Направительный",
      "Отложительный",
      "Ine",
      "Внешне-местный",
      "Уподобительный"
    ],
    "exercise_types": {
        "rule_card": "карточка-правило: падеж + окончание + функция + "
                     "пример (только во вводном блоке)",
        "learn_word": "карточка: слово + перевод + аудио",
        "translate_word": "осетинское слово -> выбрать перевод; 3 дистрактора бот берет в рантайме",
        "choose_case": "дана форма в падеже -> назвать падеж (4 варианта)",
        "choose_form": "дана словарная форма + нужный падеж -> выбрать "
                       "правильную форму (дистракторы — другие падежи того же слова)",
        "match_case": "сопоставить формы и падежи (вся парадигма)",
        "assemble_sentence": "дан перевод -> собрать осетbycrjt предложение из перемешанных слов",
    },
    "distractor_note": "options для translate_word не фиксируются: бот берет 3 дистрактора из "
                       "distractor_pool среди уже показанных пользователю слов (seen_lemmas).",
    "audio_note": "поля audio:null - слоты под запись носителя.",
    "verbs_note": "Глаголы (с темы 3) вводятся только как лексика + формы по временам,"
                  "для разбавки и для сборки предложений."
                  "choose_tense можно добавить позже на поле forms.",
}

course = {"meta": meta, "blocks": out_blocks}
os.makedirs("/mnt/user-data/outputs", exist_ok=True)
with open(OUT, "w", encoding="utf-8") as f:
    json.dump(course, f, ensure_ascii=False, indent=2)
