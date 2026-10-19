#!/usr/bin/env python3
"""將整棟手繪母稿依明示UV校準切回原生格子，保存原生遮罩；僅製圖候選。"""

import argparse
import hashlib
import json
import os
import shutil
import tempfile
from pathlib import Path

from PIL import Image, ImageDraw

from verify_fd2_terrain_sources import check_binding, sha, verify


def knots(values, extent):
    if values[0] != 0 or values[-1] != extent or any(a >= b for a, b in zip(values, values[1:])):
        raise ValueError("UV錨點必須嚴格遞增並覆蓋完整畫布")


def fit(master, recipe):
    sx, sy = recipe["painted_x_anchors"], recipe["painted_y_anchors"]
    tx, ty = recipe["native_x_anchors"], recipe["native_y_anchors"]
    if len(sx) != len(tx) or len(sy) != len(ty):
        raise ValueError("UV來源及目的錨點數不同")
    width, height = recipe["native_reference_size"]
    scale = recipe["scale"]
    for values, extent in [(sx, master.width), (sy, master.height), (tx, width), (ty, height)]:
        knots(values, extent)
    mesh = []
    for row in range(len(sy) - 1):
        for col in range(len(sx) - 1):
            destination = (tx[col] * scale, ty[row] * scale, tx[col + 1] * scale, ty[row + 1] * scale)
            source = (sx[col], sy[row], sx[col], sy[row + 1], sx[col + 1], sy[row + 1], sx[col + 1], sy[row])
            mesh.append((destination, source))
    return master.transform((width * scale, height * scale), Image.Transform.MESH, mesh, Image.Resampling.BICUBIC)


def calibrate_repeat_edges(canvas, obj, policies, scale):
    if not policies:
        return []
    if not isinstance(policies, dict) or len(policies) != 1:
        raise ValueError("本階段只接受一個重複中段")
    name, policy = next(iter(policies.items()))
    if not isinstance(policy, dict) or set(policy) != {"axis", "blend_rows"} or policy["axis"] != "y":
        raise ValueError("重複中段只接受垂直交界校準")
    size = 24 * scale
    rows = policy["blend_rows"]
    if type(rows) is not int or not 1 <= rows <= size // 4:
        raise ValueError("交界校準列數越界")
    selected = [cell for cell in obj["cells"] if str(cell["frame"]) == name]
    if len(selected) != 1:
        raise ValueError("缺少唯一的重複中段")
    cell = selected[0]
    neighbours = [next((c for c in obj["cells"] if c["x"] == cell["x"] and c["y"] == y), None)
                  for y in (cell["y"]-1, cell["y"]+1)]
    if any(c is None for c in neighbours) or len({c["frame"] for c in [cell]+neighbours}) != 3:
        raise ValueError("重複中段需要上下兩張獨立的相鄰幀")
    x0, y0 = cell["x"]*size, cell["y"]*size
    if canvas.crop((x0,y0-size,x0+size,y0+size*2)).getextrema()[3] != (255,255):
        raise ValueError("交界校準只接受全opaque製圖")
    original = canvas.copy()
    before = hashlib.sha256(original.tobytes()).hexdigest()
    changed = 0
    for x in range(x0,x0+size):
        top, bottom = original.getpixel((x,y0)), original.getpixel((x,y0+size-1))
        target = tuple(round((a+b)/2) for a,b in zip(top[:3],bottom[:3]))
        for offset in range(rows):
            weight = (rows-offset)/rows
            for y in (y0+offset,y0-1-offset,y0+size-1-offset,y0+size+offset):
                old = original.getpixel((x,y))
                value = tuple(round(a*(1-weight)+b*weight) for a,b in zip(old[:3],target))+(255,)
                changed += value != old
                canvas.putpixel((x,y),value)
    for x in range(x0,x0+size):
        if len({canvas.getpixel((x,y)) for y in (y0-1,y0,y0+size-1,y0+size)}) != 1:
            raise ValueError("重複中段交界仍不相同")
    return [{"frame":cell["frame"],"affected_frames":[c["frame"] for c in neighbours[:1]+[cell]+neighbours[1:]],
             "axis":"y","blend_rows":rows,"changed_pixels":changed,"repeat_and_neighbour_edges_equal":True,
             "before_canvas_rgba_sha256":before,"after_canvas_rgba_sha256":hashlib.sha256(canvas.tobytes()).hexdigest(),
             "rgb_source":"painted master only; no original RGB mixed"}]


