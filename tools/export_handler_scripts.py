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


def normalize(beats):
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
            item = {"op": "spawn", "group": args[0], "source": src}
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
        elif op == "delay":
            item = {"op": "delay", "ms": args[0], "source": src}
        elif op == "activate_unit":
            item = {"op": "activate_unit", "source": src}
            if isinstance(args[0], int):
                item["unit_slot"] = args[0]
            else:
                item["unit_slot_expr"] = args[0]
        elif op == "spawn_intro":
            item = {"op": "spawn_intro", "group": args[0], "source": src}
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
        elif op == "unknown":
            item = {"op": "unknown", "native_target": beat["target"], "raw_args": args, "source": src}
        else:
            # Conditions and currently non-runtime operations remain editable
            # named records.  Keeping them prevents a lossy “known only” dump.
            item = {"op": op, "args": args, "source": src}
        out.append(item)
    if pending_chapter is not None:
        out.append({"op": "set_chapter", "chapter": pending_chapter,
                    "source": pending_chapter_source})
    return out


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
            "beats": normalize(handler["beats"]),
        }
        unknown = sum(1 for beat in script["beats"] if beat["op"] == "unknown")
        script["diagnostics"] = {"unknown_ops": unknown}
        path = os.path.join(outdir, f"ch{chapter:02d}_{tag}.json")
        with open(path, "w", encoding="utf-8") as f:
            json.dump(script, f, ensure_ascii=False, indent=2)
            f.write("\n")
        summary.append({"chapter": chapter, "phase": tag, "handler": handler["handler"],
                        "beats": len(script["beats"]), "unknown_ops": unknown})
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
