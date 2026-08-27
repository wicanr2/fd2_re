#!/usr/bin/env python3
"""Export FD2 EXE cutscene handlers into editable, versioned Handler Script IR.

The EXE remains the evidence source.  This program consumes the deterministic
instruction-level output from dump_chapter_beats.py and emits JSON that is safe
to edit: calls become named operations, while source addresses stay alongside
them for reverse-engineering audit.  Unknown calls are deliberately retained
as ``op: unknown`` rather than silently discarded.

Usage:
  python3 tools/export_handler_scripts.py <FD2.EXE> all <outdir>
  python3 tools/export_handler_scripts.py <FD2.EXE> ch0 <outdir>
"""
import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(__file__))
import dump_chapter_beats as raw


SCHEMA_VERSION = 1


def as_int(value):
    """Return an immediate integer, otherwise preserve the original operand."""
    if isinstance(value, int):
        return value
    if isinstance(value, str):
        try:
            return int(value, 0)
        except ValueError:
            pass
    return value


def source_of(beat):
    src = {"addr": beat["addr"]}
    if "target" in beat:
        src["target"] = beat["target"]
    return src


def repeat_of(beat):
    hint = beat.get("repeat_hint")
    if not hint or not isinstance(hint.get("limit"), int):
        return None
    # Handler loops use a zero-based counter compared after the call.  A cmp
    # of N means exactly N calls in the verified chapter-0 loops.
    return hint["limit"]


def normalize(beats, chapter=None):
    """Convert raw disassembly beats to the stable editable IR."""
    out = []
    pending_chapter = None
    pending_chapter_source = None
    for beat in beats:
        op = beat["op"]
        args = [as_int(arg) for arg in beat.get("args", [])]
        src = source_of(beat)
        if op == "loadch_var":
            pending_chapter = as_int(beat["chapter"])
            pending_chapter_source = src
            continue
        if op == "loadch_call":
            item = {"op": "loadch", "source": src}
            if isinstance(pending_chapter, int):
                item["chapter"] = pending_chapter
            else:
                item["chapter_expr"] = pending_chapter
            pending_chapter = None
            pending_chapter_source = None
        elif op == "pan":
            item = {"op": "pan", "grid_x": args[0], "grid_y": args[1], "source": src}
        elif op == "dialog":
            item = {"op": "dialog", "text_index": args[1], "source": src}
            if isinstance(args[0], str):
                item["text_table"] = args[0]
        elif op == "act":
            item = {"op": "act", "acting_id": args[0], "source": src}
        elif op == "spawn":
            item = {
                "op": "spawn", "group": args[0],
                "raw_placement_gate": beat["raw_placement_gate"], "source": src,
            }
        elif op == "join":
            item = {"op": "join", "char_id": args[0], "source": src}
        elif op == "bgm":
            item = {"op": "bgm", "track": args[0], "loop": args[1], "source": src}
        elif op == "scroll_step":
            # 0x13185 follows the supplied original unit slot while scrolling;
            # its argument is not a compass direction.
            item = {"op": "scroll_step", "unit_slot": args[0], "source": src}
            repeat = repeat_of(beat)
            if repeat is not None:
                item["repeat"] = repeat
        elif op == "palfade":
            item = {"op": "palette_fade", "source": src}
        elif op.startswith("native_"):
            # A named native operation means only that the callee's narrow
            # role and arity are evidence-backed. Preserve source PUSH order;
            # the compiler still owns caller-specific lowering and fails closed
            # when the exact ABI or binding is absent.
            item = {
                "op": "native_call",
                "native_semantic": NATIVE_SEMANTICS.get(beat["target"], op),
                "native_target": beat["target"],
                "raw_args": args,
                "native_confidence": "已證實",
                "native_evidence": [
                    evidence for evidence in NATIVE_EVIDENCE[beat["target"]]
                ],
                "source": src,
            }
        elif op == "delay":
            item = {"op": "delay", "ms": args[0], "source": src}
        elif op == "deactivate_unit":
            item = {"op": "deactivate_unit", "source": src}
            if isinstance(args[0], int):
                item["unit_slot"] = args[0]
            else:
                item["unit_slot_expr"] = args[0]
        elif op == "spawn_intro":
            item = {
                "op": "spawn_intro", "group": args[0],
                "raw_placement_gate": beat["raw_placement_gate"], "source": src,
            }
        elif op == "layout_units":
            # 0x233c6 reads call-site-specific X/Y/pose arrays through
            # registers. Preserve the native call as a named operation; an
            # address-keyed binding supplies the recovered absolute layout.
            item = {"op": "layout_units", "source": src}
        elif op == "reset_pose":
            item = {"op": "reset_pose", "source": src}
        elif op == "focus_unit":
            item = {"op": "focus_unit", "source": src}
            if isinstance(args[0], int):
                item["unit_slot"] = args[0]
            else:
                item["unit_slot_expr"] = args[0]
        elif op == "sync_party":
            item = {"op": "sync_party", "source": src}
        elif op == "grant_item":
            item = {"op": "grant_item", "item_id": args[0], "source": src}
        elif op == "increment_chapter":
            if isinstance(chapter, int):
                item = {"op": "set_chapter", "chapter": chapter + 1, "source": src}
            else:
                item = {"op": "increment_chapter", "source": src}
        elif op == "if":
            item = {
                "op": "if",
                "condition": beat["condition"],
                "then": normalize(beat.get("then", []), chapter),
                "else": normalize(beat.get("else", []), chapter),
                "source": src,
            }
        elif op == "unknown":
            target = beat["target"]
            evidence = NATIVE_EVIDENCE.get(target)
            unresolved = UNRESOLVED_NATIVE.get(target)
            if evidence:
                item = {
                    "op": "native_call",
                    "native_target": target,
                    "native_semantic": NATIVE_SEMANTICS[target],
                    "native_confidence": "已證實",
                    "native_evidence": list(evidence),
                    "raw_args": args,
                    "source": src,
                }
            elif unresolved:
                item = {
                    "op": "unresolved_native_call",
                    "native_target": target,
                    "native_semantic": unresolved["semantic"],
                    "native_confidence": unresolved["confidence"],
                    "native_evidence": list(unresolved["evidence"]),
                    "raw_args": args,
                    "source": src,
                }
            else:
                item = {"op": "unknown", "native_target": target, "raw_args": args, "source": src}
        else:
            # Conditions and currently non-runtime operations remain editable
            # named records.  Keeping them prevents a lossy “known only” dump.
            item = {"op": op, "args": args, "source": src}
        out.append(item)
    if pending_chapter is not None:
        out.append({"op": "set_chapter", "chapter": pending_chapter,
                    "source": pending_chapter_source})
    return out


