#!/usr/bin/env python3
"""把死亡效果降成各章劇本的 native_death_programs。僅限 Docker。

來源是 tools/extract_native_death_events.py 驗證過的轉寫
（remake/assets/data/native_death_events.json）與地圖單位的 death_effect：

- 型態 3 N：播章節戰場文字庫第 N 句。文字庫是 FDTXT 資源 [0x53C03]+1，也就是
  第 k 關（map k-1）用 FDTXT_0kk。
- 型態 2 id：全域事件 id 的動作清單；其中的對白同樣展開成逐段參照。

對白參照用 cutscenes/dialogue-index/count-aligned.json 對到故事腳本的場景與句子，
與首關回合事件同一條管線（generate_native_story_dialogue）。對不齊的章節不產生
參照，該章的死亡程式整筆不寫，runtime 失敗即關閉。

另外兩件事一起做：

- 死亡事件生成的群組（spawn_group／staging）從 initial_groups 移除——它們在原版
  是頭目倒下才出現，開局就在場會讓 runtime 記錄順序整個位移。
- 首關手寫的 on_unit_death「hawat_berserk」由原生事件 4 取代。

用法（容器內）：
  python3 tools/sync_native_death_programs.py --check
  python3 tools/sync_native_death_programs.py --write
"""

import argparse
import hashlib
import json
from pathlib import Path

from generate_native_story_dialogue import decode_layouts, mapping_for, parse_fdtxt
from import_editor_legacy import import_legacy, write_canonical

ROOT = Path(__file__).resolve().parents[1]
EVENTS = ROOT / "remake/assets/data/native_death_events.json"
MAPPING = ROOT / "remake/assets/cutscenes/dialogue-index/count-aligned.json"
GLYPHS = ROOT / "docs/data/glyph_map.json"
RAW_TEXT = ROOT / "extracted/raw/FDTXT"
SCENARIOS = ROOT / "remake/assets/scenarios"
MAPS = ROOT / "remake/assets/maps"
CANONICAL = ROOT / "remake/assets/editor-canonical"
LEGACY_DEATH_EVENTS = {"ch01.json": "hawat_berserk"}
# 0x1366A 的資源庫是 EXE 靜態 bank 的 106 項（doc50「acting resource library」），
# 全域共用；檔名 map32 是歷史名稱。
ACTING_LIBRARY = "assets/cutscenes/acting/map32.json"


def program_key(effect):
    return f'{effect["type"]}:{effect["value"]}'


class Chapter:
    def __init__(self, scenario_path: Path, mapping, glyphs):
        self.path = scenario_path
        self.scenario = json.loads(scenario_path.read_text(encoding="utf-8"))
        self.map = self.scenario["map"]
        self.source_dat = f"FDTXT_{self.map + 1:03d}"
        self.script = scenario_path.name
        self.mapping = mapping
        self.glyphs = glyphs
        raw = RAW_TEXT / f"{self.source_dat}.bin"
        self.strings = parse_fdtxt(raw) if raw.exists() else None
        self.inputs = [raw] if raw.exists() else []

    def dialogue(self, index, source, event_id=None):
        if self.strings is None:
            raise ValueError(f"{self.source_dat} 原始檔不存在")
        layouts = decode_layouts(self.source_dat, index, self.strings[index], self.glyphs)
        entry, targets = mapping_for(self.mapping, self.source_dat, self.script, index)
        refs = [(t["scene_index"], line) for t in targets for line in t["lines"]]
        if not (len(layouts) == entry["utterance_count"] == len(refs)):
            raise ValueError(f"{self.source_dat}#{index} 段數不一致")
        actions = []
        for (scene, line), layout in zip(refs, layouts):
            action = {"type": "dialogue", "native_source": source, "native_text_index": index,
                      "native_dialogue_ref": {"script": f"assets/story/{self.script}",
                                              "scene_index": scene, "line": line, **layout}}
            if event_id is not None:
                action["native_event_id"] = event_id
            actions.append(action)
        return actions


def lower(chapter: Chapter, event, event_id):
    """把一個事件的轉寫降成劇本動作。when 原樣帶到每個動作上。"""
    actions = []
    for o in event["ops"]:
        kind, source = o["op"], o["source"]
        when = o.get("when")
        if kind == "dialogue":
            lowered = chapter.dialogue(o["text"], source, event_id)
        elif kind == "pan":
            lowered = [{"type": "pan", "native_source": source, "grid": [o["x"], o["y"]]}]
        elif kind == "acting":
            lowered = [{"type": "native_acting", "native_source": source,
                        "native_acting": {"resource": o["resource"], "source": source}}]
        elif kind == "reset_pose":
            lowered = [{"type": "reset_pose", "native_source": source}]
        elif kind == "spawn_group":
            # 0x10B4E 建構器把 +5 寫成 0（0x10EED），新單位沒有「已行動」標記。
            lowered = [{"type": "spawn_group", "groups": [o["group"]], "act_immediately": True,
                        "native_spawns": [{"group": o["group"], "via": "spawn_group", "source": source,
                                           "raw_placement_gate": o["gate"]}]}]
        else:
            args = {k: v for k, v in o.items() if k not in ("op", "source", "tail_sources", "when", "rodata")}
            lowered = [{"type": "native_death_op", "native_source": source,
                        "native_death_op": {"op": kind, **args}}]
        for action in lowered:
            action.setdefault("native_event_id", event_id)
            if when:
                action["native_when"] = when
        actions.extend(lowered)
    return actions


