#!/usr/bin/env python3
"""把已轉寫的回合事件處理器降成各章劇本的完整動作序列。僅限 Docker。

gen_campaign.py 只從 turn_events.json 產生「第 N 回合登場 group」的 spawn_group；
處理器裡的其餘動作（AI 模式範圍、鏡頭、演出、對白）它看不到。這裡拿
tools/extract_native_death_events.py 驗證過的全域事件轉寫
（remake/assets/data/native_death_events.json，回合事件與死亡效果同一張表 0x51B91），
把每章 turn_events 裡已轉寫的事件整筆降成劇本動作，取代原本只有 spawn 的版本；
沒有轉寫的事件維持 gen_campaign 的 spawn 版本，並列在報告裡。

對白參照、演出資源與 canonical 文件的更新方式與 sync_native_death_programs.py 相同。
每個動作都帶 native_event_id 與該列的 camp（0x1A813 phase selector：ally=1、enemy=0、
special=2），runtime 由此決定事件在回合裡的時機。

用法（容器內）：
  python3 tools/sync_native_turn_events.py --check --chapters 4,5
  python3 tools/sync_native_turn_events.py --write --chapters 4,5

--chapters 限定只改哪些章：每一章接上完整回合事件之後都要跑該章的 111 對拍才算數，
不要一次把 30 章全換掉（例如 ch13 的事件 5 也已轉寫，但那一章還沒對拍）。
"""

import argparse
import hashlib
import json
from pathlib import Path

from import_editor_legacy import import_legacy, write_canonical
from sync_native_death_programs import ACTING_LIBRARY, Chapter, lower, spawned_groups

ROOT = Path(__file__).resolve().parents[1]
EVENTS = ROOT / "remake/assets/data/native_death_events.json"
TURN_EVENTS = ROOT / "docs/data/turn_events.json"
EVENT_ID_GROUPS = ROOT / "docs/data/event_id_groups.json"
MAPPING = ROOT / "remake/assets/cutscenes/dialogue-index/count-aligned.json"
GLYPHS = ROOT / "docs/data/glyph_map.json"
SCENARIOS = ROOT / "remake/assets/scenarios"
CANONICAL = ROOT / "remake/assets/editor-canonical"


def bind_round(event, row):
    """group 取自回合計數 [0x53BEF] 的 spawn_group：回合事件列在回合計數等於該列回合時分派，
    所以代入 row["turn"]。其餘事件原樣回傳。"""
    if not any(o.get("group_from") == "round" for o in event["ops"]):
        return event
    ops = [dict(o, group=row["turn"]) if o.get("group_from") == "round" else o for o in event["ops"]]
    return {**event, "ops": ops}


def lowered_turn_event(chapter: Chapter, event, row, cid: str, group_sources):
    event = bind_round(event, row)
    actions = lower(chapter, event, row["event_id"])
    for action in actions:
        action["camp"] = row["camp"]
        if action["type"] == "spawn_group":
            # 與 gen_campaign 的形狀一致：groups 給正規化路徑，native_spawns 給原生路徑；
            # 呼叫來源沿用 event_id_groups.json 記的 `call 0x10b4e` 位址（轉寫的 source
            # 是該動作第一條指令，gate=1 時會是前面的 [0x53afa] 寫入）。
            for call in action["native_spawns"]:
                key = (row["event_id"], call["group"])
                if key not in group_sources:
                    # event_id_groups.json 對回合計數群組記的是 "$turn_counter[0x53bef]"。
                    key = (row["event_id"], "$turn_counter[0x53bef]")
                call["source"] = group_sources[key]
            action["groups"] = [call["group"] for call in action["native_spawns"]]
            action.pop("act_immediately", None)
    prefix = "reinforce" if spawned_groups(event) else "turn"  # event 已代入回合
    return {
        "id": f'{prefix}_ch{cid}_e{row["event_id"]}_t{row["turn"]}',
        "trigger": "on_turn_end",
        "when": {"turn": row["turn"]},
        "once": True,
        "do": actions,
    }


