#!/usr/bin/env python3
"""依第30章READY原始文字庫契約重生2BCE5正常終局對話；僅限Docker。"""
import argparse
import hashlib
import json
from pathlib import Path
import tempfile

from sync_native_death_programs import Chapter, MAPPING, GLYPHS
from generate_native_story_dialogue import parse_fdtxt, canonical_glyphs
from export_story_index_map import build_count_aligned_mapping, load_story

ROOT = Path(__file__).resolve().parents[1]
TARGET = ROOT / "remake/assets/endings/native_2bce5.json"
CONTRACT = ROOT / "docs/data/ida/fd2_ch30_flow_20261017.json"


def plain_layout(words, glyphs, index):
    pages, page, row, ids, id_page, id_row = [], [], [], [], [], []
    def flush_row():
        if row:
            page.append("".join(row)); id_page.append(list(id_row))
            row.clear(); id_row.clear()
    def flush_page():
        flush_row()
        if page:
            pages.append(list(page)); ids.append(list(id_page))
            page.clear(); id_page.clear()
    for word in words:
        if word == 0xFFFE:
            flush_row()
        elif word == 0xFFFD:
            flush_page()
        elif word >= 0xFF00 or word not in glyphs:
            raise ValueError(f"FDTXT031#{index}: unexpected word {word:#x}")
        else:
            row.append(glyphs[word]); id_row.append(word)
    flush_page()
    if not pages:
        raise ValueError(f"FDTXT031#{index}: empty message")
    result = {"pages": pages}
    canonical = canonical_glyphs()
    if any(canonical.get(glyphs[word]) != word for page in ids for row in page for word in row):
        result["glyph_ids"] = ids
    return result


def generate():
    contract = json.loads(CONTRACT.read_text(encoding="utf-8"))
    spec = contract["post_ending_contract"]
    if spec["status"] not in ("READY", "CONFORMED"):
        raise ValueError("終局文字規格尚未READY")
    raw = ROOT / "extracted/raw/FDTXT/FDTXT_031.bin"
    if hashlib.sha256(raw.read_bytes()).hexdigest() != spec["text_resources"]["31"]:
        raise ValueError("FDTXT_031來源雜湊不同")
    mapping = json.loads(MAPPING.read_text(encoding="utf-8"))
    glyphs = {int(k): v for k, v in json.loads(GLYPHS.read_text(encoding="utf-8")).items() if k != "_comment"}
    strings = parse_fdtxt(raw)
    story_path = ROOT / "remake/assets/story/ch31.json"
    story = json.loads(story_path.read_text(encoding="utf-8"))
    lines = story["scenes"][2]["lines"]
    missing = [{"speaker": -1, "speaker_name": "(旁白)",
                "text": "".join(sum(plain_layout(strings[index], glyphs, index)["pages"], []))}
               for index in range(40, 46)]
    if len(lines) == 30:
        lines.extend(missing)
    elif len(lines) != 36 or lines[30:] != missing:
        raise ValueError("ch31尾聲與FDTXT031 strings40..45不一致")
    story_text = json.dumps(story, ensure_ascii=False, indent=2) + "\n"
    document = json.loads(TARGET.read_text(encoding="utf-8"))
    with tempfile.TemporaryDirectory() as tmp:
        # Chapter只用map+1定位原始字庫；此檔是生成器輸入，不是新增可玩章節。
        source = Path(tmp) / "ch31.json"
        source.write_text(json.dumps({"map": 30}), encoding="utf-8")
        editable = Path(tmp) / "story"; editable.mkdir()
        (editable / "ch31.json").write_text(story_text, encoding="utf-8")
        mapped = build_count_aligned_mapping(strings, load_story(editable / "ch31.json", editable))
        resource = next(r for r in mapping["resources"] if r["source_dat"] == "FDTXT_031")
        resource["script_mappings"] = [mapped]
        mapping["diagnostics"] = [d for d in mapping["diagnostics"]
                                  if not (d.get("source_dat") == "FDTXT_031" and d.get("script") == "ch31.json")]
        chapter = Chapter(source, mapping, glyphs)
        for segment in document["segments"]:
            if segment.get("op") != "native_text_branch_opaque":
                continue
            for block in segment["else_dialogue"]:
                index = block["string_index"]
                if not 2 <= index <= 7:
                    raise ValueError("2BCE5正常分支文字索引越界")
                entry = mapped["mappings"][index]
                target = entry["targets"][0]
                if entry["utterance_count"] != 1 or len(entry["targets"]) != 1:
                    raise ValueError("2C39B單句文字結構不符")
                refs = [{"scene_index": target["scene_index"], "line": target["lines"][0],
                         "control": "", "operand": block["portrait_id"],
                         **plain_layout(strings[index], glyphs, index)}]
                if not refs or len({ref["scene_index"] for ref in refs}) != 1:
                    raise ValueError("終局來源段落無法以同一場景表達")
                block.update(source_dat="FDTXT_031", script="ch31.json", scene_index=refs[0]["scene_index"], line=refs[0]["line"], count=len(refs))
                utterances = []
                for ref in refs:
                    native = {"control_source": "fdtxt" if ref["control"] else "caller_2c39b",
                              "control": ref["control"],
                              "operand": ref["operand"] if ref["control"] else block["portrait_id"],
                              "pages": ref["pages"]}
                    for key in ("glyph_pages", "glyph_ids"):
                        if key in ref:
                            native[key] = ref[key]
                    utterances.append(native)
                block["native_utterances"] = utterances
    return {TARGET: json.dumps(document, ensure_ascii=False, indent=2) + "\n",
            story_path: story_text, MAPPING: json.dumps(mapping, ensure_ascii=False, indent=2) + "\n"}


def main():
    assert Path("/.dockerenv").exists(), "僅限Docker"
    parser = argparse.ArgumentParser()
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--write", action="store_true")
    args = parser.parse_args()
    for path, text in generate().items():
        if args.write:
            path.write_text(text, encoding="utf-8")
        elif path.read_text(encoding="utf-8") != text:
            raise SystemExit(f"{path}: 終局文字與已證實FDTXT031不一致；重跑--write")
    print("正常終局2BCE5六組文字與FDTXT031原始controls／glyphs相符；26分支保持。")


if __name__ == "__main__":
    main()
