"""匯出 #166 物理演出 caller 與雙背景捲動，保留原始定位及證據分級。

入口：docs/data/ida/fd2_physical_background_selection_20261004.json。
只在合法授權的 IDA Pro 9.4 Docker 內對一次性資料庫執行。
"""
import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_funcs
import ida_nalt
import idaapi
import idautils
import idc

ida_auto.auto_wait()
with open(os.environ.get("FD2_IDA_INPUT", "/input/FD2.EXE"), "rb") as source:
    raw = source.read()
assert len(raw) == 357074
assert hashlib.sha256(raw).hexdigest() == "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
assert ida_nalt.retrieve_input_file_md5().hex() == hashlib.md5(raw).hexdigest()
hud_probe = os.environ.get("FD2_IDA_HUD_REDRAW") == "1"
canonical_path = ("docs/data/ida/fd2_hud_redraw_20261006.json" if hud_probe else
                  "docs/data/ida/fd2_physical_background_selection_20261004.json")
with open("/src/" + canonical_path, encoding="utf-8") as source:
    canonical = json.load(source)
result = {
    "schema_version": 1,
    "input": "FD2.EXE", "size": len(raw), "md5": hashlib.md5(raw).hexdigest(),
    "sha256": hashlib.sha256(raw).hexdigest(), "tool": "IDA Pro " + idaapi.get_kernel_version(),
    "address_space": "IDA LE linear", "functions": [],
}


def annotations_at(ea):
    annotations = []
    claims = (canonical.get("ida_claims", []) if hud_probe else
              canonical.get("physical_scroll_evidence", {}).get("claims", []) +
                  canonical.get("physical_tail_evidence", {}).get("claims", []) +
                  canonical.get("physical_sound_evidence", {}).get("claims", []) +
                  canonical.get("physical_owner_return_dac_evidence", {}).get("claims", []))
    for claim in claims:
        assert claim["level"] in ("已證實", "強推論", "假說", "未知")
        for location in claim["original"]:
            bounds = [int(part, 16) for part in location.split("..")]
            if bounds[0] <= ea <= bounds[-1]:
                annotations.append({"semantic": claim["semantic"], "level": claim["level"],
                                    "source": [canonical_path] + claim["source"]})
                break
    return annotations


# 音效補證只匯出正常tail及既有wrapper，不重做場景RE或硬體driver。
sound_probe = os.environ.get("FD2_IDA_PHYSICAL_SOUND") == "1"
owner_dac_probe = os.environ.get("FD2_IDA_PHYSICAL_OWNER_DAC") == "1"
assert sum((sound_probe, owner_dac_probe, hud_probe)) <= 1
targets = (0x28A6C, 0x25A96) if sound_probe else (0x11EB0, 0x29164, 0x2939D, 0x29C90, 0x29DED, 0x2BC9A)
sound_ranges = {0x28A6C: (0x29050, 0x29117), 0x25A96: (0x25A96, 0x25B45)}
if owner_dac_probe:
    targets = (0x28A6C, 0x1F882, 0x1F525, 0x11D40)
    result["scope"] = {"kind": "physical-owner-DAC-176",
                       "ranges": {"0x28a6c": ["0x29090", "0x29117"]},
                       "limits": "只複核已知全DAC ramp與正常caller；不解硬體時序或原版配置器。"}
if sound_probe:
    result["scope"] = {"kind": "physical-sound-owner", "ranges": {
        hex(target): [hex(lo), hex(hi)] for target, (lo, hi) in sound_ranges.items()}}
if hud_probe:
    targets = (0x1CFF0, 0x1DB65, 0x196CB)
    result["scope"] = {"kind": "hud-redraw-callers-41",
                       "limits": "只補既有caller的分支與收尾順序，不重做HUD或物理演出RE。"}
for target in targets:
    fn = ida_funcs.get_func(target)
    assert fn is not None and fn.start_ea == target
    rows = []
    for ea in idautils.FuncItems(target):
        if sound_probe and not (sound_ranges[target][0] <= ea < sound_ranges[target][1]):
            continue
        if owner_dac_probe and target == 0x28A6C and not (0x29090 <= ea < 0x29117):
            continue
        if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):
            continue
        annotations = annotations_at(ea)
        level = "已證實" if annotations and all(a["level"] == "已證實" for a in annotations) else "未知"
        rows.append({
            "address": hex(ea), "bytes": ida_bytes.get_bytes(ea, idc.get_item_size(ea)).hex(),
            "original": idc.generate_disasm_line(ea, 0), "semantic": "", "level": level,
            "warning": "" if level == "已證實" else "未審查語意；原始指令僅作定位",
            "source": ["tools/ida_probe_physical_presentation.py"], "semantic_annotations": annotations,
        })
    result["functions"].append({
        "address": hex(target), "end": hex(fn.end_ea), "original_name": idc.get_func_name(target),
        "instructions": rows, "semantic": "", "level": "未知",
        "warning": "只閉合已審查的 caller-specific claims", "source": ["tools/ida_probe_physical_presentation.py"],
        "callers": [{"address": hex(x.frm), "original": idc.generate_disasm_line(x.frm, 0),
                     "bytes": ida_bytes.get_bytes(x.frm, idc.get_item_size(x.frm)).hex(),
                     "level": "未知", "warning": "呼叫定位；語意另見 canonical claims",
                     "source": ["tools/ida_probe_physical_presentation.py"]}
                    for x in idautils.XrefsTo(target) if x.iscode and
                    (not sound_probe or target != 0x25A96 or 0x28A6C <= x.frm < 0x29C90)],
    })
result["tables"] = []
for ea, size in (() if owner_dac_probe or hud_probe else ((0x5255F, 24), (0x52577, 24), (0x525D6, 6))):
    annotations = annotations_at(ea)
    confirmed = bool(annotations) and all(a["level"] == "已證實" for a in annotations)
    result["tables"].append({
        "address": hex(ea), "original_name": idc.get_name(ea),
        "bytes": ida_bytes.get_bytes(ea, size).hex(),
        "semantic_annotations": annotations,
        "level": "已證實" if confirmed else "未知",
        "warning": "" if confirmed else "未審查資料表語意",
        "source": ["tools/ida_probe_physical_presentation.py"],
    })
with open(os.environ["FD2_IDA_OUTPUT"], "w", encoding="utf-8") as output:
    json.dump(result, output, ensure_ascii=False, indent=2)
    output.write("\n")
idc.qexit(0)
