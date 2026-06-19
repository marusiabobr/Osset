import re
import time
import argparse
from pathlib import Path
from uniparser_morph import Analyzer


DEFAULT_INPUT = "corpus.conllu"
DEFAULT_OUTPUT = "corpus_annotated.conllu"
DEFAULT_GRAMMAR = "uniparser-grammar-ossetic"
DEFAULT_WORDLIST = "uniparser-grammar-ossetic/wordlists/wordlist_analyzed.txt"

CYRILLIC_RE = re.compile(r"[а-яёА-ЯЁ\u04d0-\u04ff]")
NUM_RE = re.compile(r"^\d+([.,\-]\d+)?$")
LATIN_RE = re.compile(r"^[a-zA-Z0-9_\-/\.]+$")
PUNCT_CHARS = set(".,;:!?()[]{}«»\"\"\'\'…–—‒·•|/")

UPOS_MAP = {
    "N": "NOUN",
    "V": "VERB",
    "ADJ": "ADJ",
    "ADV": "ADV",
    "PRON": "PRON",
    "NUM": "NUM",
    "CONJ": "CCONJ",
    "PTCL": "PART",
    "INTJ": "INTJ",
    "POST": "ADP",
    "PREP": "ADP",
    "EXT": "VERB",
    "ENCL": "PART",
    "PARENTH": "ADV",
    "PART": "PART",
    "F": "PROPN",
    "M": "PROPN",
}

# Морфологические признаки [UD Features]
CASE_FEATS = {
    "nom": "Nom", "gen": "Gen", "dat": "Dat", "all":  "All",
    "abl": "Abl", "in": "Ine", "super": "Sup", "equ": "Equ",
    "comit": "Com", "adess": "Ade", "iness": "Ine", "dir": "Dir",
    "voc": "Voc",
}
NUMBER_FEATS = {"sg": "Sing", "pl": "Plur", "pltantum": "Plur"}
PERSON_FEATS = {"1": "1", "2": "2", "3": "3"}
TENSE_FEATS = {"prs": "Pres", "pst": "Past", "fut": "Fut"}
MOOD_FEATS = {"imp": "Imp", "sbjv": "Sub", "opt": "Opt", "cntrf": "Cnd"}
VERBFORM_FEATS = {
    "inf": "Inf", "ptcp": "Part", "ptcp.pst": "Part",
    "ptcp.ag": "Part", "ptcp.aeg": "Part", "ptcp.gae": "Part",
    "ptcp.inag": "Part", "nmlz": "Vnoun",
}


def build_ud_feats(tags: list[str]) -> str:
    feats = {}
    for t in tags:
        if t in CASE_FEATS: feats["Case"] = CASE_FEATS[t]
        elif t in NUMBER_FEATS: feats["Number"] = NUMBER_FEATS[t]
        elif t in PERSON_FEATS: feats["Person"] = PERSON_FEATS[t]
        elif t in TENSE_FEATS: feats["Tense"] = TENSE_FEATS[t]
        elif t in MOOD_FEATS: feats["Mood"] = MOOD_FEATS[t]
        elif t in VERBFORM_FEATS: feats["VerbForm"] = VERBFORM_FEATS[t]
        elif t == "neg": feats["Polarity"] = "Neg"
        elif t == "refl": feats["Reflex"] = "Yes"
    return "|".join(f"{k}={v}" for k, v in sorted(feats.items())) or "_"


def gramm_to_ud(lex: str, gr: str) -> tuple[str, str, str, str]:
    tags = [t.strip() for t in gr.split(",") if t.strip()]
    upos = next((UPOS_MAP[t] for t in tags if t in UPOS_MAP), "X")
    if "prop" in tags:
        upos = "PROPN"
    return lex, upos, gr, build_ud_feats(tags)