# 這份索引只涵蓋 raw chapter handler 會輸出的 native calls；它與完整
# docs/data/ida/fd2_semantic_index.json 共用同一證據契約。export 時把等級與
# evidence 內嵌到每筆 beat，避免名稱脫離證據後被誤當成完整 runtime 語意。
NATIVE_EVIDENCE = {
    "0x11d40": ["docs/data/ida/fd2_ch23_post_ida.txt"],
    "0x11df2": ["docs/data/fd2_11df2_palette_disasm.txt"],
    "0x12cea": ["docs/data/ida/fd2_ch28_post_ida.txt"],
    "0x13536": ["docs/data/ida/fd2_ch09_post_ida.txt"],
    "0x17aa9": ["docs/data/ida/fd2_ch29_input_cleanup_ida.txt"],
    "0x1b8e7": ["docs/knowledge-base/91-worklist.md"],
    "0x1c2da": ["docs/knowledge-base/91-worklist.md"],
    "0x1f882": ["docs/data/ida/fd2_ch21_post_ida.txt"],
    "0x2189a": [
        "docs/data/ida/fd2_ch22_post_ida.txt",
        "docs/data/ida/fd2_command10_12_presentation_ida.txt",
    ],
    "0x22253": [
        "docs/data/ida/fd2_ch28_post_ida.txt",
        "docs/knowledge-base/54-acting-pose-semantics.md",
    ],
    "0x24618": ["docs/data/ida/fd2_ch21_post_ida.txt"],
    "0x24b14": ["docs/data/ida/fd2_ch22_post_ida.txt"],
    "0x24b4d": ["docs/data/ida/fd2_ch22_post_ida.txt"],
    "0x24bde": ["docs/data/ida/fd2_ch22_post_ida.txt"],
    "0x24d22": [
        "docs/data/ida/fd2_ch23_post_ida.txt",
        "docs/data/fd2_chapter_aux_graphics_10652_ida.txt",
    ],
    "0x25052": ["docs/knowledge-base/91-worklist.md"],
    "0x25089": ["docs/knowledge-base/91-worklist.md"],
    "0x2bce5": [
        "docs/data/ida/fd2_ch29_terminal_body_ida.txt",
        "docs/data/ida/fd2_ch29_post_montage_tail_ida.txt",
    ],
    "0x31860": ["docs/data/ida/fd2_ch22_post_ida.txt"],
    "0x33f78": ["docs/knowledge-base/91-worklist.md"],
    "0x35822": ["docs/data/ida/fd2_ch27_ch28_pre_owner_ida.txt"],
    "0x35bba": ["docs/data/ida/fd2_ch28_post_ida.txt"],
    "0x35e5a": ["docs/data/ida/fd2_ch28_post_ida.txt"],
    "0x37416": ["docs/data/ida/fd2_ch23_post_ida.txt"],
    "0x24336": ["docs/data/ida/fd2_ch20_sky_key_sequence_ida.txt"],
    "0x4dbfc": ["docs/data/ida/fd2_ch22_post_ida.txt"],
}

