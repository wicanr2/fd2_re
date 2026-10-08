#!/usr/bin/env python3
"""編譯及驗證主程式區的 C 候選，所有完整區間逐位元組比較。"""

import argparse
import hashlib
import itertools
import json
import os
from pathlib import Path
import re
import subprocess

from fd2_matching_c_restore import INPUTS


CASES = (("CLEAR3", 0x134E4), ("REDRAW", 0x127A9), ("CYCLE", 0x1F525), ("MASK", 0x146A7),
         ("ROWCOPY", 0x11EB0), ("RELEASE", 0x15E71), ("CONDITIONAL_FREE", 0x1A7F1),
         ("CACHE_INIT", 0x1D4CB), ("CONDITIONAL_INIT", 0x1A7BD), ("INVENTORY", 0x1B8E7),
         ("SET_BIT7", 0x13512), ("CLEAR_BIT7", 0x13536), ("SLOT_BYTE", 0x1B722),
         ("SET_BYTE5", 0x32975), ("GET_BIT0", 0x3453E), ("RANGE_LOW_BYTE", 0x3419C),
         ("COPY_WORDS", 0x25089), ("COMPARE_VALUE", 0x2EF8F))
RECORD_CASES = (("CURSOR_Y_DEC", 0x11B48), ("CURSOR_Y_INC", 0x11B9B),
                ("CURSOR_X_INC", 0x11BFA), ("CURSOR_X_DEC", 0x11C59),
                ("FIND_RECORD", 0x12C0D), ("SUM_FIELD", 0x15DA2),
                ("DERIVED_WORDS", 0x1145A), ("BLIT_CELL", 0x126F7),
                ("READ_INFO", 0x12E38), ("CONDITIONAL_INFO", 0x13A44))
COPY_CASES = (("VIDEO_COPY", 0x16559), ("RESTORE_72", 0x17643),
              ("HCLIP_86", 0x182AD), ("VCLIP_86", 0x18312),
              ("BOTTOM_102", 0x1839B), ("PANEL_COPY", 0x1AF99),
              ("INSERT_CELL", 0x1BB8C))
QUERY_CASES = (("MEMBER_SIX", 0x1C1C3), ("FACING_BYTES", 0x1F04A),
               ("BOX_POSITION", 0x2A289), ("REVERSE_INFO", 0x2B5E1),
               ("RATIO_WORDS", 0x1E7F6), ("FILL_SQUARE", 0x1F6EF))
CONTROL_CASES = tuple(("F" + format(address, "X"), address) for address in (
    0x10620, 0x13460, 0x1685C, 0x16886, 0x173E7, 0x17AA9, 0x1875D, 0x18795,
    0x1C8ED, 0x1CA89, 0x1F183, 0x206C5, 0x20707, 0x2073D, 0x20822, 0x2084A,
    0x20926, 0x21206, 0x21396, 0x21527))
BATTLE_CASES = tuple(("F" + format(address, "X"), address) for address in (
    0x13565, 0x14625, 0x14B16, 0x164E8, 0x175A9, 0x1B83D, 0x1B8A6,
    0x1C142, 0x1C220)) + (("COUNTS_GROUP", 0x1B5F1),)
EVENT_CASES = (("F21082", 0x21082), ("EVENT_PAIR", 0x2111A), ("F214AD", 0x214AD))
LAYOUT_CASES = (("EVENT_FULL", 0x2111A),)
QUAKE_CASES = (("F21548", 0x21548),)
EFFECT_TAIL_CASES = tuple(("F" + format(address, "X"), address) for address in (
    0x2185F, 0x2189A, 0x219AD, 0x21A9E, 0x21AD9, 0x21B18, 0x21B99)) + (
    ("INDEXED_PAIR", 0x2111A), ("INDEXED_ALL", 0x2111A), ("QUAKE_GROUP", 0x21527))
SPARSE_GROUPS = {"INDEXED_PAIR": (0x2111A, 0x21B18),
                 "INDEXED_ALL": (0x2111A, 0x21AD9, 0x21B18, 0x21B99),
                 "QUAKE_GROUP": (0x21527, 0x21548, 0x2185F, 0x21A9E),
                 "RADIAL_GROUP": (0x190AC, 0x219AD)}
