#!/usr/bin/env python3
"""在 Docker 核對分離 FIGANI 清冊；只產生盤點與原版參照板。"""

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import struct

from PIL import Image, ImageDraw, __version__ as pillow_version


REPO = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def require(condition, message):
    if not condition:
        raise ValueError(message)


def audit(pack, catalog, focus):
    manifest_path = pack / "manifest.json"
    manifest = json.loads(manifest_path.read_text())
    reference = json.loads((REPO / "docs/data/fd2-reference-files.json").read_text())
    inputs = []
    for name in ["FD2.EXE", "FIGANI.DAT"]:
        pinned = next(x for x in reference["files"] if x["file"] == name)
        source = next(x for x in manifest["source_set"] if x["file"] == name)
        require(source == pinned, f"來源指紋不符：{name}")
        inputs.append(pinned)
    entries = [x for x in manifest["assets"] if x.get("source_file") == "FIGANI.DAT"]
    assets = {x["asset_id"]: x for x in entries}
    require(len(assets) == len(entries), "FIGANI asset_id 重複")
    by_path = {x["path"]: x["asset_id"] for x in entries}
    require(len(by_path) == len(entries), "FIGANI 資產路徑重複")
    resources = [x for x in manifest["source_resources"] if x["source_file"] == "FIGANI.DAT"]
    resources.sort(key=lambda x: x["source_resource"])
    require([x["source_resource"] for x in resources] == list(range(len(resources))), "資源索引不連續")
    require(0 <= focus * 3 and focus * 3 + 2 < len(resources), "焦點 selector 超出圖庫")
    verified = set()
    reference_aliases = 0

    def asset(ident):
        require(ident in assets, f"資產未登錄：{ident}")
        item = assets[ident]
        relative = Path(item["path"])
        require(not relative.is_absolute() and ".." not in relative.parts, f"資產路徑越界：{ident}")
        # 既有分離包用符號連結指向同雜湊來源；核對每份內容，不把 resolve
        # 後不在 pack 內誤判為壞包。manifest 的絕對路徑與上跳仍拒絕。
        path = pack / relative
        if ident not in verified:
            require(sha(path) == item["sha256"], f"資產雜湊不符：{ident}")
            verified.add(ident)
        return path

    result = []
    for resource in resources:
        rid = resource["source_resource"]
        raw_path = asset(resource["raw_asset_id"])
        raw = raw_path.read_bytes()
        require(len(raw) == resource["raw_bytes"] and sha(raw_path) == resource["raw_sha256"], f"raw 指紋不符：{rid}")
        for ident in resource["output_asset_ids"]:
            asset(ident)
        status_id = f"metadata/animations/figani_{rid:03d}/resource.json"
        status_path = asset(status_id)
        status = json.loads(status_path.read_text())
        require(status["source"] == {"file": "FIGANI.DAT", "resource": rid}, f"狀態來源不符：{rid}")
        require(status["raw_size"] == len(raw), f"狀態大小不符：{rid}")
        row = {"resource": rid, "status": status["status"], "reason_code": status["reason_code"],
               "raw_sha256": sha(raw_path), "status_sha256": sha(status_path), "frames": []}
        meta_id = f"metadata/animations/figani_{rid:03d}/animation.json"
        if status["status"] == "decoded":
            path = asset(meta_id)
            doc = json.loads(path.read_text())
            require(doc["source"] == status["source"] and doc["animation_id"] == f"animation/figani_{rid:03d}", f"動畫來源不符：{rid}")
            require(doc["schema_version"] == 1 and doc["kind"] == "animation", f"動畫格式不符：{rid}")
            frames = doc["frames"]
            require(len(raw) >= 8 + 4 * len(frames) and len(frames) == raw[0] > 0, f"影格數不符：{rid}")
            require(doc["native_header"] == {"byte_1": raw[1], "byte_2": raw[2], "byte_4": raw[4]}, f"原始標頭不符：{rid}")
            row.update(metadata_sha256=sha(path), native_header=doc["native_header"])
            for index, frame in enumerate(frames):
                require(frame["frame_id"] == f"frame/{index:03d}", f"影格索引不符：{rid}/{index}")
                offset = struct.unpack_from("<I", raw, 8 + 4 * index)[0]
                end = struct.unpack_from("<I", raw, 12 + 4 * index)[0] if index + 1 < len(frames) else len(raw)
                require(8 + 4 * len(frames) <= offset and offset + 13 <= end <= len(raw), f"描述子越界：{rid}/{index}")
                x, y = struct.unpack_from("<hh", raw, offset)
                width, height = struct.unpack_from("<HH", raw, offset + 9)
                expected = {"x": x, "y": y, "width": width, "height": height,
                            "delay_native": raw[offset + 6], "raw_byte_4": raw[offset + 4],
                            "raw_byte_5": raw[offset + 5], "raw_byte_7": raw[offset + 7]}
                require(all(frame[k] == v for k, v in expected.items()), f"描述子欄位不符：{rid}/{index}")
                for key, filename_key, mode in [("asset_id", "path", "P"), ("mask_asset_id", "mask_path", "L")]:
                    filename = frame[filename_key]
                    require(Path(filename).name == filename, f"影格路徑越界：{rid}/{index}")
                    relative = str((path.parent / filename).relative_to(pack))
                    require(relative in by_path, f"影格未登錄：{rid}/{index}")
                    ident = by_path[relative]
                    require(ident in resource["output_asset_ids"], f"影格不屬於本資源：{rid}/{index}")
                    image_path = asset(ident)
                    # 正式 loader 以 path/mask_path 定位。舊 exporter 的 frame_NNN
                    # asset_id 與 manifest 的檔名 ID 不一致，保留計數，不改原包。
                    reference_aliases += int(frame[key] != ident)
                    with Image.open(image_path) as picture:
                        require(picture.size == (width, height) and picture.mode == mode, f"PNG 形狀或模式不符：{rid}/{index}")
                row["frames"].append({"frame": index, **expected,
                                      "pixels_sha256": sha(path.parent / frame["path"]),
                                      "mask_sha256": sha(path.parent / frame["mask_path"])})
        else:
            require(meta_id not in assets, f"非動畫資源帶有動畫文件：{rid}")
            if status["status"] == "empty_header_zero":
                require(len(raw) >= 2 and raw[:2] == b"\0\0", f"空標頭不符：{rid}")
            elif status["reason_code"] == "empty_resource":
                require(not raw, f"空資源不符：{rid}")
            else:
                raise ValueError(f"未審查資源狀態：{rid}/{status['status']}")
        result.append(row)
    require(verified == set(assets), "manifest 有未歸屬於資源的 FIGANI 資產")
    # 以既有 LLLLLL 容器契約重建目錄與全部唯讀 raw entry 的串流，直接
    # 對固定原版檔案雜湊；不只相信 manifest 自報的 source_set。
    offsets = []
    cursor = 6 + 4 * len(resources)
    for resource in resources:
        offsets.append(cursor)
        cursor += resource["raw_bytes"]
    header = b"LLLLLL" + b"".join(struct.pack("<I", x) for x in offsets)
    digest, md5 = hashlib.sha256(header), hashlib.md5(header)
    for resource in resources:
        raw = asset(resource["raw_asset_id"]).read_bytes()
        digest.update(raw)
        md5.update(raw)
    pinned = next(x for x in inputs if x["file"] == "FIGANI.DAT")
    require(cursor == pinned["size"] and digest.hexdigest() == pinned["sha256"] and
            md5.hexdigest() == pinned["md5"], "全部 raw entry 重組後不等於固定 FIGANI.DAT")
    theme = json.loads(catalog.read_text())
    registered = [x["asset_id"] for x in theme["assets"] if "battle_animation" in x.get("role", "")]
    counts = dict(Counter(row["status"] for row in result))
    return {"schema_version": 1, "kind": "fd2_battle_animation_hd_inventory",
            "status": "PASS-SOURCE-INVENTORY", "scope": "FIGANI 分離資產、原始描述子與清冊；不代表玩家路徑或 HD 完成",
            "source_inputs": inputs, "pack_manifest_sha256": sha(manifest_path),
            "archive_reconstruction": {"contract": "LLLLLL + little-endian u32 offsets + raw entries", "size": cursor,
                                       "sha256": digest.hexdigest(), "md5": md5.hexdigest(), "matches_pinned_original": True},
            "catalog_sha256": sha(catalog), "tool_sha256": sha(Path(__file__)),
            "tools": {"pillow": pillow_version}, "verified_manifest_assets": len(verified),
            "descriptor_asset_id_aliases": reference_aliases,
            "frame_resolution": "依正式 loader 的 path/mask_path；檔案必須同時列在 manifest 及本資源 output_asset_ids，逐檔 SHA-256 相符",
            "resource_count": len(result), "resource_status_counts": counts,
            "decoded_frame_count": sum(len(x["frames"]) for x in result),
            "registered_hd_animation_assets": registered, "full_battle_animation_hd_complete": False,
            "focus": {"battle_fig": focus, "idle_resource": focus * 3, "attack_resource": focus * 3 + 1,
                      "third_resource_status": result[focus * 3 + 2]["status"]}, "resources": result}