NATIVE_SEMANTICS = {
    "0x2189a": "十輪 LUT／radius／snapshot／object redraw／viewport copy 索引呈現",
    "0x22253": "具五參數 ABI 與座標 commit 邊界的 11＋6＋10 indexed 呈現排程",
    "0x24336": "ch20 天空之鑰固定合成演出序列",
    "0x24bde": "persistent raw +8 identity exact-match lookup",
    "0x24d22": "raw stage latch setter 與 312-byte staging 列旋轉",
    "0x2bce5": "終局前綴、文字、蒙太奇與尾段的來源約束 owner",
}

# Callee 有窄證據，但仍缺 caller-specific ABI、renderer 或 campaign owner。
# 這些不能算「完全未知」，也不能當作可執行 native_call。
UNRESOLVED_NATIVE = {}


def walk_beats(beats):
    """Yield editable beats recursively so diagnostics include branch arms."""
    for beat in beats:
        yield beat
        if beat.get("op") == "if":
            yield from walk_beats(beat.get("then", []))
            yield from walk_beats(beat.get("else", []))


def export_table(cg, fx, entries, tag, outdir):
    unique = sorted({handler for _, handler in entries})
    table = raw.handler_beats(cg, fx, entries, unique, raw.OBJ1_END)
    summary = []
    for chapter, handler in table.items():
        script = {
            "schema_version": SCHEMA_VERSION,
            "chapter": chapter,
            "phase": tag,
            "handler": handler["handler"],
            "beats": normalize(handler["beats"], chapter),
        }
        unknown = sum(1 for beat in walk_beats(script["beats"]) if beat["op"] == "unknown")
        native = sum(1 for beat in walk_beats(script["beats"]) if beat["op"] == "native_call")
        unresolved = sum(1 for beat in walk_beats(script["beats"]) if beat["op"] == "unresolved_native_call")
        script["diagnostics"] = {"unknown_ops": unknown}
        if native:
            script["diagnostics"]["classified_native_ops"] = native
        if unresolved:
            script["diagnostics"]["unresolved_native_ops"] = unresolved
        path = os.path.join(outdir, f"ch{chapter:02d}_{tag}.json")
        with open(path, "w", encoding="utf-8") as f:
            json.dump(script, f, ensure_ascii=False, indent=2)
            f.write("\n")
        summary_item = {"chapter": chapter, "phase": tag, "handler": handler["handler"],
                        "beats": len(script["beats"]), "unknown_ops": unknown}
        if native:
            summary_item["classified_native_ops"] = native
        if unresolved:
            summary_item["unresolved_native_ops"] = unresolved
        summary.append(summary_item)
    return summary


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("exe")
    parser.add_argument("scope", choices=("all", "ch0"))
    parser.add_argument("outdir")
    args = parser.parse_args(argv)

    cg = raw.CG(args.exe)
    fx = raw.fixup_map(cg.d, cg.meta)
    os.makedirs(args.outdir, exist_ok=True)
    pre = raw.resolve_table(fx, raw.TABLE_PRE, raw.N_CHAPTERS)
    post = raw.resolve_table(fx, raw.TABLE_POST, raw.N_CHAPTERS)
    if args.scope == "ch0":
        pre = [entry for entry in pre if entry[0] == 0]
        post = [entry for entry in post if entry[0] == 0]
    summary = export_table(cg, fx, pre, "pre", args.outdir)
    summary.extend(export_table(cg, fx, post, "post", args.outdir))
    with open(os.path.join(args.outdir, "_manifest.json"), "w", encoding="utf-8") as f:
        json.dump({"schema_version": SCHEMA_VERSION, "scripts": summary}, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(f"exported {len(summary)} handler scripts to {args.outdir}")


if __name__ == "__main__":
    main()