def spawned_groups(event):
    groups = []
    for o in event["ops"]:
        if o["op"] in ("spawn_group", "staging"):
            groups.append(o["group"])
    return groups


def sync(write: bool):
    document = json.loads(EVENTS.read_text(encoding="utf-8"))
    events = {e["id"]: e for e in document["events"]}
    mapping = json.loads(MAPPING.read_text(encoding="utf-8"))
    glyphs = {int(k): v for k, v in json.loads(GLYPHS.read_text(encoding="utf-8")).items()
              if k != "_comment"}

    report, changed, problems = [], [], []
    for path in sorted(SCENARIOS.glob("ch[0-9][0-9].json")):
        chapter = Chapter(path, mapping, glyphs)
        units_path = MAPS / f"map{chapter.map}/map{chapter.map}_units.json"
        units = json.loads(units_path.read_text(encoding="utf-8"))["units"]
        effects = sorted({(u["death_effect"]["type"], u["death_effect"]["value"])
                          for u in units if u.get("death_effect") and u["death_effect"]["type"] in (2, 3)})
        programs, removed = {}, []
        for kind, value in effects:
            key = program_key({"type": kind, "value": value})
            try:
                if kind == 3:
                    # 0x1AC2B..0x1AC49：型態 3 直接以 payload 當文字索引。
                    programs[key] = chapter.dialogue(value, "0x1ac49")
                else:
                    event = events[value]
                    programs[key] = lower(chapter, event, value)
                    removed.extend(spawned_groups(event))
            except (KeyError, ValueError) as error:
                problems.append(f"{path.name} {key}：{error}")
        scenario = chapter.scenario
        before = json.dumps(scenario, ensure_ascii=False, indent=2) + "\n"
        if programs:
            scenario["native_death_programs"] = programs
            uses_acting = any(a["type"] == "native_acting" for actions in programs.values() for a in actions)
            if uses_acting and not scenario.get("native_acting_resources"):
                scenario["native_acting_resources"] = ACTING_LIBRARY
                report.append(f"{path.name} 死亡程式要播演出，補上全域演出資源庫")
        else:
            scenario.pop("native_death_programs", None)
        removed = sorted(set(removed))
        if removed:
            present = [g for g in removed if g in scenario["initial_groups"]]
            scenario["initial_groups"] = [g for g in scenario["initial_groups"] if g not in removed]
            if present:
                report.append(f"{path.name} 開局移除死亡事件生成的群組 {present}")
        legacy = LEGACY_DEATH_EVENTS.get(path.name)
        if legacy:
            scenario["events"] = [e for e in scenario["events"] if e.get("id") != legacy]
        after = json.dumps(scenario, ensure_ascii=False, indent=2) + "\n"
        if programs:
            report.append(f"{path.name} {len(programs)} 個死亡程式（{', '.join(sorted(programs))}）")
        if after != before:
            changed.append((path, after))

    for line in report:
        print(line)
    if problems:
        print("無法降階（該章對應程式不寫，runtime 失敗即關閉）：")
        for line in problems:
            print("  " + line)
    if not write:
        if changed:
            raise SystemExit(f"{len(changed)} 份劇本與死亡程式不一致；重跑 --write")
        print("劇本與死亡程式一致")
        return
    summary_path = CANONICAL / "bundle-summary.json"
    summary = json.loads(summary_path.read_text(encoding="utf-8"))
    for path, text in changed:
        path.write_text(text, encoding="utf-8")
        relative = f"remake/assets/scenarios/{path.name}"
        canonical_doc, diagnostics = import_legacy(json.loads(text), relative, "scenario")
        canonical_path = CANONICAL / "scenarios" / path.name
        write_canonical(canonical_doc, canonical_path)
        entries = [d for d in summary["documents"] if d["output"] == f"scenarios/{path.name}"]
        if len(entries) != 1 or entries[0]["document_id"] != canonical_doc["document_id"]:
            raise SystemExit(f"{path.name} 在 bundle-summary 找不到唯一的 canonical 條目")
        entries[0]["sha256"] = hashlib.sha256(canonical_path.read_bytes()).hexdigest()
        entries[0]["diagnostics"] = len(diagnostics)
        for item in summary["diagnostics"]:
            if item.get("source") == relative:
                item["items"] = diagnostics
    summary_path.write_text(json.dumps(summary, ensure_ascii=False, sort_keys=True, indent=2) + "\n",
                            encoding="utf-8")
    print(f"已更新 {len(changed)} 份劇本與對應的 canonical 文件")


def main():
    assert Path("/.dockerenv").exists(), "只允許在 Docker 內執行"
    ap = argparse.ArgumentParser()
    mode = ap.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--write", action="store_true")
    sync(ap.parse_args().write)


if __name__ == "__main__":
    main()