TREASURE_CASES = (("F190AC", 0x190AC), ("RADIAL_GROUP", 0x190AC))
CASE_GROUPS = {"COUNTS_GROUP": (0x1B5F1, 0x1B653, 0x1B6B7), "EVENT_PAIR": (0x2111A, 0x211A4)}
CASE_GROUPS['EVENT_FULL'] = (0x2111A, 0x211A4, 0x21206, 0x21227, 0x212B9,
    0x2134B, 0x21364, 0x2137D, 0x21396, 0x213B7, 0x21449, 0x21462,
    0x2147B, 0x21494, 0x214AD)
ALL_CASES = (*CASES, *RECORD_CASES, *COPY_CASES, *QUERY_CASES, *CONTROL_CASES, *BATTLE_CASES, *EVENT_CASES, *LAYOUT_CASES, *QUAKE_CASES, *EFFECT_TAIL_CASES, *TREASURE_CASES)
BINDINGS = {"dword_53A45": 0x53A45, "dword_53BEB": 0x53BEB,
            "dword_53AC1": 0x53AC1, "dword_53A51": 0x53A51, "__CHK": 0x36CD7,
            "sub_375B2": 0x375B2, "sub_3453E": 0x3453E, "sub_127E0": 0x127E0,
            "sub_129EC": 0x129EC, "sub_11D40": 0x11D40, "memmove": 0x373C4,
            "sub_4E92C": 0x4E92C, "_free": 0x37416, "byte_53AF9": 0x53AF9,
            "dword_53B0F": 0x53B0F, "dword_53B13": 0x53B13,
            "aFdotherDat": 0x51A4D, "sub_111BA": 0x111BA,
            "dword_53BF7": 0x53BF7, "dword_53BFB": 0x53BFB}
BINDINGS.update({"dword_53AB1": 0x53AB1, "dword_53AB5": 0x53AB5,
                 "dword_53AB9": 0x53AB9, "dword_53ABD": 0x53ABD,
                 "dword_53AA9": 0x53AA9, "dword_53AAD": 0x53AAD,
                 "dword_53AC5": 0x53AC5, "dword_51A83": 0x51A83,
                 "dword_51A87": 0x51A87, "dword_51A8B": 0x51A8B,
                 "dword_51A8F": 0x51A8F, "dword_53A49": 0x53A49,
                 "dword_53A4D": 0x53A4D, "dword_53A55": 0x53A55,
                 "dword_53A69": 0x53A69, "sub_11CAC": 0x11CAC,
                 "sub_4E56C": 0x4E56C, "sub_4DEDA": 0x4DEDA,
                 "_sub_4E56C": 0x4E56C, "_sub_4DEDA": 0x4DEDA,
                 "sub_12E38": 0x12E38})
BINDINGS.update({"dword_53A71": 0x53A71, "dword_53A85": 0x53A85,
                 "dword_53C67": 0x53C67, "sub_4E8AF": 0x4E8AF,
                 "sub_4E8E1": 0x4E8E1})
BINDINGS.update({"dword_53C03": 0x53C03, "byte_52363": 0x52363,
                 "sub_4E53E": 0x4E53E, "_sub_4E53E": 0x4E53E,
                 "abs": 0x375E2, "sub_18C6D": 0x18C6D,
                 "sub_1F183": 0x1F183, "sub_1E739": 0x1E739,
                 "memset": 0x375C0})
BINDINGS.update({"dword_53A0C": 0x53A0C, "dword_53A2C": 0x53A2C,
                 "dword_53C57": 0x53C57, "dword_53ECC": 0x53ECC,
                 "dword_53BEF": 0x53BEF, "sub_4E9BB": 0x4E9BB,
                 "sub_4E63D": 0x4E63D, "sub_187D6": 0x187D6,
                 "sub_17D6F": 0x17D6F, "sub_4E516": 0x4E516,
                 "sub_1C916": 0x1C916, "loc_205BE": 0x205BE,
                 "sub_21227": 0x21227, "sub_213B7": 0x213B7,
                 "sub_21548": 0x21548})
BINDINGS.update({"_sub_4E9BB": 0x4E9BB, "_sub_4E63D": 0x4E63D,
                 "_sub_4E516": 0x4E516})
