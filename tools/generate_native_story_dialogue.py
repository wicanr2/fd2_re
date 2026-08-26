#!/usr/bin/env python3
"""由原始 FDTXT control words 重生 handler 的逐句原生對話版面。"""

from __future__ import annotations

import argparse
import json
import struct
from pathlib import Path


SPEAKER_CONTROLS = set(range(0xFFEC, 0xFFF0))
ROW_BREAK = 0xFFFE
PAGE_BREAK = 0xFFFD
STRING_END = 0xFFFF


def load_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def parse_fdtxt(path: Path) -> list[list[int]]:
    raw = path.read_bytes()
    if len(raw) < 2:
        raise ValueError(f"{path}: shorter than first offset")
    first = struct.unpack_from("<H", raw)[0]
    if first == 0 or first % 2 or first > len(raw):
        raise ValueError(f"{path}: invalid first offset {first:#x}")
    offsets = list(struct.unpack_from(f"<{first // 2}H", raw))
    if any(offset < first or offset > len(raw) or offset % 2 for offset in offsets):
        raise ValueError(f"{path}: invalid offset table")
    if offsets != sorted(offsets):
        raise ValueError(f"{path}: descending offset table")
    strings: list[list[int]] = []
    for index, start in enumerate(offsets):
        end = offsets[index + 1] if index + 1 < len(offsets) else len(raw)
        words = []
        for cursor in range(start, end, 2):
            word = struct.unpack_from("<H", raw, cursor)[0]
            if word == STRING_END:
                break
            words.append(word)
        strings.append(words)
    return strings


def decode_layouts(source: str, string_index: int, words: list[int], glyphs: dict[int, str]):
    layouts = []
    cursor = 0
    while cursor < len(words):
        if words[cursor] not in SPEAKER_CONTROLS:
            cursor += 1
            continue
        if cursor + 1 >= len(words):
            raise ValueError(f"{source}#{string_index}: speaker control at end")
        control, operand = words[cursor], words[cursor + 1]
        cursor += 2
        pages: list[list[str]] = []
        page: list[str] = []
        row = ""

        def flush_row():
            nonlocal row
            if row:
                page.append(row)
                row = ""

        def flush_page():
            nonlocal page
            flush_row()
            if page:
                pages.append(page)
                page = []

        while cursor < len(words) and words[cursor] not in SPEAKER_CONTROLS:
            word = words[cursor]
            cursor += 1
            if word == ROW_BREAK:
                flush_row()
            elif word == PAGE_BREAK:
                flush_page()
            elif word >= 0xFF00:
                raise ValueError(f"{source}#{string_index}: unsupported control {word:#06x}")
            elif word not in glyphs:
                raise ValueError(f"{source}#{string_index}: glyph {word:#x} absent")
            else:
                row += glyphs[word]
        flush_page()
        if not pages or any(len(page_rows) > 64 for page_rows in pages):
            raise ValueError(
                f"{source}#{string_index}: empty or over-height page {pages!r}"
            )
        if any(len(row_text) > 13 for page_rows in pages for row_text in page_rows):
            raise ValueError(f"{source}#{string_index}: row exceeds 13 glyphs")
        layouts.append(
            {
                "source_dat": source,
                "string_index": string_index,
                "utterance": len(layouts),
                "control": f"{control:04X}",
                "operand": operand,
                "pages": pages,
            }
        )
    return layouts


def mapping_for(index_data, source: str, script: str, string_index: int):
    resources = [item for item in index_data["resources"] if item["source_dat"] == source]
    if len(resources) != 1:
        raise ValueError(f"{source}: resource mapping count={len(resources)}")
    scripts = [item for item in resources[0]["script_mappings"] if item["script"] == script]
    if len(scripts) != 1 or scripts[0]["status"] != "count_aligned":
        raise ValueError(f"{source}/{script}: no unique count-aligned mapping")
    mappings = [item for item in scripts[0]["mappings"] if item["string_index"] == string_index]
    if len(mappings) != 1 or len(mappings[0]["targets"]) != 1:
        raise ValueError(f"{source}/{script}#{string_index}: no unique target")
    return mappings[0], mappings[0]["targets"][0]