def build(pack, maps, sources, recipe_path, master_path, out):
    if os.path.lexists(out) or not out.parent.is_dir():
        raise ValueError("輸出須為既有父目錄下的新目錄")
    source_report = verify(pack, maps, sources)
    manifest = json.loads((sources / "manifest.json").read_text())
    recipe = json.loads(recipe_path.read_text())
    if (recipe["schema_version"], recipe["kind"], recipe["status"], recipe["scale"], recipe["runtime_integrated"]) != (
            1, "hd_terrain_object_art_recipe", "prototype", 6, False):
        raise ValueError("只接受明示六倍製圖原型，不接受正式執行期宣稱")
    master = Image.open(master_path).convert("RGBA")
    if list(master.size) != recipe["painted_master_size"] or master.getextrema()[3] != (255, 255):
        raise ValueError("母稿尺寸或不透明契約不符")
    objects = [obj for obj in manifest["objects"] if obj["id"] == recipe["object"]]
    if len(objects) != 1:
        raise ValueError("缺少唯一原生物件來源")
    obj = objects[0]
    if [obj["width"] * 24, obj["height"] * 24] != recipe["native_reference_size"]:
        raise ValueError("原生物件尺寸不符")
    if {(cell["x"], cell["y"]) for cell in obj["cells"]} != {
            (x, y) for y in range(obj["height"]) for x in range(obj["width"])}:
        raise ValueError("物件來源格子缺漏或重排")
    source = manifest["maps"][obj["map"]]
    bank = json.loads(check_binding(pack, source["bank"]).read_text())
    indices = Image.open(sources / f"map{obj['map']:02d}-indices.png")
    opaque = Image.open(sources / f"map{obj['map']:02d}-mask.png")
    remap = Image.open(sources / f"map{obj['map']:02d}-remap.png")
    canvas = fit(master, recipe)
    repeat_calibration = calibrate_repeat_edges(canvas, obj, recipe.get("repeat_edges", {}), recipe["scale"])
    entries = []
    pictures = {}
    foreground_sources = []
    scale = recipe["scale"]
    size = 24 * scale
    for cell in obj["cells"]:
        if bank["controls"][(cell["tile_word"] & 1023) * 4] & 0x1c:
            raise ValueError("本物件候選未授權略過原生動畫幀")
        position = (cell["x"] * size, cell["y"] * size,
                    (cell["x"] + 1) * size, (cell["y"] + 1) * size)
        painting = canvas.crop(position)
        base_index = cell["frame"]
        base_rect = (base_index % 16 * 24, base_index // 16 * 24,
                     (base_index % 16 + 1) * 24, (base_index // 16 + 1) * 24)
        base = indices.crop(base_rect)
        if opaque.crop(base_rect).getextrema() != (255, 255) or remap.crop(base_rect).getbbox():
            raise ValueError("本切片只接受完全opaque、無目的地remap的base幀")
        for role, frame in [("terrain", base_index), ("foreground", cell.get("foreground_frame"))]:
            if frame is None:
                continue
            if frame in pictures:
                raise ValueError("不同物件格子共用frame，需要先制定重用策略")
            rect = (frame % 16 * 24, frame // 16 * 24, (frame % 16 + 1) * 24, (frame // 16 + 1) * 24)
            mask = opaque.crop(rect)
            selected_painting = painting
            if role == "foreground":
                differences = sum(a != b for a, b, visible in
                                  zip(base.getdata(), indices.crop(rect).getdata(), mask.getdata()) if visible)
                override = recipe.get("foreground_overrides", {}).get(str(frame))
                if differences and override is None:
                    raise ValueError("前景opaque RGB來源與base不同，不能直接共用新母稿")
                if override is not None:
                    if not differences or Path(override["file"]).name != override["file"]:
                        raise ValueError("獨立前景來源不適用或路徑不符")
                    path = master_path.parent / override["file"]
                    separate = Image.open(path).convert("RGBA").crop(override["crop"])
                    if list(separate.size) != override["painted_master_size"] or separate.getextrema()[3] != (255, 255):
                        raise ValueError("獨立前景母稿尺寸或不透明契約不符")
                    selected_painting = fit(separate, override)
                    if selected_painting.size != (size, size):
                        raise ValueError("獨立前景取樣尺寸不符")
                    foreground_sources.append({"frame": frame, "native_index_differences_from_base": differences,
                                               "file": override["file"], "sha256": sha(path),
                                               "method": "separate painted source"})
                else:
                    foreground_sources.append({"frame": frame, "native_index_differences_from_base": 0,
                                               "method": "shared painted source; original opaque indices identical"})
            alpha = mask.resize((size, size), Image.Resampling.NEAREST)
            picture = selected_painting.copy()
            picture.putdata([pixel[:3] + (255,) if present else (0, 0, 0, 0)
                             for pixel, present in zip(selected_painting.getdata(), alpha.getdata())])
            # 獨立用來源格子座標驗alpha，避免插值方式偷偷改變遮罩。
            for y in range(size):
                for x in range(size):
                    if picture.getpixel((x, y))[3] != mask.getpixel((x // scale, y // scale)):
                        raise ValueError("HD遮罩沒有保持原生覆蓋範圍")
            pictures[frame] = picture
            entries.append({"map": obj["map"], "frame": frame, "role": role,
                            "file": f"map{obj['map']:02d}-frame{frame:04d}-hd.png",
                            "native_content_sha256": source["frames"][frame]["native_content_sha256"],
                            "object_cell": [cell["x"], cell["y"]], "width": size, "height": size})
    reconstructed = Image.new("RGBA", canvas.size)
    for cell in obj["cells"]:
        reconstructed.paste(pictures[cell["frame"]], (cell["x"] * size, cell["y"] * size))
    if reconstructed.tobytes() != canvas.tobytes():
        raise ValueError("切回格子再重組產生了額外接縫")
    reference = Image.open(check_binding(sources, obj["reference"])).resize(canvas.size, Image.Resampling.NEAREST)
    comparison = Image.new("RGB", (canvas.width * 2, canvas.height + 32), (24, 24, 30))
    comparison.paste(reference, (0, 32))
    comparison.paste(canvas, (canvas.width, 32))
    labels = ImageDraw.Draw(comparison)
    labels.text((12, 10), "Native art reference / not runtime", fill="white")
    labels.text((canvas.width + 12, 10), "Painted object candidate / not runtime", fill="white")
    stage = Path(tempfile.mkdtemp(prefix=".terrain-object-stage-", dir=out.parent))
    try:
        for entry in entries:
            path = stage / entry["file"]
            pictures[entry["frame"]].save(path)
            entry["sha256"] = sha(path)
        canvas.save(stage / "object-assembled.png")
        comparison.save(stage / "object-comparison.png")
        result = {"schema_version": 1, "kind": "fd2_hd_terrain_object_candidate",
                  "status": "DATA-CANDIDATE", "private_local_only": True, "runtime_integrated": False,
                  "source_manifest_sha256": sha(sources / "manifest.json"),
                  "recipe_sha256": sha(recipe_path), "painted_master_sha256": sha(master_path),
                  "source_verification": source_report, "frames": entries,
                  "foreground_sources": foreground_sources,
                  "verification": {"native_alpha": "PASS", "foreground_source_consistency": "PASS",
                                   "cut_reconstruction_difference_bytes": 0,
                                   "external_terrain_seams_verified": False, "gpu_verified": False},
                  "assembled_sha256": sha(stage / "object-assembled.png"),
                  "comparison_sha256": sha(stage / "object-comparison.png"),
                  "tool_sha256": sha(Path(__file__))}
        if repeat_calibration:
            result["repeat_calibration"] = repeat_calibration
            result["verification"]["reconstruction_basis"] = "post-UV, post-repeat-edge calibration canvas"
        (stage / "manifest.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
        if os.path.lexists(out):
            raise ValueError("輸出在製作期間出現，保留既有內容")
        stage.rename(out)
        return result
    finally:
        shutil.rmtree(stage, ignore_errors=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ["pack", "maps", "sources", "recipe", "master", "out"]:
        parser.add_argument("--" + name, type=Path, required=True)
    args = parser.parse_args()
    result = build(args.pack, args.maps, args.sources, args.recipe, args.master, args.out)
    print(json.dumps({"status": result["status"], "frames": len(result["frames"]),
                      "verification": result["verification"]}, ensure_ascii=False))


if __name__ == "__main__":
    main()
