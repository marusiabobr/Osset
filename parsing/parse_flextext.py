"""
parse_flextext.py — превращает глоссированные тексты FLEx (.flextext, это XML)
в структурированный JSON, готовый для генератора заданий.

Из каждого предложения достаём:
  - ossetic        — осетинское предложение целиком
  - translation_ru — свободный перевод (готовый, проверенный человеком)
  - tokens[]       — по словам: форма, лемма, глоссы и РАСПОЗНАННЫЕ признаки
                     (case/tense/person/number) — из строки глосс, а не из автотегов

Запуск:
    python parse_flextext.py texts.flextext glossed.json
"""
import sys
import os
import glob
import json
import re
import xml.etree.ElementTree as ET

# глосса -> признак 
CASE_GLOSS = {
    "NOM": "Nom", "GEN": "Gen", "DAT": "Dat",
    "ALL": "All",                       # направительный  (-мӕ)
    "ABL": "Abl",                       # отложительный   (-ӕй)
    "IN": "Ine", "INE": "Ine",          # местный внутр.  (-ы)
    "SUPER": "Sup", "SUPE": "Sup", "ON": "Sup",   # местный внешн. (-ыл)
    "COM": "Com",                       # союзный         (-имӕ)
    "EQU": "Equ", "EQ": "Equ", "SIM": "Equ",      # уподобительный (-ау)
}
TENSE_GLOSS = {"PST": "Past", "PRS": "Pres", "PRES": "Pres", "FUT": "Fut"}
NUM_GLOSS = {"SG": "Sing", "PL": "Plur"}
PERS_NUM = re.compile(r"^([123])(SG|PL)$")     # напр. 1SG, 3PL


def parse_gloss_feats(gloss, feats):
    """Разбираем строку глоссы на признаки. Глосса вида 'PST.INTR.1SG' или 'ALL'."""
    if not gloss:
        return
    for tag in re.split(r"[.\s\-]", gloss):
        tag = tag.strip()
        if tag in CASE_GLOSS:
            feats["Case"] = CASE_GLOSS[tag]
        elif tag in TENSE_GLOSS:
            feats["Tense"] = TENSE_GLOSS[tag]
        elif tag in NUM_GLOSS and "Number" not in feats:
            feats["Number"] = NUM_GLOSS[tag]
        else:
            m = PERS_NUM.match(tag)
            if m:
                feats["Person"] = m.group(1)
                feats["Number"] = NUM_GLOSS[m.group(2)]


def item(elem, type_, lang=None):
    """Текст прямого дочернего <item type=...>. Не лезет вглубь."""
    best = None
    for it in elem.findall("item"):
        if it.get("type") == type_:
            if lang and it.get("lang") == lang:
                return (it.text or "").strip()
            if best is None:
                best = (it.text or "").strip()
    return best


def parse_phrase(phrase):
    # свободный перевод: предпочитаем русский
    translation = item(phrase, "gls", lang="ru") or item(phrase, "gls")
    tokens, oss_words = [], []

    for word in phrase.iter("word"):
        form = item(word, "txt")
        if not form:                       # пунктуация без формы — пропуск
            continue
        oss_words.append(form)
        glosses, lemma, feats = [], None, {}
        for morph in word.iter("morph"):
            g = item(morph, "gls")
            cf = item(morph, "cf")         # citation form = кандидат в леммы
            if cf and not lemma:
                lemma = cf
            if g:
                glosses.append(g)
                parse_gloss_feats(g, feats)
        tokens.append({"form": form, "lemma": lemma or form,
                       "gloss": " ".join(glosses), **feats})

    ossetic = item(phrase, "txt") or " ".join(oss_words)
    return {"ossetic": ossetic, "translation_ru": translation, "tokens": tokens}


def run(in_path, out_path):
    
    if os.path.isdir(in_path):
        files = sorted(glob.glob(os.path.join(in_path, "*.flextext")))
        if not files:
            print(f"В папке {in_path} нет файлов .flextext"); return
    else:
        files = [in_path]

    sentences = []
    for fp in files:
        try:
            root = ET.parse(fp).getroot()
        except ET.ParseError as e:
            print(f"  ПРОПУСК (битый XML): {os.path.basename(fp)} — {e}")
            continue
        before = len(sentences)
        for phrase in root.iter("phrase"):
            s = parse_phrase(phrase)
            if s["ossetic"] and s["translation_ru"]:
                sentences.append(s)
        print(f"  {os.path.basename(fp)}: +{len(sentences) - before} предложений")

    json.dump(sentences, open(out_path, "w", encoding="utf-8"),
              ensure_ascii=False, indent=2)
    feat_tokens = sum(1 for s in sentences for t in s["tokens"]
                      if "Case" in t or "Tense" in t)
    print(f"\nИтого предложений с переводом: {len(sentences)}")
    print(f"Токенов с распознанным падежом/временем: {feat_tokens}")
    print(f"Файл: {out_path}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Использование: python parse_flextext.py <папка_или_файл> <out.json>")
        sys.exit(1)
    run(sys.argv[1], sys.argv[2])