BINDINGS.update({"byte_51AAC": 0x51AAC, "sub_1A30B": 0x1A30B,
                 "sub_146A7": 0x146A7, "dword_53A10": 0x53A10,
                 "dword_53A14": 0x53A14, "dword_53EEC": 0x53EEC,
                 "sub_16559": 0x16559, "sub_25A96": 0x25A96,
                 "sub_17AA9": 0x17AA9, "malloc": 0x36D16,
                 "sub_1B722": 0x1B722, "sub_1BB8C": 0x1BB8C})
BINDINGS.update({"dword_53EC4": 0x53EC4, "sub_1C4CC": 0x1C4CC,
                 "sub_1C2DA": 0x1C2DA, "sub_1CAC7": 0x1CAC7,
                 "sub_1C75E": 0x1C75E, "sub_1E0DB": 0x1E0DB,
                 "sub_1E1DC": 0x1E1DC, "sub_1DF58": 0x1DF58,
                 "sub_1B750": 0x1B750, "sub_1B8E7": 0x1B8E7,
                 "sub_1CA89": 0x1CA89})
BINDINGS.update({'sub_1CD17': 0x1CD17})
BINDINGS.update({'unk_52096': 0x52096, 'unk_520A2': 0x520A2, 'unk_520AE': 0x520AE,
    'dword_53A5D': 0x53A5D, 'sub_1399C': 0x1399C, 'sub_1F558': 0x1F558,
    'sub_11EB0': 0x11EB0, '_printf': 0x36DC1, 'printf': 0x36DC1,
    'exit': 0x36DE4, 'aOutOfMemoryAtE': 0x501CF})
BINDINGS.update({'dword_53A6D': 0x53A6D, 'sub_21548': 0x21548,
    'sub_2189A': 0x2189A, 'sub_219AD': 0x219AD, 'sub_21B18': 0x21B18,
    'sub_21EB1': 0x21EB1, 'sub_11EEE': 0x11EEE, 'sub_4DB9C': 0x4DB9C,
    'sub_127A9': 0x127A9, 'sub_1C8ED': 0x1C8ED,
    '_abs': 0x375E2, '_sqrt': 0x3C6FC, 'sqrt': 0x3C6FC, '__CHP': 0x377A4, 'dbl_501F0': 0x501F0})
BINDINGS.update({'dword_53AD5': 0x53AD5, 'dword_53AD9': 0x53AD9,
    'dword_53ADD': 0x53ADD, 'dword_53AE1': 0x53AE1, 'dword_53BF3': 0x53BF3,
    'sub_4E031': 0x4E031, 'sub_1956B': 0x1956B, 'sub_15F84': 0x15F84,
    'sub_19953': 0x19953, 'sub_197E5': 0x197E5, 'sub_25B45': 0x25B45,
    'sub_16C57': 0x16C57, 'sub_196CB': 0x196CB, 'sub_12263': 0x12263,
    'sub_1B932': 0x1B932, 'funcs_1199C': 0x1199C, 'dword_53A7D': 0x53A7D})
SOURCE_DATE_EPOCH = 315532800  # DOS 可表示的 1980-01-01 UTC。
SOURCES = {
    "game": ("tools/fd2_matching_game_slices.c", tuple(macro for macro, _ in CASES)),
    "record_layout": ("tools/fd2_matching_record_layout.c", ("SET_BIT7", "SLOT_BYTE", "COPY_WORDS")),
    "game_records": ("tools/fd2_matching_game_records.c", tuple(macro for macro, _ in RECORD_CASES)),
    "game_copy": ("tools/fd2_matching_game_copy.c", tuple(macro for macro, _ in COPY_CASES)),
    "game_queries": ("tools/fd2_matching_game_queries.c", tuple(macro for macro, _ in QUERY_CASES)),
    "game_controls": ("tools/fd2_matching_game_controls.c", tuple(macro for macro, _ in CONTROL_CASES)),
    "game_battle_records": ("tools/fd2_matching_game_battle_records.c", tuple(macro for macro, _ in BATTLE_CASES)),
    "game_events": ("tools/fd2_matching_game_events.c", tuple(macro for macro, _ in EVENT_CASES)),
    "game_event_layout": ("tools/fd2_matching_game_event_layout.c", tuple(macro for macro, _ in LAYOUT_CASES)),
    "game_quake": ("tools/fd2_matching_game_quake.c", tuple(macro for macro, _ in QUAKE_CASES)),
    "game_effect_tail": ("tools/fd2_matching_game_effect_tail.c", tuple(macro for macro, _ in EFFECT_TAIL_CASES)),
    "game_treasure": ("tools/fd2_matching_game_treasure.c", tuple(macro for macro, _ in TREASURE_CASES)),
}
COSTS = {"balanced": (), "space": ("-os",), "speed": ("-ot",)}
SOURCE_INCLUDES = {'game_effect_tail': (('QUAKE.C', 'tools/fd2_matching_game_quake.c'),),
    'game_treasure': (('EFFECT.C', 'tools/fd2_matching_game_effect_tail.c'),
                     ('QUAKE.C', 'tools/fd2_matching_game_quake.c'))}
