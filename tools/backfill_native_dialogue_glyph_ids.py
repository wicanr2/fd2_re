#!/usr/bin/env python3
"""替既有原生對白版面補上 `glyph_ids`（原始 FDTXT 字模索引）。

glyph_map 有一字多模：584（兩點）與 347（一點）都解成「．」，另有「查」「、」「：」等
十一組。執行期由 token 查 `unicode_to_glyph.json` 的正規字模，原始 word 不是正規字模的
句子就會畫成另一個字模。`generate_native_story_dialogue.decode_layouts` 現在會輸出
`glyph_ids`；本工具把同一份結果回填到已產生的資料，並以原始位元組重解核對既有的
`pages`／`glyph_pages`，不一致就失敗、不寫檔。

editor-canonical 由 `tools/export_editor_canonical.py` 從正式資料重生，本工具不碰。

用法：
  tools/backfill_native_dialogue_glyph_ids.py          # 寫檔
  tools/backfill_native_dialogue_glyph_ids.py --check  # 有待補項就回非零
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "tools"))

from generate_native_story_dialogue import decode_layouts, parse_fdtxt  # noqa: E402

GLYPHS = ROOT / "docs/data/glyph_map.json"
RAW_TEXT = ROOT / "extracted/raw/FDTXT"
ASSETS = ROOT / "remake/assets"
SKIP = ASSETS / "editor-canonical"
LAYOUT_KEYS = {"source_dat", "string_index", "utterance", "control", "operand", "pages"}
TEXT_LAYOUT = re.compile(
    r'"source_dat": "(FDTXT_\d+)",\s*"string_index": (\d+),\s*"utterance": (\d+),'
    r'\s*"control": "\w+",\s*"operand": \d+,\s*"pages": '
)


class Decoder:
    def __init__(self):
        encoded = json.loads(GLYPHS.read_text(encoding="utf-8"))
        self.glyphs = {int(k): v for k, v in encoded.items() if k != "_comment"}
        self.strings: dict[str, list[list[int]]] = {}

    def layout(self, source: str, index: int, utterance: int) -> dict:
        if source not in self.strings:
            self.strings[source] = parse_fdtxt(RAW_TEXT / f"{source}.bin")
        return decode_layouts(source, index, self.strings[source][index], self.glyphs)[utterance]


def is_layout(node) -> bool:
    return (
        isinstance(node, dict)
        and LAYOUT_KEYS <= node.keys()
        and str(node["source_dat"]).startswith("FDTXT_")
    )


def expected_ids(decoder: Decoder, node: dict, where: str):
    decoded = decoder.layout(node["source_dat"], node["string_index"], node["utterance"])
    if decoded["pages"] != node["pages"] or decoded.get("glyph_pages") != node.get("glyph_pages"):
        raise ValueError(
            f"{where}: {node['source_dat']}#{node['string_index']}/{node['utterance']} "
            "現有 pages 與原始位元組重解不一致"
        )
    return decoded.get("glyph_ids")


def fix_tree(decoder: Decoder, node, where: str) -> int:
    changed = 0
    if isinstance(node, dict):
        if is_layout(node):
            ids = expected_ids(decoder, node, where)
            if ids != node.get("glyph_ids"):
                if ids is None:
                    node.pop("glyph_ids", None)
                else:
                    node["glyph_ids"] = ids
                changed += 1
        for value in node.values():
            changed += fix_tree(decoder, value, where)
    elif isinstance(node, list):
        for value in node:
            changed += fix_tree(decoder, value, where)
    return changed


def dump_style(text: str, data):
    for indent in (2, 1):
        rendered = json.dumps(data, ensure_ascii=False, indent=indent) + "\n"
        if rendered == text:
            return indent
    return None


def array_end(text: str, start: int) -> int:
    """回傳從 start 的 `[` 起配對的 `]` 之後位置（JSON 字串內的括號略過）。"""
    depth, cursor, in_string = 0, start, False
    while cursor < len(text):
        ch = text[cursor]
        if in_string:
            if ch == "\\":
                cursor += 1
            elif ch == '"':
                in_string = False
        elif ch == '"':
            in_string = True
        elif ch == "[":
            depth += 1
        elif ch == "]":
            depth -= 1
            if depth == 0:
                return cursor + 1
        cursor += 1
    raise ValueError("array 未閉合")


def fix_text(decoder: Decoder, text: str, where: str) -> tuple[str, int]:
    """格式無法整檔重寫的檔案：在 pages（或其後的 glyph_pages）之後插入一行 glyph_ids。"""
    out, cursor, changed = [], 0, 0
    for match in TEXT_LAYOUT.finditer(text):
        if match.start() < cursor:
            continue
        end = array_end(text, match.end())
        pages = json.loads(text[match.end():end])
        glyph_pages = None
        tail = re.match(r',\s*"glyph_pages": ', text[end:])
        if tail:
            gp_start = end + tail.end()
            gp_end = array_end(text, gp_start)
            glyph_pages = json.loads(text[gp_start:gp_end])
            end = gp_end
        existing = None
        ids_tail = re.match(r',\s*"glyph_ids": ', text[end:])
        if ids_tail:
            ids_end = array_end(text, end + ids_tail.end())
            existing = json.loads(text[end + ids_tail.end():ids_end])
        node = {
            "source_dat": match.group(1), "string_index": int(match.group(2)),
            "utterance": int(match.group(3)), "control": "", "operand": 0, "pages": pages,
        }
        if glyph_pages is not None:
            node["glyph_pages"] = glyph_pages
        ids = expected_ids(decoder, node, where)
        if existing is not None:
            if existing != ids:
                raise ValueError(f"{where}: 文字格式檔的既有 glyph_ids 與重解不一致，請手動核對")
            continue
        out.append(text[cursor:end])
        cursor = end
        if ids is not None:
            if "\n" in text[match.start():end]:
                line_start = text.rfind("\n", 0, match.start()) + 1
                indent = re.match(r"\s*", text[line_start:]).group(0)
                out.append(f',\n{indent}"glyph_ids": {json.dumps(ids)}')
            else:
                out.append(f', "glyph_ids": {json.dumps(ids)}')
            changed += 1
    out.append(text[cursor:])
    return "".join(out), changed


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    assert Path("/.dockerenv").exists(), "只允許在 Docker 內執行"
    decoder = Decoder()
    pending = 0
    for path in sorted(ASSETS.rglob("*.json")):
        if SKIP in path.parents:
            continue
        text = path.read_text(encoding="utf-8")
        if '"pages"' not in text or '"source_dat"' not in text:
            continue
        where = str(path.relative_to(ROOT))
        data = json.loads(text)
        indent = dump_style(text, data)
        if indent is not None:
            changed = fix_tree(decoder, data, where)
            updated = json.dumps(data, ensure_ascii=False, indent=indent) + "\n"
        else:
            updated, changed = fix_text(decoder, text, where)
            probe = json.loads(updated)
            if fix_tree(decoder, probe, where):
                raise ValueError(f"{where}: 文字插入後仍有版面未補齊")
        if changed:
            pending += changed
            print(f"{changed:4d} {where}")
            if not args.check:
                path.write_text(updated, encoding="utf-8")
    if args.check and pending:
        print(f"{pending} 句原生對白缺 glyph_ids", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
