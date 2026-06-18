"""Tests for scripts/regenerate_seeds.py (pure helpers only)."""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "regenerate_seeds.py"


def _load_module():
    spec = importlib.util.spec_from_file_location("regenerate_seeds", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules["regenerate_seeds"] = mod
    spec.loader.exec_module(mod)
    return mod


rs = _load_module()


@pytest.mark.parametrize(
    "raw,expected_subset",
    [
        ("район", {"район"}),
        ("мужчина; человек", {"мужчина; человек", "мужчина", "человек"}),
        ("а, б, в", {"а, б, в", "а", "б", "в"}),
        ("", set()),
    ],
)
def test_split_answers(raw: str, expected_subset: set[str]) -> None:
    got = set(rs.split_answers(raw))
    assert expected_subset.issubset(got)


def test_stable_shuffle_is_deterministic() -> None:
    items = ["A", "B", "C", "D", "E"]
    a = rs.stable_shuffle(items, "key")
    b = rs.stable_shuffle(items, "key")
    c = rs.stable_shuffle(items, "other")
    assert a == b
    assert sorted(a) == sorted(items)
    assert a != c or len(items) <= 1


def test_pick_case_options_includes_correct() -> None:
    pool = ["Именительный", "Родительный", "Дательный", "Направительный", "Отложительный"]
    opts = rs.pick_case_options("Родительный", pool, "lemma:form")
    assert "Родительный" in opts
    assert len(opts) == 4


def test_make_learn_word_seed_with_audio() -> None:
    seed = rs.make_learn_word_seed(
        "b1_01",
        {"lemma": "аз", "trans_ru": "год", "audio": "b1_01.ogg"},
    )
    assert seed["type"] == "vocab"
    assert seed["data"]["audio"] == "b1_01.ogg"
    assert "аудио скоро" not in seed["data"]["prompt"]


def test_make_learn_word_seed_without_audio() -> None:
    seed = rs.make_learn_word_seed("x", {"lemma": "x", "trans_ru": "y", "audio": None})
    assert seed["data"]["audio"] is None
    assert "аудио скоро" in seed["data"]["prompt"]


def test_sentence_variants_includes_lower_first_token() -> None:
    tokens = ["Уы", "лӕг", "хорз"]
    variants = rs.sentence_variants(tokens)
    assert " ".join(tokens) in variants
    assert "уы лӕг хорз" in variants