def sync(write: bool, chapters=None):
    events = {e["id"]: e for e in json.loads(EVENTS.read_text(encoding="utf-8"))["events"]}
    rows_by_chapter = {r["chapter"]: r["turn_events"] for r in json.loads(TURN_EVENTS.read_text(encoding="utf-8"))}
    group_sources = {(int(event_id), spawn["group"]): spawn["source"]
                     for event_id, entry in json.loads(EVENT_ID_GROUPS.read_text(encoding="utf-8")).items()
                     if event_id.isdigit() for spawn in entry.get("spawns", [])}
    mapping = json.loads(MAPPING.read_text(encoding="utf-8"))
    glyphs = {int(k): v for k, v in json.loads(GLYPHS.read_text(encoding="utf-8")).items()
              if k != "_comment"}

    report, changed, problems, pending = [], [], [], []
    for path in sorted(SCENARIOS.glob("ch[0-9][0-9].json")):
        chapter = Chapter(path, mapping, glyphs)
        if chapters and chapter.scenario["chapter"] not in chapters:
            continue
        cid = f"{chapter.scenario['chapter']:02d}"
        rows = rows_by_chapter.get(chapter.scenario["chapter"], [])
        scenario = chapter.scenario
        before = json.dumps(scenario, ensure_ascii=False, indent=2) + "\n"
        replaced = []
        for row in rows:
            event = events.get(row["event_id"])
            if event is None:
                pending.append(f'{path.name} event {row["event_id"]}@T{row["turn"]}')
                continue
            try:
                new_event = lowered_turn_event(chapter, event, row, cid, group_sources)
            except (KeyError, ValueError) as error:
                problems.append(f'{path.name} event {row["event_id"]}：{error}')
                continue
            ids = {f'reinforce_ch{cid}_e{row["event_id"]}_t{row["turn"]}',
                   f'turn_ch{cid}_e{row["event_id"]}_t{row["turn"]}'}
            scenario["events"] = [e for e in scenario["events"] if e.get("id") not in ids]
            scenario["events"].append(new_event)
            removed = [g for g in spawned_groups(bind_round(event, row)) if g in scenario["initial_groups"]]
            if removed:
                scenario["initial_groups"] = [g for g in scenario["initial_groups"] if g not in removed]
                report.append(f"{path.name} 開局移除回合事件生成的群組 {removed}")
            if any(a["type"] == "native_acting" for a in new_event["do"]) and \
                    not scenario.get("native_acting_resources"):
                scenario["native_acting_resources"] = ACTING_LIBRARY
                report.append(f"{path.name} 回合事件要播演出，補上全域演出資源庫")
            replaced.append(new_event["id"])
        # 回合事件依回合排序，opening 之類的手寫事件維持在前。
        turn_ids = {e["id"] for e in scenario["events"] if e.get("trigger") == "on_turn_end"
                    and (e["id"].startswith("reinforce_ch") or e["id"].startswith("turn_ch"))}
        fixed = [e for e in scenario["events"] if e["id"] not in turn_ids]
        turn_events = sorted((e for e in scenario["events"] if e["id"] in turn_ids),
                             key=lambda e: (e["when"]["turn"], e["id"]))
        scenario["events"] = fixed + turn_events
        after = json.dumps(scenario, ensure_ascii=False, indent=2) + "\n"
        if replaced:
            report.append(f"{path.name} {len(replaced)} 個回合事件（{', '.join(replaced)}）")
        if after != before:
            changed.append((path, after))

    for line in report:
        print(line)
    if pending:
        print(f"尚未轉寫的回合事件（維持 gen_campaign 的 spawn 版本）：{len(pending)} 筆")
    if problems:
        print("無法降階（該事件維持原樣）：")
        for line in problems:
            print("  " + line)
    if not write:
        if changed:
            raise SystemExit(f"{len(changed)} 份劇本與回合事件轉寫不一致；重跑 --write")
        print("劇本與回合事件轉寫一致")
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
    ap.add_argument("--chapters", default="",
                    help="只處理這些章（逗號分隔）；沒給就全部。每章接上後要跑該章的對拍再擴大。")
    args = ap.parse_args()
    chapters = {int(c) for c in args.chapters.split(",") if c.strip()}
    sync(args.write, chapters)


if __name__ == "__main__":
    main()
