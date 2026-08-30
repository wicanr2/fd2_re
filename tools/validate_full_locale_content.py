#!/usr/bin/env python3
"""Validate the four complete FD2 content catalogs."""
from __future__ import annotations

import hashlib
import json
import re
import sys
from pathlib import Path

LOCALES = ("zh-Hant", "zh-Hans", "ja", "en")
VAR_RE = re.compile(r"%(?:%|[-+0-9.#]*[a-zA-Z])")
CJK_RE = re.compile(r"[\u3400-\u9fff]")


def fail(message: str) -> None:
    raise ValueError(message)


def load(path: Path) -> dict:
    data = json.loads(path.read_text(encoding="utf-8"))
    required = {"schema_version", "kind", "locale", "source_locale", "entry_count",
                "inventory_sha256", "glossary_sha256", "pivot_content_sha256", "translation_engine", "entries"}
    if set(data) != required:
        fail(f"{path}: 頂層欄位不符合契約")
    if data["schema_version"] != 1 or data["kind"] != "fd2_full_locale_content":
        fail(f"{path}: 版本或 kind 不符")
    if data["locale"] not in LOCALES or data["source_locale"] != "zh-Hant":
        fail(f"{path}: 語系身分不符")
    if data["locale"] == "ja":
        if data["translation_engine"].startswith("NLLB "):
            if data["pivot_content_sha256"] is not None:
                fail(f"{path}: NLLB 直譯不得宣告英文中介")
        elif not isinstance(data["pivot_content_sha256"], str) or not re.fullmatch(r"[0-9a-f]{64}", data["pivot_content_sha256"]):
            fail(f"{path}: Argos 日文包缺英文中介雜湊")
    elif data["pivot_content_sha256"] is not None:
        fail(f"{path}: 非日文包不得宣告中介內容")
    entries = data["entries"]
    if data["entry_count"] != len(entries) or not entries:
        fail(f"{path}: entry_count 不符")
    seen = set()
    for entry in entries:
        expected = {"string_id", "id_status", "role", "text", "variables", "status", "source"}
        if set(entry) != expected:
            fail(f"{path}: {entry.get('string_id')} 欄位不符")
        key = entry["string_id"]
        if not key or key in seen:
            fail(f"{path}: 空白或重複 string_id {key}")
        seen.add(key)
        if not entry["text"]:
            fail(f"{path}: {key} 空文字")
        if VAR_RE.findall(entry["text"]) != entry["variables"]:
            fail(f"{path}: {key} 變數簽章不符")
        want_status = "source" if data["locale"] == "zh-Hant" else "machine_draft"
        if entry["status"] != want_status:
            fail(f"{path}: {key} 審查狀態不符")
        if data["locale"] == "en" and CJK_RE.search(entry["text"]):
            fail(f"{path}: {key} 英文包殘留 CJK 字形")
    return data


def validate(paths: list[Path]) -> None:
    packs = [load(path) for path in paths]
    if sorted(pack["locale"] for pack in packs) != sorted(LOCALES):
        fail("必須恰有四個官方語系")
    baseline = next(pack for pack in packs if pack["locale"] == "zh-Hant")
    by_locale = {pack["locale"]: pack for pack in packs}
    identity = [(e["string_id"], e["role"], e["variables"], e["source"]) for e in baseline["entries"]]
    for pack in packs:
        if pack["inventory_sha256"] != baseline["inventory_sha256"] or pack["glossary_sha256"] != baseline["glossary_sha256"]:
            fail(f"{pack['locale']}: 輸入雜湊不一致")
        got = [(e["string_id"], e["role"], e["variables"], e["source"]) for e in pack["entries"]]
        if got != identity:
            fail(f"{pack['locale']}: key／role／variables／source 契約不一致")
    baseline_entries = {entry["string_id"]: entry for entry in baseline["entries"]}
    english_entries = {entry["string_id"]: entry for entry in by_locale["en"]["entries"]}
    for entry in by_locale["ja"]["entries"]:
        key = entry["string_id"]
        text = entry["text"]
        source = baseline_entries[key]["text"]
        if "お問い合わせ" in text:
            fail(f"ja:{key}: 偵測到已知 Argos 污染詞")
        if len(text) > max(160, 4 * len(source)):
            fail(f"ja:{key}: 譯文異常膨脹")
        if entry["role"] in {"dialogue", "dialogue_or_system", "scene_label", "location_name", "chapter_title"}:
            if CJK_RE.search(source) and text == english_entries[key]["text"]:
                fail(f"ja:{key}: 敘事文字仍是完整英文")


if __name__ == "__main__":
    try:
        paths = [Path(arg) for arg in sys.argv[1:]]
        if not paths:
            paths = [Path("remake/assets/locales") / locale / "content.json" for locale in LOCALES]
        validate(paths)
    except (OSError, json.JSONDecodeError, ValueError) as exc:
        print(f"全量語系驗證失敗：{exc}", file=sys.stderr)
        raise SystemExit(1)
    print("已驗證四個全量內容目錄")
