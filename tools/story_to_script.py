#!/usr/bin/env python3
"""劇情文本(人工精校 transcript)→ 引擎 script JSON。

輸入:extracted/story/序章_transcript.md(本機,遊戲著作權內容,不入版控)。
輸出:remake/assets/story/chNN.json(assets/* 已 gitignore,不入公開 repo)。

只處理**有人工精校**的章節;`extracted/story/full_story_auto.md` 是自動解碼版,
glyph 對照表有系統性遞增偏移已知不可信,本工具不碰它(見 docs/knowledge-base 記憶)。

章節編號慣例:chapter = FDTXT_NNN 的 NNN(序章=FDTXT_001=chapter 1,對齊既有
remake/assets/scenarios/ch01.json 的 chapter 欄位),非 0-index。

speaker 對映:角色名 → face_portrait,來源 docs/data/exe_tables/characters.json
(32 筆,index==face_portrait)。查不到的 speaker(NPC/敵人/場景旁白)給 -1,
引擎不畫頭像,並列入缺口清單回報。
"""
import json
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
TRANSCRIPT = REPO / "extracted" / "story" / "序章_transcript.md"
CHARACTERS_JSON = REPO / "docs" / "data" / "exe_tables" / "characters.json"
OUT_DIR = REPO / "remake" / "assets" / "story"

CHAPTER_HEADER_RE = re.compile(r"^##\s+(.+?)\s*\(FDTXT_(\d+)\)\s*$")
LOCATION_RE = re.compile(r"^\*\*地點\*\*[:：]\s*(.+)$")
SCENE_MARKER_RE = re.compile(r"^〔(.+)〕\s*$")
DIALOGUE_BOLD_RE = re.compile(r"^-\s*\*\*(.+?)\*\*[:：]\s*(.+)$")
DIALOGUE_PAREN_RE = re.compile(r"^-\s*\((.+?)\)[:：]\s*(.+)$")
PAREN_STRIP_RE = re.compile(r"[\(（][^\)）]*[\)）]")


def load_speaker_map():
    chars = json.loads(CHARACTERS_JSON.read_text(encoding="utf-8"))
    return {c["name"]: c["face_portrait"] for c in chars}


def normalize_speaker(raw: str) -> str:
    """去掉附註性括號(如「海盜頭目(豹人)」「(刺客)約」→ 海盜頭目 / 約),核心名稱供對映查表。"""
    name = PAREN_STRIP_RE.sub("", raw).strip()
    return name if name else raw.strip()


def parse_transcript(text: str):
    """回傳 [(chapter_num, title, location, scenes)]，scenes = [(label_or_None, [(raw_speaker,text)])]。"""
    lines = text.splitlines()
    chapters = []
    cur = None  # dict: chapter, title, location, scenes(list of [label, lines])

    def start_scene(label):
        cur["scenes"].append({"label": label, "raw_lines": []})

    for line in lines:
        line = line.rstrip()
        m = CHAPTER_HEADER_RE.match(line)
        if m:
            if cur is not None:
                chapters.append(cur)
            cur = {
                "chapter": int(m.group(2)),
                "title": m.group(1).strip(),
                "location": None,
                "scenes": [],
            }
            start_scene(None)  # 隱含開場場景(在第一個〔〕之前)
            continue
        if cur is None:
            continue  # 檔頭說明,尚未進入任何章節
        loc = LOCATION_RE.match(line)
        if loc:
            cur["location"] = loc.group(1).strip()
            continue
        sm = SCENE_MARKER_RE.match(line)
        if sm:
            start_scene(sm.group(1).strip())
            continue
        db = DIALOGUE_BOLD_RE.match(line)
        if db:
            cur["scenes"][-1]["raw_lines"].append((db.group(1).strip(), db.group(2).strip()))
            continue
        dp = DIALOGUE_PAREN_RE.match(line)
        if dp:
            cur["scenes"][-1]["raw_lines"].append((dp.group(1).strip(), dp.group(2).strip()))
            continue
        # 其他(空行 / --- / > 註解 / 章節分隔)一律忽略,不是劇本正文
    if cur is not None:
        chapters.append(cur)
    return chapters


def build_chapter_json(chapter_data, speaker_map, gap_counter):
    scenes_out = []
    total_lines = 0
    resolved = 0
    for scene in chapter_data["scenes"]:
        if not scene["raw_lines"]:
            continue  # 隱含開場場景若無對白(章節開頭就是〔〕)則略過空場景
        lines_out = []
        for raw_speaker, text in scene["raw_lines"]:
            name = normalize_speaker(raw_speaker)
            portrait = speaker_map.get(name, -1)
            total_lines += 1
            if portrait != -1:
                resolved += 1
            else:
                gap_counter[name] = gap_counter.get(name, 0) + 1
            lines_out.append({"speaker": portrait, "speaker_name": name, "text": text})
        scenes_out.append({"label": scene["label"], "lines": lines_out})
    out = {
        "chapter": chapter_data["chapter"],
        "title": chapter_data["title"],
        "source_dat": f"FDTXT_{chapter_data['chapter']:03d}",
        "location": chapter_data["location"],
        "scenes": scenes_out,
    }
    return out, total_lines, resolved


def inventory_gap(known_chapters):
    """盤點 extracted/story/ 還有哪些章有 PNG 素材但缺人工精校。"""
    story_dir = REPO / "extracted" / "story"
    if not story_dir.exists():
        return []
    nums = set()
    for p in story_dir.glob("FDTXT_*_p*.png"):
        m = re.match(r"FDTXT_(\d+)_p\d+\.png", p.name)
        if m:
            nums.add(int(m.group(1)))
    missing = sorted(n for n in nums if n not in known_chapters)
    return missing


def main():
    if not TRANSCRIPT.exists():
        print(f"找不到人工精校 transcript:{TRANSCRIPT}", file=sys.stderr)
        return 1
    speaker_map = load_speaker_map()
    text = TRANSCRIPT.read_text(encoding="utf-8")
    chapters = parse_transcript(text)

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    gap_counter = {}
    known_chapters = []
    summary = []
    for ch in chapters:
        out, total, resolved = build_chapter_json(ch, speaker_map, gap_counter)
        known_chapters.append(ch["chapter"])
        out_path = OUT_DIR / f"ch{ch['chapter']:02d}.json"
        out_path.write_text(
            json.dumps(out, ensure_ascii=False, indent=1) + "\n", encoding="utf-8"
        )
        rate = (resolved / total * 100) if total else 0.0
        summary.append((ch["chapter"], ch["title"], total, resolved, rate, out_path))

    print("=== 轉出結果 ===")
    for num, title, total, resolved, rate, path in summary:
        print(f"ch{num:02d} 《{title}》: {total} 句, speaker 對映成功 {resolved}/{total} ({rate:.1f}%) -> {path}")

    print("\n=== speaker 對映缺口(用 -1,無頭像) ===")
    for name, cnt in sorted(gap_counter.items(), key=lambda x: -x[1]):
        print(f"  {name}: {cnt} 句")

    missing = inventory_gap(known_chapters)
    print("\n=== 文本精校缺口清單(有渲染 PNG 但無人工精校 transcript) ===")
    print(f"  已精校章節: {sorted(known_chapters)}")
    print(f"  缺精校章節(共 {len(missing)}): {missing}")
    print("  (以上章節僅有 extracted/story/FDTXT_NNN_p*.png 可讀渲染圖 + 不可信的自動解碼"
          " full_story_auto.md,需比照序章方式人工對照 PNG 轉錄)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