def classify_token(form: str) -> tuple[str, str, str, str] | None:
    if not form:
        return (form, "X", "_", "_")
    if (len(form) == 1 and form in PUNCT_CHARS) or form in {"...", "–", "—"}:
        return (form, "PUNCT", "_", "_")
    if NUM_RE.match(form):
        return (form, "NUM", "_", "_")
    if LATIN_RE.match(form):
        return (form.lower(), "X", "_", "_")
    if not CYRILLIC_RE.search(form):
        return (form, "X", "_", "_")
    return None


def load_wordlist_cache(wordlist_path: str) -> dict[str, tuple[str, str]]:
    ANA_RE = re.compile(r'<ana lex="([^"]*)" gr="([^"]*)"')
    FORM_RE = re.compile(r"</ana>([^<]+)</w>")

    cache = {}
    with open(wordlist_path, encoding="utf-8") as f:
        for line in f:
            fm   = FORM_RE.search(line)
            anas = ANA_RE.findall(line)
            if fm and anas:
                cache[fm.group(1).strip()] = anas[0]
    return cache


def load_analyzer(grammar_dir: str) -> Analyzer:
    a = Analyzer()
    a.g.DERIV_LIMIT = 2
    a.paradigmFile = f"{grammar_dir}/oss_paradigms.txt"
    a.lexFile = f"{grammar_dir}/oss_lexemes.txt"
    a.lexRulesFile  = ""
    a.derivFile = f"{grammar_dir}/oss_derivations.txt"
    a.conversionFile = ""
    a.cliticFile = f"{grammar_dir}/oss_clitics.txt"
    a.delAnaFile = f"{grammar_dir}/bad_analyses.txt"
    a.freqListFile = ""
    a.load_grammar()
    return a

def annotate(input_path: str, output_path: str, grammar_dir: str, wordlist_path: str, verbose: bool = True) -> dict:
    if verbose:
        print("Загрузка словарного кеша...")
    t0 = time.time()
    wl_cache = load_wordlist_cache(wordlist_path)
    if verbose:
        print(f"  {len(wl_cache):,} словоформ загружено за {time.time()-t0:.2f}s")
    if verbose:
        print("Поиск токенов для анализатора...")
    need_analysis = set()
    OSSETIC_RE = re.compile(r"[\u04d8\u04d9]")
    with open(input_path, encoding="utf-8") as f:
        for line in f:
            parts = line.strip().split("\t")
            if len(parts) == 10 and parts[0].isdigit():
                form = parts[1]
                if classify_token(form) is not None:
                    continue
                if form in wl_cache or form.lower() in wl_cache:
                    continue
                if OSSETIC_RE.search(form):
                    need_analysis.add(form)
    ana_cache: dict[str, tuple | None] = {}
    if need_analysis:
        if verbose:
            print(f"Анализ {len(need_analysis):,} токенов через UniParser...")
        t0 = time.time()
        a = load_analyzer(grammar_dir)
        batch = list(need_analysis)
        CHUNK = 500
        for i in range(0, len(batch), CHUNK):
            chunk = batch[i:i + CHUNK]
            results = a.analyze_words(chunk)
            for word, anas in zip(chunk, results):
                ana_list = anas if isinstance(anas, list) else [anas]
                best = next((x for x in ana_list if x.lemma), None)
                if best and best.lemma:
                    tags = [t.strip() for t in best.gramm.split(",") if t.strip()]
                    upos = next((UPOS_MAP[t] for t in tags if t in UPOS_MAP), "X")
                    if "prop" in tags:
                        upos = "PROPN"
                    ana_cache[word] = (
                        best.lemma, upos, best.gramm, build_ud_feats(tags)
                    )
                else:
                    ana_cache[word] = None
            if verbose and i % 5000 == 0 and i > 0:
                elapsed = time.time() - t0
                eta = elapsed / i * (len(batch) - i)
                print(f"  {i:,}/{len(batch):,} | {elapsed:.0f}s | ETA {eta:.0f}s")
        if verbose:
            found = sum(1 for v in ana_cache.values() if v)
            print(f"Анализатор покрыл: {found:,}/{len(ana_cache):,}")

    if verbose:
        print("Запись аннотированного файла...")
    Path(output_path).parent.mkdir(parents=True, exist_ok=True)

    stats = dict(total=0, wordlist=0, analyzer=0,
                 punct=0, num=0, noise=0, unknown=0)

    with open(input_path, encoding="utf-8") as fin, \
         open(output_path, "w", encoding="utf-8") as fout:

        for line in fin:
            raw = line.rstrip("\n")

            # Комментарии и пустые строки - без изменений
            if raw.startswith("#") or raw.strip() == "":
                fout.write(raw + "\n")
                continue

            parts = raw.split("\t")
            if len(parts) == 10 and parts[0].isdigit():
                form = parts[1]
                stats["total"] += 1

                # 1) Быстрая классификация (PUNCT, NUM, X)
                quick = classify_token(form)
                if quick:
                    lemma, upos, xpos, feats = quick
                    if upos == "PUNCT": stats["punct"]    += 1
                    elif upos == "NUM": stats["num"]      += 1
                    else:               stats["noise"]    += 1

                # 2) Словарный кеш (точное совпадение или строчные)
                elif form in wl_cache or form.lower() in wl_cache:
                    lex, gr = wl_cache.get(form) or wl_cache[form.lower()]
                    lemma, upos, xpos, feats = gramm_to_ud(lex, gr)
                    stats["wordlist"] += 1

                # 3) UniParser
                elif form in ana_cache and ana_cache[form]:
                    lemma, upos, xpos, feats = ana_cache[form]
                    stats["analyzer"] += 1

                # 4) Эвристика: заглавная буква → имя собственное
                else:
                    if form[0].isupper() and len(form) > 2:
                        lemma, upos, xpos, feats = form, "PROPN", "_", "_"
                    else:
                        lemma, upos, xpos, feats = form.lower(), "X", "_", "_"
                    stats["unknown"] += 1

                parts[2] = lemma
                parts[3] = upos
                parts[4] = xpos
                parts[5] = feats
                fout.write("\t".join(parts) + "\n")

            else:
                # Нестандартная строка (мусор от токенизатора)
                fout.write("# NOISE\t" + raw + "\n")

    return stats


