#!/usr/bin/env python3
"""編製FDOTHER5第68..79項；原始母稿與索引資料保持不變。"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def build(original: Path, recipe_path: Path, out: Path) -> dict:
    recipe = json.loads(recipe_path.read_text())
    if set(recipe) != {"schema_version", "bank_sha256", "sources", "entries"} or recipe["schema_version"] != 1:
        raise ValueError("invalid recipe")
    bank_dir = original / "ui/fdother_005_lmi1_opaque"
    bank_path = bank_dir / "bank.json"
    bank = json.loads(bank_path.read_text())
    if (sha(bank_path) != recipe["bank_sha256"] or bank["source"]["resource"] != 5 or
            bank["source"]["sha256"] != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"):
        raise ValueError("native bank mismatch")
    if set(recipe["entries"]) != {str(i) for i in range(68, 80)}:
        raise ValueError("incomplete entry set")
    sources, source_files, source_hashes = {}, [], []
    for key, source in recipe["sources"].items():
        if set(source) != {"file", "sha256"}:
            raise ValueError("invalid master descriptor")
        path = recipe_path.parent / source["file"]
        if Path(source["file"]).name != source["file"] or sha(path) != source["sha256"]:
            raise ValueError("master sha256 mismatch")
        image = Image.open(path)
        if image.mode != "RGBA" or image.getchannel("A").getextrema()[0] != 0 or image.getchannel("A").getextrema()[1] < 250:
            raise ValueError("invalid source alpha")
        sources[key] = image
        source_files.append(source["file"])
        source_hashes.append(source["sha256"])
    compiled, descriptions, records, used = [], [], [], set()
    for index in range(68, 80):
        assignment = recipe["entries"][str(index)]
        if set(assignment) != {"source", "opacity"} or assignment["source"] not in sources:
            raise ValueError("invalid assignment")
        opacity = assignment["opacity"]
        if type(opacity) not in (float, int) or not 0 < opacity <= 1:
            raise ValueError("invalid opacity")
        entry = bank["entries"][index]
        if entry["index"] != index or entry["width"] not in range(1, 129) or entry["height"] not in range(1, 129):
            raise ValueError("invalid native dimensions")
        native = Image.open(bank_dir / entry["frame"])
        if native.mode != "P" or native.size != (entry["width"], entry["height"]):
            raise ValueError("invalid indexed source")
        pixels = bytes(native.getdata())
        mask = Image.frombytes("L", native.size, bytes(255 if v else 0 for v in pixels))
        box = mask.getbbox()
        if box is None:
            raise ValueError("empty native entry")
        mother = sources[assignment["source"]]
        used.add(assignment["source"])
        ink = mother.crop(mother.getbbox()).resize(((box[2]-box[0])*6, (box[3]-box[1])*6), Image.Resampling.LANCZOS)
        if opacity != 1:
            ink.putalpha(ink.getchannel("A").point([round(i*opacity) for i in range(256)]))
        picture = Image.new("RGBA", (native.width*6, native.height*6))
        picture.alpha_composite(ink, (box[0]*6, box[1]*6))
        low, high = picture.getchannel("A").getextrema()
        if low != 0 or high == 0:
            raise ValueError("compiled frame missing transparency or ink")
        compiled.append(picture)
        descriptions.append(dict(index=index, width=native.width, height=native.height, pixels_sha256=hashlib.sha256(pixels).hexdigest()))
        records.append(dict(index=index, source=assignment["source"], opacity=opacity, native_ink_box=list(box), alpha_extrema=[low, high]))
    if used != set(sources):
        raise ValueError("unused source")
    # All sources are checked before any output directory or frame is created.
    out.mkdir(parents=True, exist_ok=True)
    files = []
    for index, picture in enumerate(compiled, 68):
        name = f"death-fdother005-{index:03d}-hd6.png"
        picture.save(out/name)
        files.append(name)
    for source in recipe["sources"].values():
        (out/source["file"]).write_bytes((recipe_path.parent/source["file"]).read_bytes())
    asset = dict(asset_id="shared-death-fdother005-hd6", role="map_overlay_hd_set", status="runtime_candidate",
                 files=files, width=1920, height=1200, frame_count=12, frame_sha256=[sha(out/f) for f in files],
                 consumer_contract="native_lmi1_rgba_hd6_v1", alpha_contract="coverage", cycle_policy="native_lmi1_entries",
                 source_archive="FDOTHER.DAT", source_resource=5, source_metadata_sha256=sha(bank_path),
                 native_lmi_frames=descriptions, layout_policy="native_lmi1_rect_hd6", source_files=source_files,
                 source_sha256=source_hashes, source_refs=["docs/knowledge-base/61-modern-theme-runtime-spec.md",
                 "docs/data/fd2-modern-hd-battlefield-20261019.json", "tools/build_fd2_lmi_death_hd.py"])
    (out/"asset.json").write_text(json.dumps(asset,ensure_ascii=False,indent=2)+"\n")
    (out/"compiled.json").write_text(json.dumps(dict(schema_version=1,recipe_sha256=sha(recipe_path),entries=records),indent=2)+"\n")
    return asset


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original-pack", type=Path, required=True)
    parser.add_argument("--recipe", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(build(args.original_pack,args.recipe,args.out),ensure_ascii=False))
