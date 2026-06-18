"""
osnorm.py — нормализация осетинского текста.

Главная задача: привести текст к единому виду до токенизации.
Самая частая проблема осетинских текстов — буква Ӕ/ӕ.

правильная осетинская буква:
    ӕ = U+04D5 (CYRILLIC SMALL LIGATURE A IE)
    Ӕ = U+04D4 (CYRILLIC CAPITAL LIGATURE A IE)

В текстах вместо неё встречаются:
    - латинская  æ/Æ  (U+00E6 / U+00C6)  — из плохой перекодировки;
    - кириллическая шва  ә/Ә  (U+04D9 / U+04D8) — её набирают по ошибке,
      т.к. она похожа и есть на казахских/татарских раскладках.
Если это не починить, "бӕлас" в трёх вариантах станет тремя разными
словами для токенизатора и лемматизатора.
"""
import re
import unicodedata

OSS_AE_LOWER = "\u04D5"   # ӕ правильная
OSS_AE_UPPER = "\u04D4"   # Ӕ правильная

# Всё, что нужно привести к правильной осетинской ӕ/Ӕ
AE_FIXES = {
    "\u00E6": OSS_AE_LOWER,  # æ латинская  -> ӕ
    "\u00C6": OSS_AE_UPPER,  # Æ латинская  -> Ӕ
    "\u04D9": OSS_AE_LOWER,  # ә шва        -> ӕ
    "\u04D8": OSS_AE_UPPER,  # Ә шва        -> Ӕ
}

_CYRILLIC = re.compile(r"[\u0400-\u04FF]")
# разрешаем кириллицу, латиницу, цифры, базовую пунктуацию
_ALLOWED = re.compile(r"[^\u0400-\u04FFA-Za-z0-9\s.,!?;:()«»\"'\-—…]")
_MULTISPACE = re.compile(r"[ \t]+")
_MULTINEWLINE = re.compile(r"\n{3,}")


def fix_ae(text: str) -> str:
    """Приводим все варианты æ/ә к правильной осетинской ӕ (U+04D5)."""
    for bad, good in AE_FIXES.items():
        text = text.replace(bad, good)
    return text


def normalize_ossetic(text: str) -> str:
    """Полная нормализация одного текста."""
    if not text:
        return ""
    text = unicodedata.normalize("NFC", text)   # единая форма Unicode
    text = fix_ae(text)                          # все ӕ -> U+04D5
    text = _ALLOWED.sub(" ", text)               # выкидываем посторонние символы
    text = _MULTISPACE.sub(" ", text)            # схлопываем пробелы
    text = _MULTINEWLINE.sub("\n\n", text)       # максимум одна пустая строка
    return text.strip()


def cyrillic_ratio(text: str) -> float:
    """Доля кириллицы среди букв — чтобы отсеять не-осетинские строки."""
    letters = [c for c in text if c.isalpha()]
    if not letters:
        return 0.0
    return sum(1 for c in letters if _CYRILLIC.match(c)) / len(letters)


def is_ossetic(text: str, min_ratio: float = 0.5, min_len: int = 20) -> bool:
    """Грубый фильтр осетинского текста."""
    return len(text) >= min_len and cyrillic_ratio(text) >= min_ratio


if __name__ == "__main__":
    import unicodedata as ud
    # три варианта ОДНОГО слова "бӕлас": правильный, латинский æ, шва ә
    variants = ["б\u04D5лас", "б\u00E6лас", "б\u04D9лас"]
    fixed = [fix_ae(w) for w in variants]
    print("после нормализации все одинаковы:", len(set(fixed)) == 1, "->", fixed[0])
    for ch in fixed[0]:
        if not ch.isascii():
            print(" ", repr(ch), "U+%04X" % ord(ch), ud.name(ch))
    dirty = "Бæлас ӕмә хур.  Lorem ipsum.\n\n\n\nИрон æвзаг!!!"
    print("норм:", repr(normalize_ossetic(dirty)))