def main():
    parser = argparse.ArgumentParser(description="Автоматическая морфоразметка осетинского корпуса (CoNLL-U)")
    parser.add_argument("--input",    default=DEFAULT_INPUT,    help="Входной CoNLL-U файл")
    parser.add_argument("--output",   default=DEFAULT_OUTPUT,   help="Выходной CoNLL-U файл")
    parser.add_argument("--grammar",  default=DEFAULT_GRAMMAR,  help="Папка с грамматикой UniParser")
    parser.add_argument("--wordlist", default=DEFAULT_WORDLIST, help="Предразмеченный словарь")
    parser.add_argument("--quiet", action="store_true", help="Без вывода прогресса")
    args = parser.parse_args()

    print("Осетинский корпус [автоматическая разметка]")

    t_start = time.time()
    stats = annotate(input_path = args.input, output_path = args.output, grammar_dir = args.grammar,
                     wordlist_path = args.wordlist, verbose = not args.quiet)
    elapsed = time.time() - t_start

    total = stats["total"]
    print("РЕЗУЛЬТАТЫ:\n")
    print(f"Всего токенов: {total:>10,}")
    print(f"Словарный кеш: {stats['wordlist']:>10,} ({100*stats['wordlist']/total:.1f}%)")
    print(f"UniParser: {stats['analyzer']:>10,} ({100*stats['analyzer']/total:.1f}%)")
    print(f"Пунктуация (PUNCT): {stats['punct']:>10,} ({100*stats['punct']/total:.1f}%)")
    print(f"Числа (NUM): {stats['num']:>10,} ({100*stats['num']/total:.1f}%)")
    print(f"Шум разметки (X): {stats['noise']:>10,} ({100*stats['noise']/total:.1f}%)")
    print(f"Неизвестные (X/PROPN): {stats['unknown']:>10,} ({100*stats['unknown']/total:.1f}%)")
    print(f"\nВремя: {elapsed:.1f}s | Файл: {args.output}")
    
if __name__ == "__main__":
    main()
