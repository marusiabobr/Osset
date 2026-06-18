"""
tokenize_corpus.py - токенизация осетинского корпуса.

принцип работы:
  читает чистые JSONL-файлы (wiki_clean.jsonl, news_*.jsonl), разбивает текст
  на предложения и слова и пишет результат в формате CoNLL-U, который дальше понимает лемматизатор/
  POS-теггер (UniParser). Сейчас заполняем только два столбца: номер токена
  и сам токен (FORM). Остальные столбцы — заглушки "_", их позже заполнит
  разметчик (лемма, часть речи, морфопризнаки).

"""
import sys
import json

from razdel import sentenize, tokenize
from osnorm import normalize_ossetic

# 10 столбцов стандарта CoNLL-U
EMPTY = "\t".join(["_"] * 8)   # столбцы под леммы


def doc_to_conllu(doc, doc_index, out) -> int:
    """Один документ -> блоки CoNLL-U. Возвращает число токенов."""
    src = doc.get("source", "doc")
    text = normalize_ossetic(doc.get("text", ""))
    n_tokens = 0

    for s_idx, sent in enumerate(sentenize(text)):
        tokens = [t.text for t in tokenize(sent.text)]
        if not tokens:
            continue
        out.write(f"# sent_id = {src}-{doc_index}-{s_idx}\n")
        out.write(f"# text = {sent.text}\n")
        for i, tok in enumerate(tokens, 1):
            # ID \t FORM \t (8 заглушек)
            out.write(f"{i}\t{tok}\t{EMPTY}\n")
            n_tokens += 1
        out.write("\n")            # пустая строка = конец предложения
    return n_tokens


def run(out_path: str, in_paths: list) -> None:
    total_tokens = total_docs = 0
    with open(out_path, "w", encoding="utf-8") as out:
        for path in in_paths:
            with open(path, encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    doc = json.loads(line)
                    total_tokens += doc_to_conllu(doc, total_docs, out)
                    total_docs += 1
    print(f"Готово. Документов: {total_docs}, токенов: {total_tokens}")
    print(f"Файл: {out_path}")
    print(f"(ориентир по плану: цель ~50–100 тыс. токенов)")


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Использование: python tokenize_corpus.py <out.conllu> <in1.jsonl> [in2.jsonl ...]")
        sys.exit(1)
    run(sys.argv[1], sys.argv[2:])