def generate(
    binding_path: Path,
    raw_dir: Path,
    glyph_map: Path,
    expected_callers: int,
    expected_utterances: int,
):
    binding = load_json(binding_path)
    handler = load_json((binding_path.parent / binding["handler_script"]).resolve())
    index_data = load_json((binding_path.parent / binding["story_index_map"]).resolve())
    encoded_glyphs = load_json(glyph_map)
    glyphs = {int(key): value for key, value in encoded_glyphs.items() if key != "_comment"}
    contexts = {key.lower(): value for key, value in binding["dialogue_contexts"].items()}
    raw_cache = {}
    overrides = {}
    callers = 0
    utterances = 0
    for beat in handler["beats"]:
        if beat["op"] != "dialog":
            continue
        callers += 1
        addr = beat["source"]["addr"].lower()
        string_index = beat["text_index"]
        context = contexts.get(addr)
        if context is None:
            raise ValueError(f"{addr}: missing dialogue_context")
        source, script = context["source_dat"], context["script"]
        if source not in raw_cache:
            raw_cache[source] = parse_fdtxt(raw_dir / f"{source}.bin")
        if string_index < 0 or string_index >= len(raw_cache[source]):
            raise ValueError(f"{addr}: {source} string {string_index} out of range")
        mapping, target = mapping_for(index_data, source, script, string_index)
        layouts = decode_layouts(source, string_index, raw_cache[source][string_index], glyphs)
        lines = target["lines"]
        if mapping["utterance_count"] != len(layouts) or len(lines) != len(layouts):
            raise ValueError(
                f"{addr}: mapping/raw/line count={mapping['utterance_count']}/{len(layouts)}/{len(lines)}"
            )
        typed_lines = []
        for line, layout in zip(lines, layouts):
            typed_lines.append(
                {
                    "line": line,
                    "upper": layout["control"] in ("FFED", "FFEF"),
                    "native_dialogue": layout,
                }
            )
        key = f"{addr}#{string_index}"
        overrides[key] = {
            "script": script,
            "scene_index": target["scene_index"],
            "lines": typed_lines,
        }
        utterances += len(layouts)
    if callers != expected_callers or utterances != expected_utterances:
        raise ValueError(
            f"expected {expected_callers} callers/{expected_utterances} utterances, "
            f"got {callers}/{utterances}"
        )
    binding["dialogue_overrides"] = overrides

    # caller-specific dialogue_overrides 完整建立後，移除同位址的舊現代
    # overrides.dialog，否則 compiler 的地址覆寫會遮蔽raw tuple版面。
    # 同一地址若另有其他具型別操作則必須保留；本工具不清除整個override。
    for beat in handler["beats"]:
        if beat["op"] != "dialog":
            continue
        addr = beat["source"]["addr"]
        for authored_addr in list(binding["overrides"]):
            if authored_addr.lower() != addr.lower():
                continue
            binding["overrides"][authored_addr].pop("dialog", None)
            if not binding["overrides"][authored_addr]:
                del binding["overrides"][authored_addr]
    return binding


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("binding", type=Path)
    parser.add_argument("--raw-dir", type=Path, required=True)
    parser.add_argument("--glyph-map", type=Path, required=True)
    parser.add_argument("--expected-callers", type=int, required=True)
    parser.add_argument("--expected-utterances", type=int, required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    generated = json.dumps(
        generate(
            args.binding.resolve(),
            args.raw_dir.resolve(),
            args.glyph_map.resolve(),
            args.expected_callers,
            args.expected_utterances,
        ),
        ensure_ascii=False,
        indent=2,
    ) + "\n"
    current = args.binding.read_text(encoding="utf-8")
    if args.check:
        if current != generated:
            raise SystemExit(f"{args.binding}: generated native dialogue binding is stale")
        return 0
    args.binding.write_text(generated, encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
