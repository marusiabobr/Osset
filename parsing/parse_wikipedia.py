"""
parse_wikipedia.py - превращает дамп осетинской Википедии в чистый текст.
 
ВХОД:  oswiki-latest-pages-articles.xml.bz2  (скачивается с dumps.wikimedia.org)
       Файл сжатый (.bz2) - скрипт сам его распаковывает.
ВЫХОД: wiki_clean.jsonl - по одной статье на строку:
       {"id":..., "title":..., "source":"wikipedia", "text":...}
 
Запуск:
    python parse_wikipedia.py oswiki-latest-pages-articles.xml.bz2 wiki_clean.jsonl
"""
import sys
import bz2
import json
import mwxml
import mwparserfromhell
 
from osnorm import normalize_ossetic, is_ossetic
 
 
def open_dump(path):
    """Открываем дамп. Если он .bz2 — распаковываем на лету через bz2.open."""
    if path.endswith(".bz2"):
        return bz2.open(path, "rb")
    return open(path, "rb")
 
 
def extract_plain(wikitext: str) -> str:
    """Wiki-разметка -> чистый текст (ссылки разворачиваются, шаблоны убираются)."""
    code = mwparserfromhell.parse(wikitext or "")
    return code.strip_code()
 
 
def parse_dump(dump_path: str, out_path: str) -> None:
    dump = mwxml.Dump.from_file(open_dump(dump_path))
    kept = skipped = 0
 
    with open(out_path, "w", encoding="utf-8") as out:
        for page in dump:
            if page.namespace != 0 or page.redirect:   # только статьи, без редиректов
                continue
            for rev in page:                            # берём последнюю ревизию
                raw = extract_plain(rev.text or "")
                text = normalize_ossetic(raw)
                if not is_ossetic(text, min_len=100):   # отсекаем заглушки/не-осетинское
                    skipped += 1
                    break
                out.write(json.dumps({
                    "id": page.id,
                    "title": page.title,
                    "source": "wikipedia",
                    "text": text,
                }, ensure_ascii=False) + "\n")
                kept += 1
                break
 
    print(f"Готово. Сохранено статей: {kept}, пропущено: {skipped}")
    print(f"Файл: {out_path}")
 
 
if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Использование: python parse_wikipedia.py <dump.xml.bz2> <out.jsonl>")
        sys.exit(1)
    parse_dump(sys.argv[1], sys.argv[2])
