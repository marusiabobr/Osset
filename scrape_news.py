"""
scrape_news.py - сбор статей с осетиноязычных сайтов.

Дампа у газет нет, поэтому скрейпинг нужен. тело статьи находит
trafilatura, писать селекторы под каждый сайт НЕ требуется.

Запуск:
    python scrape_news.py https://rastdzinad.ru news_rastdzinad.jsonl 300
"""
import sys
import json
import time
import urllib.robotparser as robotparser
from urllib.parse import urlparse

import trafilatura
from trafilatura.sitemaps import sitemap_search

from osnorm import normalize_ossetic, is_ossetic

DELAY_SEC = 2.0
USER_AGENT = "OsseticCorpusBot/1.0 (student research; contact: your@email)"


def allowed_by_robots(base_url: str) -> bool:
    parsed = urlparse(base_url)
    robots_url = f"{parsed.scheme}://{parsed.netloc}/robots.txt"
    rp = robotparser.RobotFileParser()
    try:
        rp.set_url(robots_url)
        rp.read()
    except Exception:
        return True
    return rp.can_fetch(USER_AGENT, base_url)


def collect_urls(base_url: str, limit: int) -> list:
    # trafilatura сам разбирает sitemap_index и вложенные sitemap'ы.
    # Язык не указываем 
    urls = sitemap_search(base_url)
    return list(dict.fromkeys(urls))[:limit]


def scrape(base_url: str, out_path: str, limit: int = 300) -> None:
    if not allowed_by_robots(base_url):
        print("robots.txt запрещает обход. Останавливаемся.")
        return

    urls = collect_urls(base_url, limit)
    print(f"Найдено URL в sitemap: {len(urls)}")
    kept = skipped = 0

    with open(out_path, "w", encoding="utf-8") as out:
        for i, url in enumerate(urls, 1):
            downloaded = trafilatura.fetch_url(url)
            if downloaded:
                raw = trafilatura.extract(downloaded)        # без языкового фильтра
                text = normalize_ossetic(raw or "")
                if is_ossetic(text, min_len=100):            # осетинский отбираем сами
                    out.write(json.dumps(
                        {"url": url, "source": "news", "text": text},
                        ensure_ascii=False) + "\n")
                    kept += 1
                else:
                    skipped += 1
            else:
                skipped += 1
            if i % 25 == 0:
                print(f"  {i}/{len(urls)} обработано, сохранено {kept}")
            time.sleep(DELAY_SEC)

    print(f"Готово. Сохранено: {kept}, пропущено: {skipped}. Файл: {out_path}")


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Использование: python scrape_news.py <base_url> <out.jsonl> [limit]")
        sys.exit(1)
    lim = int(sys.argv[3]) if len(sys.argv) > 3 else 300
    scrape(sys.argv[1], sys.argv[2], lim)
