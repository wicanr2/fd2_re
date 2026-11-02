#!/usr/bin/env python3
"""Compile source-bound HD facility reactions from registered AI paintings.

Run with Pillow 11.3.0 in the project Docker toolchain. This is a deterministic
crop/compositing compiler; it does not generate or infer character artwork.
"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw
import PIL


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def pinned(root, ref):
    path = root / ref["file"]
    if path.name != ref["file"] or digest(path) != ref["sha256"]:
        raise ValueError(f"source differs: {path}")
    return Image.open(path).convert("RGBA")


def local_mask(rect, feather=6):
    mask = Image.new("L", (480, 480))
    x0, y0, x1, y1 = rect
    for y in range(y0, y1):
        for x in range(x0, x1):
            edge = min(x-x0, x1-1-x, y-y0, y1-1-y)
            mask.putpixel((x, y), min(255, max(0, round(255*edge/feather))))
    return mask


def flare(picture, spec):
    if not spec:
        return picture
    cx, cy, radius, ring = spec
    layer = Image.new("RGBA", (1920, 1920))
    draw = ImageDraw.Draw(layer)
    cx, cy, radius = cx*4, cy*4, radius*4
    narrow = max(2, radius//4)
    shape = [(cx, cy-radius), (cx+narrow, cy-narrow), (cx+radius, cy),
             (cx+narrow, cy+narrow), (cx, cy+radius), (cx-narrow, cy+narrow),
             (cx-radius, cy), (cx-narrow, cy-narrow)]
    if ring:
        draw.line(shape+[shape[0]], fill=(173, 213, 255, 245), width=5)
    else:
        draw.polygon(shape, fill=(173, 213, 255, 255))
        draw.ellipse((cx-5, cy-5, cx+5, cy+5), fill=(255, 255, 255, 255))
    return Image.alpha_composite(picture, layer.resize((480, 480), Image.Resampling.LANCZOS))


def compile_recipe(recipe, source_root, output):
    if PIL.__version__ != recipe["pillow"]:
        raise ValueError("Pillow version differs")
    if not output.is_dir() or output.stat().st_uid != source_root.stat().st_uid:
        raise ValueError("explicit owned output directory required")
    result = {"schema": "fd2-facility-success-build-v1", "entries": []}
    for entry in recipe["entries"]:
        scene = pinned(source_root, entry["scene"])
        if scene.size != (1920, 1200):
            raise ValueError("scene shape differs")
        offset = entry["portrait_offset"]
        x, y = offset % 320 * 6, offset // 320 * 6
        original = scene.crop((x, y, x+480, y+480))
        edited = pinned(source_root, entry["painting"]).resize((480, 480), Image.Resampling.LANCZOS)
        mask = local_mask(entry["edit_rect_hd"])
        reaction = Image.composite(edited, original, mask)
        base = reaction if entry["resource"] == 14 else original
        files = []
        for i, step in enumerate(entry["steps"]):
            if entry["resource"] == 14:
                # The explicit diamond contour transforms the generated gemstone.
                # No color-key or image-derived transparency is used.
                diamond = Image.new("L", (1920, 1920))
                ImageDraw.Draw(diamond).polygon([(a*4, b*4) for a, b in entry["gem_polygon_hd"]], fill=round(255*step["white"]))
                diamond = diamond.resize((480, 480), Image.Resampling.LANCZOS)
                picture = Image.composite(Image.new("RGBA", (480, 480), "white"), base, diamond)
            else:
                picture = Image.blend(original, reaction, step["wink"])
                picture = flare(picture, step.get("flare"))
            # Every authored change must remain inside the declared local region.
            box = entry["edit_rect_hd"]
            for py in range(480):
                for px in range(480):
                    if not (box[0] <= px < box[2] and box[1] <= py < box[3]) and picture.getpixel((px, py)) != original.getpixel((px, py)):
                        raise ValueError(f"reaction escaped edit region: {entry['resource']}/{i}/{px}/{py}")
            name = f"fdother-{entry['resource']:03d}-success-hd6-v1-{i:02d}.png"
            picture.save(output/name)
            files.append({"file": name, "sha256": digest(output/name)})
        row = {"resource": entry["resource"], "frames": files}
        if entry["resource"] == 14:
            scene.paste(base, (x, y))
            scene_name = "fdother-014-scene-handpainted-hd6-v2.png"
            master_name = "fdother-014-scene-handpainted-hd6-master-v2.png"
            scene.save(output/scene_name)
            scene.save(output/master_name)
            row["scene"] = {"file": scene_name, "sha256": digest(output/scene_name)}
            row["master"] = {"file": master_name, "sha256": digest(output/master_name), "size": [1920, 1200]}
            row["portraits"] = []
            for size in (80, 480):
                name = f"dato-131-facility-portrait-handpainted-v2-{size}.png"
                base.resize((size, size), Image.Resampling.LANCZOS).save(output/name)
                row["portraits"].append({"file": name, "width": size, "height": size, "sha256": digest(output/name)})
        result["entries"].append(row)
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--recipe", type=Path, required=True)
    parser.add_argument("--source-root", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    recipe = json.loads(args.recipe.read_text())
    result = compile_recipe(recipe, args.source_root, args.output)
    (args.output/"facility-success-build.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({"status": "CANDIDATE", "sets": len(result["entries"]), "frames": sum(len(x["frames"]) for x in result["entries"])}))


if __name__ == "__main__":
    main()
