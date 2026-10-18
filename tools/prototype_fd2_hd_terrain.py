#!/usr/bin/env python3
"""核對原生地形圖庫，產生不接正式路徑的首章 HD 草地排版原型。

只供 Docker 內執行。原型不含人物、動畫、LUT 或遊戲狀態，不能當作 GPU／對拍收據。
"""

from __future__ import annotations

import argparse
import hashlib
import json
from collections import defaultdict
from pathlib import Path

from PIL import Image, ImageDraw


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def audit(maps: Path, banks: Path, reference: Path) -> dict:
    expected = next(item for item in json.loads(reference.read_text())["files"]
                    if item["file"] == "FDSHAP.DAT")
    rows = []
    inputs = []

    def bind(path: Path, role: str) -> None:
        inputs.append({"role": role, "file": str(path), "sha256": sha(path)})

    bind(reference, "reference_manifest")
    for map_id in range(33):
        directory = banks / f"map_{map_id:02d}"
        metadata = directory / "bank.json"
        document = json.loads(metadata.read_text())
        for key in ("file", "size", "md5", "sha256"):
            if document["source"][key] != expected[key]:
                raise ValueError(f"map{map_id}: FDSHAP 來源指紋不符")
        if (document["map_index"] != map_id or document["image_resource"] != map_id * 2
                or document["control_resource"] != map_id * 2 + 1):
            raise ValueError(f"map{map_id}: 原生資源身分不符")
        map_path = maps / f"map{map_id}" / "map.json"
        atlas_path = map_path.with_name("tileset.png")
        mapping = json.loads(map_path.read_text())
        atlas = Image.open(atlas_path).convert("RGBA")
        count = len(document["sprites"])
        if (mapping["tileW"], mapping["tileH"], mapping["cols"]) != (24, 24, 16):
            raise ValueError(f"map{map_id}: 正規化圖塊契約不符")
        if atlas.size != (384, ((count + 15) // 16) * 24):
            raise ValueError(f"map{map_id}: 圖集幾何不符")
        if mapping["native_terrain_control"] != document["controls"]:
            raise ValueError(f"map{map_id}: 地形控制表不符")
        bind(metadata, "native_bank")
        bind(map_path, "normalized_map")
        bind(atlas_path, "normalized_atlas")
        seen = set()
        palette_colors = defaultdict(set)
        alpha_differences = 0
        different_frames = []
        remap_pixels = opaque_pixels = 0
        for entry in document["sprites"]:
            index = entry["index"]
            if index in seen or not 0 <= index < count:
                raise ValueError(f"map{map_id}: 重複或越界的原生幀")
            seen.add(index)
            layers = []
            for key, filename in (("frame", "frame.png"), ("mask", "mask.png"),
                                  ("remap_mask", "remap_mask.png")):
                if entry[key] != f"tile_{index:04d}/{filename}":
                    raise ValueError(f"map{map_id}: 非預期的分離圖層路徑")
                path = directory / entry[key]
                layer = Image.open(path)
                if layer.size != (24, 24):
                    raise ValueError(f"map{map_id}: 原生圖層尺寸不符")
                if key == "frame" and layer.mode != "P":
                    raise ValueError(f"map{map_id}: 原生幀必須保留調色盤索引")
                if key != "frame":
                    layer = layer.convert("L")
                    if not set(layer.getdata()) <= {0, 255}:
                        raise ValueError(f"map{map_id}: 遮罩不是二元值")
                layers.append(layer)
                bind(path, key)
            raw, mask, remap = layers
            tile = atlas.crop(((index % 16) * 24, (index // 16) * 24,
                               (index % 16 + 1) * 24, (index // 16 + 1) * 24))
            differences = 0
            for value, visible, remapped, rgba in zip(raw.getdata(), mask.getdata(),
                                                     remap.getdata(), tile.getdata()):
                differences += bool(visible) != bool(rgba[3])
                remap_pixels += bool(remapped)
                opaque_pixels += bool(visible)
                if visible and rgba[3]:
                    palette_colors[value].add(rgba[:3])
            alpha_differences += differences
            if differences:
                different_frames.append({"frame": index, "pixels": differences})
        if len(seen) != count:
            raise ValueError(f"map{map_id}: 原生幀缺漏")
        conflicts = sorted(index for index, colors in palette_colors.items() if len(colors) > 1)
        rows.append({"map_id": map_id, "frame_count": count,
                     "control_records": len(document["controls"]) // 4,
                     "opaque_pixels": opaque_pixels, "remap_pixels": remap_pixels,
                     "normalized_alpha_difference_pixels": alpha_differences,
                     "normalized_alpha_difference_frames": different_frames,
                     "normalized_palette_conflict_indices": conflicts,
                     "normalized_native_correspondence":
                         "NOT_EQUIVALENT" if alpha_differences or conflicts else "CONSISTENT_SAMPLED_LAYERS"})
    return {"source_metadata_binding": expected, "original_dat_rehashed": False,
            "scope": "分離圖層與正規化圖集的資料比對；不判定原版執行結果或差異根因",
            "maps": rows, "inputs": inputs}


def prototype(maps: Path, modern: Path, material: Path) -> tuple[Image.Image, dict]:
    mapping = json.loads((maps / "map0/map.json").read_text())
    original = Image.open(maps / "map0/tileset.png").convert("RGBA")
    atlas = Image.open(modern / "map0-tileset-style-a.png").convert("RGBA")
    if atlas.size != original.size:
        raise ValueError("首章現代圖集尺寸不符")
    texture = Image.open(material).convert("RGBA")
    if texture.width != texture.height or texture.width < 144:
        raise ValueError("草地材質須為至少144像素的正方形")
    if texture.getextrema()[3] != (255, 255):
        raise ValueError("草地材質須完全不透明")
    texture = texture.resize((144, 144), Image.Resampling.LANCZOS)
    # map0 frame52 是本次人工核對的無物件草地。它的四個來源顏色只用於
    # 可丟棄美術遮罩，不把顏色分類寫入地形語意或正式規則。
    region = (96, 72, 120, 96)
    grass_colors = set(original.crop(region).getdata())
    mean = [sum(pixel[channel] for pixel in texture.getdata()) / (144 * 144)
            for channel in range(3)]
    # 選取含最多 frame52 的6×4資料窗格，避免固定裁切完全沒顯示待比較材質。
    width = mapping["w"]
    origin_x, origin_y = max(
        ((x, y) for y in range(mapping["h"] - 3) for x in range(width - 5)),
        key=lambda origin: sum(mapping["tiles"][(origin[1] + row) * width + origin[0] + col] == 52
                               for row in range(4) for col in range(6)))
    before = Image.new("RGBA", (864, 576), (0, 0, 0, 255))
    after = before.copy()
    replacements = 0
    for row in range(4):
        for column in range(6):
            # 固定章資料的排版樣本，不冒稱即時鏡頭或一般玩家狀態。
            index = mapping["tiles"][(row + origin_y) * width + column + origin_x]
            tile = atlas.crop(((index % 16) * 24, (index // 16) * 24,
                               (index % 16 + 1) * 24, (index // 16 + 1) * 24))
            native = original.crop(((index % 16) * 24, (index // 16) * 24,
                                    (index % 16 + 1) * 24, (index // 16 + 1) * 24))
            mask = Image.new("L", (24, 24))
            mask.putdata([255 if pixel in grass_colors else 0 for pixel in native.getdata()])
            mask = mask.resize((144, 144), Image.Resampling.NEAREST)
            tile = tile.resize((144, 144), Image.Resampling.NEAREST)
            detailed = tile.copy()
            # 保留來源低頻明暗與全部非遮罩像素，只加入新母稿的細部明暗。
            detailed.putdata([
                tuple(max(0, min(255, round(base[c] * max(0.75, min(1.25, art[c] / mean[c])))))
                      for c in range(3)) + (base[3],) if present else base
                for base, art, present in zip(tile.getdata(), texture.getdata(), mask.getdata())])
            position = (column * 144, row * 144)
            before.alpha_composite(tile, position)
            after.alpha_composite(detailed, position)
            replacements += mask.getbbox() is not None
    # 兩個完整畫面並排會縮到無法比較材質，固定取同一局部以一比一顯示。
    comparison = Image.new("RGB", (1728, 616), (24, 24, 30))
    comparison.paste(before, (0, 40))
    comparison.paste(after, (864, 40))
    draw = ImageDraw.Draw(comparison)
    draw.text((16, 14), "A: existing 24px terrain x6", fill="white")
    draw.text((880, 14), "B: painted grass prototype / layout only", fill="white")
    # 接縫以數據揭露，不因生成提示寫 seamless 就視為成立。
    pixels = texture.load()
    horizontal = sum(abs(pixels[0, y][c] - pixels[143, y][c])
                     for y in range(144) for c in range(3)) / (144 * 3)
    vertical = sum(abs(pixels[x, 0][c] - pixels[x, 143][c])
                   for x in range(144) for c in range(3)) / (144 * 3)
    return comparison, {"scope": "首章固定資料、同一排版局部，只比較草地材質",
                        "normal_player_path": False, "gpu_capture": False,
                        "runtime_integrated": False, "map_origin_cells": [origin_x, origin_y],
                        "mask_color_source_frame": 52, "detailed_cells_in_view": replacements,
                        "mask_rgba_colors": sorted(grass_colors),
                        "method": "source-color-mask with bounded multiplicative painted detail; prototype only",
                        "native_size": 24, "prototype_size": 144,
                        "material_sha256": sha(material),
                        "modern_atlas_sha256": sha(modern / "map0-tileset-style-a.png"),
                        "opposite_edge_rgb_mean_absolute_difference":
                            {"horizontal": horizontal, "vertical": vertical},
                        "seamless_verified": False,
                        "limitations": ["只有無物件草地原型，其他地形仍為24像素來源",
                                        "材質接縫、動畫、LUT、前景與正式GPU尚未驗收",
                                        "不從合成圖擷取人物或地形幾何"]}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--bank-root", type=Path, required=True)
    parser.add_argument("--reference", type=Path, required=True)
    parser.add_argument("--modern-root", type=Path, required=True)
    parser.add_argument("--material", type=Path, required=True)
    parser.add_argument("--output-prefix", type=Path, required=True)
    args = parser.parse_args()
    report_path = args.output_prefix.with_suffix(".json")
    image_path = args.output_prefix.with_suffix(".png")
    if not report_path.parent.is_dir() or report_path.exists() or image_path.exists():
        raise ValueError("輸出父目錄須已存在，且不得覆寫既有收據")
    report = audit(args.maps_root, args.bank_root, args.reference)
    first = report["maps"][0]
    if first["normalized_alpha_difference_pixels"] or first["normalized_palette_conflict_indices"]:
        raise ValueError("首章來源不具備本原型需要的圖層對應")
    image, preview = prototype(args.maps_root, args.modern_root, args.material)
    report["prototype"] = preview
    report["tool_sha256"] = sha(Path(__file__))
    report["pillow_version"] = Image.__version__
    report["status"] = "PROTOTYPE_ONLY"
    image.save(image_path)
    report["preview_sha256"] = sha(image_path)
    report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": report["status"], "map_count": len(report["maps"]),
                      "non_equivalent_normalized_atlases":
                          [row["map_id"] for row in report["maps"]
                           if row["normalized_native_correspondence"] == "NOT_EQUIVALENT"],
                      "report": str(report_path), "preview": str(image_path)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