def reference_sheet(pack, focus, path):
    frames = []
    for rid in [focus * 3, focus * 3 + 1]:
        root = pack / "animations" / f"FIGANI_{rid:03d}"
        doc = json.loads((root / "animation.json").read_text())
        frames += [(rid, i, root, f) for i, f in enumerate(doc["frames"])]
    sheet = Image.new("RGB", (1600, ((len(frames) + 3) // 4) * 250), (42, 47, 55))
    draw = ImageDraw.Draw(sheet)
    for number, (rid, index, root, frame) in enumerate(frames):
        with Image.open(root / frame["path"]) as pixels, Image.open(root / frame["mask_path"]) as mask:
            picture = pixels.convert("RGBA")
            picture.putalpha(mask.convert("L"))
        canvas = Image.new("RGBA", (320, 200))
        canvas.alpha_composite(picture, (frame["x"], frame["y"]))
        canvas = canvas.resize((384, 240), Image.Resampling.NEAREST)
        x, y = (number % 4) * 400, (number // 4) * 250
        sheet.paste(canvas, (x, y), canvas)
        draw.text((x + 5, y + 5), f"FIGANI_{rid:03d} frame {index} xy={frame['x']},{frame['y']} delay={frame['delay_native']}", fill="white")
    sheet.save(path)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--pack", required=True, type=Path)
    parser.add_argument("--catalog", type=Path, default=REPO / "remake/assets/themes/modern/catalog.json")
    parser.add_argument("--focus", type=int, default=30)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--reference-sheet", type=Path)
    args = parser.parse_args()
    report = audit(args.pack, args.catalog, args.focus)
    if args.reference_sheet:
        reference_sheet(args.pack, args.focus, args.reference_sheet)
        report["reference_sheet"] = {"file": args.reference_sheet.name, "sha256": sha(args.reference_sheet), "evidence": "decoded-assets-only"}
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({k: report[k] for k in ["status", "resource_count", "resource_status_counts", "decoded_frame_count", "verified_manifest_assets", "registered_hd_animation_assets"]}, ensure_ascii=False))


if __name__ == "__main__":
    main()