COMPILER_INPUTS = {
    "10.0a": INPUTS,
    "9.5": {
        "WCC386.EXE": "3fe098187af3ed4bccbf184b0f0d1a21a18cf976fe72c6178f19b8eea8b13e78",
        "DOS4GW.EXE": "b401506365892bd7bcb4362599279504a863f05bf37d323f732fd348bd1ef3c5",
    },
}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def compile_stage(args):
    compiler_inputs = COMPILER_INPUTS[args.compiler_version]
    for name, digest in compiler_inputs.items():
        if sha(args.compiler / name) != digest:
            raise ValueError("compiler 組件雜湊不符")
    args.output.mkdir(parents=True, exist_ok=True)
    if args.output.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄擁有權不符")
    source_path, source_cases = SOURCES[args.source_key]
    source = args.repo / source_path
    selected_cases = args.cases or source_cases or [macro for macro, _ in CASES]
    if source_cases and not set(selected_cases).issubset(source_cases):
        raise ValueError("所選來源未提供這些候選")
    cases = [(macro, address) for macro, address in ALL_CASES if macro in selected_cases]
    dos_source = args.output / "GAME.C"
    dos_source.write_bytes(source.read_text(encoding="utf-8").encode("ascii", "ignore"))
    os.utime(dos_source, (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
    includes = []
    for dos_name, source_name in SOURCE_INCLUDES.get(args.source_key, ()):
        include_source = args.repo / source_name
        include_file = args.output / dos_name
        include_file.write_bytes(include_source.read_text(encoding='utf-8').encode('ascii', 'ignore'))
        os.utime(include_file, (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
        includes.append({'dos_file': dos_name, 'source_path': source_name,
                         'source_sha256': sha(include_source), 'dos_sha256': sha(include_file)})
    lines = ["[sdl]", "output=surface", "[dosbox]", "memsize=31", "[cpu]", "core=normal",
             "cycles=max 90% limit 100000", "[mixer]", "nosound=true", "[midi]", "mpu401=none",
             "mididevice=none", "[autoexec]", f"mount c {args.output}", f"mount r {args.compiler}",
             "c:", "set PATH=R:\\;Z:\\"]
    commands = []
    trials = []
    for index, ((macro, address), cpu, cost) in enumerate(itertools.product(cases, args.cpus, args.costs)):
        stem = f"G{index:02}"
        flags = ["-mf", "-" + cpu, *COSTS[cost], "-d" + macro]
        commands.append("R:\\WCC386.EXE " + " ".join(flags) + f" -fo={stem}.OBJ GAME.C > {stem}.TXT")
        trial = {"stem": stem, "address": hex(address), "flags": flags}
        if macro in CASE_GROUPS or macro in SPARSE_GROUPS:
            trial["addresses"] = [hex(value) for value in (CASE_GROUPS | SPARSE_GROUPS)[macro]]
        trials.append(trial)
    compile_batch = args.output / "BUILD.BAT"
    compile_batch.write_text("\r\n".join(commands) + "\r\n", encoding="ascii")
    lines.extend(["call BUILD.BAT", "exit"])
    config = args.output / "RUN.CONF"
    config.write_text("\n".join(lines) + "\n", encoding="ascii")
    result = subprocess.run(["dosbox", "-conf", str(config), "-noconsole"], capture_output=True, text=True, timeout=90)
    (args.output / "runner.log").write_text(result.stdout + result.stderr, encoding="utf-8")
    if result.returncode or "Exit to error:" in result.stdout + result.stderr:
        raise ValueError("compiler 執行器失敗")
    for trial in trials:
        obj = args.output / (trial["stem"] + ".OBJ")
        text = (args.output / (trial["stem"] + ".TXT")).read_text(encoding="cp437")
        if not obj.is_file() or not re.search(r"\b0 errors\b", text) or "Error!" in text:
            raise ValueError(f"{trial['stem']}: compiler 未成功產生物件")
        trial["object_sha256"] = sha(obj)
    report = {"schema_version": 1, "source_key": args.source_key,
              "source_path": source_path, "source_sha256": sha(source),
              "compiler_version": args.compiler_version, "compiler_inputs": compiler_inputs,
              "source_date_epoch": SOURCE_DATE_EPOCH,
              "driver_sha256": sha(Path(__file__)), "trials": trials}
    if includes:
        report['source_includes'] = includes
    (args.output / "compile-report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(f"{len(trials)} 個主程式 C 候選已編譯。", flush=True)


def link_stage(args):
    from fd2_matching_pilot import checked, linked_text
    from fd2_matching_legacy_verify import OBJCONV_SHA256, text_size
    if sha(args.converter) != OBJCONV_SHA256:
        raise ValueError("物件轉換器雜湊不符")
    image = json.loads(args.evidence.read_text(encoding="utf-8"))
    original = args.original.read_bytes()
    reference = next(x for x in json.loads((args.repo / "docs/data/fd2-reference-files.json").read_text(encoding="utf-8"))["files"]
                     if x["file"] == "FD2.EXE")
    identity = {"file": "FD2.EXE", "size": len(original), "md5": hashlib.md5(original).hexdigest(), "sha256": hashlib.sha256(original).hexdigest()}
    if identity != reference or image["input"] != reference:
        raise ValueError("原檔或 IDA 版本不符")
    compile_report = json.loads((args.objects / "compile-report.json").read_text(encoding="utf-8"))
    compiler_version = compile_report.get("compiler_version", "10.0a")
    if compile_report["compiler_inputs"] != COMPILER_INPUTS[compiler_version]:
        raise ValueError("compiler 組件登記不符")
    source_key = compile_report.get("source_key", "game")
    source_path = SOURCES[source_key][0]
    if compile_report.get("source_path", source_path) != source_path:
        raise ValueError("C 來源路徑未登記")
    if compile_report["source_sha256"] != sha(args.repo / source_path):
        raise ValueError("C 來源與編譯收據不符")
    if (args.objects / "GAME.C").read_bytes() != (args.repo / source_path).read_text(encoding="utf-8").encode("ascii", "ignore"):
        raise ValueError("實際 DOS C 來源與登記來源不符")
    expected_includes = []
    for dos_name, source_name in SOURCE_INCLUDES.get(source_key, ()):
        include_source = args.repo / source_name
        expected = include_source.read_text(encoding='utf-8').encode('ascii', 'ignore')
        include_file = args.objects / dos_name
        if not include_file.is_file() or include_file.read_bytes() != expected or int(include_file.stat().st_mtime) != SOURCE_DATE_EPOCH:
            raise ValueError('實際 DOS include 與登錄來源或固定時間不符')
        expected_includes.append({'dos_file': dos_name, 'source_path': source_name,
            'source_sha256': sha(include_source), 'dos_sha256': hashlib.sha256(expected).hexdigest()})
    if compile_report.get('source_includes', []) != expected_includes:
        raise ValueError('C include 登錄與編譯收據不符')
    args.output.mkdir(parents=True, exist_ok=True)
    results = []
    for trial in compile_report["trials"]:
        obj = args.objects / (trial["stem"] + ".OBJ")
        if sha(obj) != trial["object_sha256"]:
            raise ValueError("原始 compiler 物件與收據不符")
        out = args.output / trial["stem"]
        out.mkdir(exist_ok=True)
        coff, dis = out / "candidate.cof", out / "candidate.dis"
        checked([str(args.converter), "-fcoff", str(obj), str(coff)])
        checked(["wdis", "-l=" + str(dis), str(obj)])
        publics = re.findall(r"^([0-9A-Fa-f]{4,8})\s+([_A-Za-z][_A-Za-z0-9]*):", dis.read_text(), re.M)
        symbols = [name for offset, name in publics if int(offset, 16) == 0]
        if len(symbols) != 1:
            raise ValueError("候選必須有唯一的起始函式")
        size, address = text_size(coff), int(trial["address"], 16)
        addresses = trial.get("addresses", [trial["address"]])
        macro = next((flag[2:] for flag in trial["flags"] if flag.startswith("-d")), None)
        sparse = macro in SPARSE_GROUPS
        if len(addresses) > 1:
            if macro not in SOURCES[source_key][1] or addresses != [hex(value) for value in (CASE_GROUPS | SPARSE_GROUPS).get(macro, ())]:
                raise ValueError("多函式區間未依來源登記")
        by_address = {f['inventory']['start']: f for f in image['functions']}
        functions = []
        for target in addresses:
            if target not in by_address:
                if source_key != 'game_event_layout':
                    raise ValueError('來源未登記清冊外區間')
                from fd2_matching_intervals import unowned_interval
                by_address[target] = unowned_interval(int(target, 16), image, original, args.evidence.parent)
            functions.append(by_address[target])
        if functions[0]["inventory"]["start"] != trial["address"]:
            raise ValueError("候選起始位址與函式區間不符")
        group_end = int(functions[-1]["inventory"]["end"], 16)
        if any(not address <= int(i["ida_linear_address"], 16) < group_end for fn in functions for c in fn["chunks"] for i in c["instructions"]):
            raise ValueError("候選區間沒有涵蓋所需的共用尾段")
        intervals = []
        instructions = []
        cursor = address
        for fn in functions:
            start, end = int(fn["inventory"]["start"], 16), int(fn["inventory"]["end"], 16)
            if sparse:
                cursor = start
            if start != cursor:
                raise ValueError("多函式區間必須連續且不重疊")
            part = [i for c in fn["chunks"] for i in c["instructions"] if start <= int(i["ida_linear_address"], 16) < end]
            if len(addresses) == 1 and len(part) != sum(len(c["instructions"]) for c in fn["chunks"]):
                raise ValueError("單函式候選不能略過共用尾段")
            for instruction in part:
                if int(instruction["ida_linear_address"], 16) != cursor:
                    raise ValueError("原始指令不連續")
                cursor += len(bytes.fromhex(instruction["loaded_bytes"]))
            if cursor != end or end - start != fn["inventory"]["size"]:
                raise ValueError("目標指令不涵蓋完整原始函式區間")
            intervals.append({"address": hex(start), "offset": sum(p['size'] for p in intervals) if sparse else start - address, "size": end - start})
            if source_key == 'game_event_layout':
                intervals[-1]['kind'] = fn['inventory'].get('kind', 'ida_function')
                if intervals[-1]['kind'] == 'unowned_code':
                    intervals[-1]['evidence_sha256'] = fn['evidence_sha256']
            instructions.extend(part)
        entry_offsets_equal = len(publics) == len(functions)
        if len(addresses) > 1:
            actual_entries = {name.lstrip("_").lower(): int(offset, 16) for offset, name in publics}
            entry_offsets_equal = entry_offsets_equal and all(actual_entries.get("sub_" + format(int(part["address"], 16), "x")) == part["offset"] for part in intervals)
        expected = bytearray()
        for instruction in instructions:
            offset = int(instruction["file_offset"], 16)
            raw = bytes.fromhex(instruction["file_bytes"])
            if original[offset:offset + len(raw)] != raw:
                raise ValueError("原始指令版本不符")
            expected.extend(bytes.fromhex(instruction["loaded_bytes"]))
        if len(expected) != sum(part["size"] for part in intervals):
            raise ValueError("目標指令不涵蓋完整連續區間")
        script = out / "link.ld"
        definitions = "\n".join(f"{key} = {value:#x};" for key, value in BINDINGS.items())
        sparse_layout = None
        if sparse:
            from fd2_matching_sparse import split_coff, read_sparse_pe
            try:
                converted, sparse_layout = split_coff(coff.read_bytes(), dis.read_text(), [int(a, 16) for a in addresses])
            except ValueError as error:
                if str(error) not in ('連結後完整函式彼此重疊', '不能保持跨函式短分支的原始指令寬度'):
                    raise
                results.append({**trial, 'candidate_size': size, 'original_size': len(expected),
                    'exact_interval_bytes': False, 'comparison_performed': False,
                    'linkable': False, 'layout': 'sparse_functions', 'layout_rejection': str(error),
                    'function_intervals': intervals, 'exact_entry_offsets': entry_offsets_equal,
                    'original_classification': functions[0]['inventory']['classification']})
                continue
            coff = out / 'sparse.cof'
            coff.write_bytes(converted)
            sections = ' '.join(f".m{n} {p['address']} : SUBALIGN(1) {{ *({p['section']}) }}" for n, p in enumerate(sparse_layout['fragments']))
        else:
            sections = f".text {address:#x} : SUBALIGN(1) {{ *(_TEXT) }}"
        script.write_text(definitions + f"\nSECTIONS {{ {sections} "
                          "/DISCARD/ : { *(CONST) *(CONST2) *(_DATA) *(_BSS) *(.depend) *(.reloc) } }\n", encoding="ascii")
        exe = out / "linked.exe"
        checked(["ld", "-mi386pe", "--image-base", "0", "--section-alignment", "1", "--file-alignment", "1",
                 "--no-insert-timestamp", "-T", str(script), "-e", symbols[0], "-o", str(exe), str(coff)])
        code = read_sparse_pe(exe, sparse_layout['fragments']) if sparse else linked_text(exe, address, size)
        (out / "candidate.bin").write_bytes(code)
        first = next((i for i, pair in enumerate(zip(code, expected)) if pair[0] != pair[1]), min(len(code), len(expected)))
        result = {**trial, "candidate_size": size, "original_size": len(expected), "exact_interval_bytes": code == expected and entry_offsets_equal,
                  "first_difference_offset": None if code == expected else first, "code_sha256": hashlib.sha256(code).hexdigest(),
                  "original_classification": functions[0]["inventory"]["classification"]}
        if len(addresses) > 1:
            result.update(function_intervals=intervals, exact_entry_offsets=entry_offsets_equal,
                          original_classifications=[f["inventory"]["classification"] for f in functions])
        if sparse:
            result.update(layout='sparse_functions', sparse_layout=sparse_layout,
                          sparse_transform_sha256=sha(Path(__file__).with_name('fd2_matching_sparse.py')))
        results.append(result)
    report = {"schema_version": 1, "input": reference, "source_key": source_key,
              "source_path": source_path, "source_sha256": compile_report["source_sha256"],
              "compiler_version": compiler_version, "compiler_inputs": compile_report["compiler_inputs"],
              "compile_driver_sha256": compile_report["driver_sha256"],
              "driver_sha256": sha(Path(__file__)), "evidence_sha256": sha(args.evidence), "trials": results,
              "matched_addresses": sorted({part['address'] for r in results if r['exact_interval_bytes']
                    for part in r.get('function_intervals', [{'address': r['address']}]) if part.get('kind', 'ida_function') == 'ida_function'})}
    if expected_includes:
        report['source_includes'] = expected_includes
    (args.output / "report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"matched_addresses": report["matched_addresses"], "trials": len(results)}), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("stage", choices=("compile", "link"))
    parser.add_argument("--repo", type=Path, default=Path("/repo"))
    parser.add_argument("--compiler", type=Path)
    parser.add_argument("--compiler-version", choices=COMPILER_INPUTS, default="10.0a")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--objects", type=Path)
    parser.add_argument("--evidence", type=Path)
    parser.add_argument("--original", type=Path, default=Path("/input/FD2.EXE"))
    parser.add_argument("--converter", type=Path, default=Path("/work/objconv"))
    parser.add_argument("--source-key", choices=SOURCES, default="game")
    parser.add_argument("--cases", choices=[macro for macro, _ in ALL_CASES], nargs="+")
    parser.add_argument("--cpus", choices=("3s", "4s", "5s"), nargs="+", default=("3s", "4s", "5s"))
    parser.add_argument("--costs", choices=COSTS, nargs="+", default=tuple(COSTS))
    args = parser.parse_args()
    if args.compiler is None:
        args.compiler = Path("/wc10a" if args.compiler_version == "10.0a" else "/wc95")
    if not Path("/.dockerenv").exists():
        raise SystemExit("本工具只在 Docker 執行")
    compile_stage(args) if args.stage == "compile" else link_stage(args)


if __name__ == "__main__":
    main()
