#!/usr/bin/env python3
"""把四語全量內容目錄重新綁定到目前的字串清冊。

`content.json` 的暫定 `string_id` 內含檔名與行列號，Go 檔一改就漂移；整份重建
會把已審校的譯文換成新的機器初稿。這裡照 `migrate_string_review.py` 的作法，以
`(role, text, file, function, json_pointer)` 簽章把舊條目對到新清冊，已審校的
譯文與狀態原樣搬過去；只有清冊裡新出現、舊目錄找不到同一句來源文字的條目才用
`--new-translations` 提供的候選（狀態 `machine_draft`，不冒稱人工審校）。

用法：
  tools/migrate_full_locale_content.py --inventory docs/data/fd2-string-inventory.json \
      --review docs/data/fd2-string-review.json --canonical-root remake/assets/editor-canonical/story \
      --locales remake/assets/locales --new-translations <json>

`--new-translations` 是 `{來源繁中: {"zh-Hans": …, "en": …, "ja": …}}`；缺一筆就失敗，
不會靜默留下空譯文。
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from collections import defaultdict
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import build_full_locale_content as build  # noqa: E402

LOCALES = ("zh-Hant", "zh-Hans", "en", "ja")
# 與 cmd/fd2-string-inventory 的 formatVariable 同形：`%%` 不算變數。
FORMAT_VARIABLE = re.compile(r"%([0-9]+\$)?[-+# 0]*([0-9]+|\*)?(\.[0-9*]+)?[vTtbcdoOqxXUeEfgGswxp]")


def signature(entry: dict) -> tuple:
    source = entry["source"]
    return (entry["role"], entry["text"], source.get("file", ""), source.get("function", ""),
            source.get("json_pointer", ""))


def migrate(inventory_path: Path, review: dict, canonical_root: Path, locales_root: Path,
            new_translations: dict, glossary_path: Path) -> dict[str, dict]:
    inventory = build.read_json(inventory_path)
    if review["inventory_sha256"] != build.digest(inventory_path):
        raise ValueError("字串清冊與人工審查雜湊不一致；先跑 tools/rebind_string_review.sh")
    selected = build.selected_entries(inventory, review)
    stable_ids = build.canonical_story_ids(selected, canonical_root)
    old = {locale: build.read_json(locales_root / locale / "content.json") for locale in LOCALES}
    baseline = old["zh-Hant"]["entries"]
    by_id = {locale: {e["string_id"]: e for e in old[locale]["entries"]} for locale in LOCALES}
    # 同一簽章可能出現多次（同一個函式裡兩句一樣的話）；依原順序逐一配對。
    queue: dict[tuple, list[str]] = defaultdict(list)
    for entry in baseline:
        queue[signature(entry)].append(entry["string_id"])
    by_text: dict[str, str] = {}
    for entry in baseline:
        by_text.setdefault(entry["text"], entry["string_id"])
    outputs = {locale: [] for locale in LOCALES}
    stats = {"by_id": 0, "by_signature": 0, "by_text": 0, "new": 0}
    seen: set[str] = set()
    for entry in selected:
        string_id = stable_ids.get(entry["string_id"], entry["string_id"])
        if string_id in seen:
            raise ValueError(f"重複翻譯身分：{string_id}")
        seen.add(string_id)
        matched = None
        # 暫定 id 帶行號，新舊兩句不同的話可能撞在同一行；只認文字相同的同 id。
        if string_id in by_id["zh-Hant"] and by_id["zh-Hant"][string_id]["text"] == entry["text"]:
            matched, how = string_id, "by_id"
        elif queue.get(signature(entry)):
            matched, how = queue[signature(entry)].pop(0), "by_signature"
        elif entry["text"] in by_text:
            matched, how = by_text[entry["text"]], "by_text"
        else:
            how = "new"
        stats[how] += 1
        id_status = "stable_canonical" if entry["string_id"] in stable_ids else entry["id_status"]
        for locale in LOCALES:
            if matched is not None:
                previous = by_id[locale][matched]
                text, status = previous["text"], previous["status"]
                if locale == "zh-Hant" and text != entry["text"]:
                    raise ValueError(f"來源文字對不上：{string_id} {text!r} != {entry['text']!r}")
            elif locale == "zh-Hant":
                text, status = entry["text"], "source"
            else:
                candidates = new_translations.get(entry["text"])
                if not candidates or locale not in candidates:
                    raise ValueError(f"缺 {locale} 候選譯文：{entry['text']!r}")
                text, status = candidates[locale], "machine_draft"
                if [m.group(0) for m in FORMAT_VARIABLE.finditer(text)] != entry["variables"]:
                    raise ValueError(f"變數簽章改變：{string_id} {locale} {text!r}")
            outputs[locale].append({
                "string_id": string_id, "id_status": id_status, "role": entry["role"],
                "text": text, "variables": entry["variables"], "status": status,
                "source": entry["source"],
            })
    result = {}
    for locale in LOCALES:
        document = dict(old[locale])
        document["entries"] = outputs[locale]
        document["entry_count"] = len(outputs[locale])
        document["inventory_sha256"] = build.digest(inventory_path)
        document["glossary_sha256"] = build.digest(glossary_path)
        result[locale] = document
    print("配對：" + "、".join(f"{k}={v}" for k, v in stats.items()))
    return result


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--inventory", type=Path, required=True)
    parser.add_argument("--review", type=Path, required=True)
    parser.add_argument("--glossary", type=Path, default=Path("docs/data/localization/glossary.json"))
    parser.add_argument("--canonical-root", type=Path, required=True)
    parser.add_argument("--locales", type=Path, required=True)
    parser.add_argument("--new-translations", type=Path)
    args = parser.parse_args()
    new_translations = build.read_json(args.new_translations) if args.new_translations else {}
    result = migrate(args.inventory, build.read_json(args.review), args.canonical_root, args.locales,
                     new_translations, args.glossary)
    for locale, document in result.items():
        build.write_json(args.locales / locale / "content.json", document)
        print(f"{locale}: {document['entry_count']} 筆")


if __name__ == "__main__":
    try:
        main()
    except ValueError as exc:
        print(f"遷移失敗：{exc}", file=sys.stderr)
        raise SystemExit(1)
