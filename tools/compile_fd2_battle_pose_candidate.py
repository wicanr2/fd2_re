#!/usr/bin/env python3
"""把單張生成母稿編製成原始 FIGANI 尺寸乘六的離線候選，保留 alpha。"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--inventory", required=True, type=Path)
    parser.add_argument("--resource", required=True, type=int)
    parser.add_argument("--frame", required=True, type=int)
    parser.add_argument("--master", required=True, type=Path)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--receipt", required=True, type=Path)
    args = parser.parse_args()
    if args.out.exists() or args.receipt.exists():
        raise SystemExit("候選或收據已存在，請使用新的版本檔名")
    inventory = json.loads(args.inventory.read_text())
    if inventory["status"] != "PASS-SOURCE-INVENTORY":
        raise SystemExit("原始盤點未通過")
    resource = next(x for x in inventory["resources"] if x["resource"] == args.resource)
    frame = next(x for x in resource["frames"] if x["frame"] == args.frame)
    with Image.open(args.master) as source:
        if source.mode != "RGBA":
            raise SystemExit("母稿必須帶有獨立 RGBA alpha")
        alpha = source.getchannel("A")
        ink = alpha.getbbox()
        if not ink or alpha.getextrema()[0] != 0 or alpha.getextrema()[1] != 255:
            raise SystemExit("母稿缺少透明背景或不透明核心")
        art = source.crop(ink)
        size = (frame["width"] * 6, frame["height"] * 6)
        scale = min(size[0] / art.width, size[1] / art.height)
        fitted = art.resize((max(1, round(art.width * scale)), max(1, round(art.height * scale))), Image.Resampling.LANCZOS)
        origin = ((size[0] - fitted.width) // 2, size[1] - fitted.height)
        canvas = Image.new("RGBA", size)
        # paste 不使用 mask，避免 alpha 被平方；原有覆蓋值原樣存入。
        canvas.paste(fitted, origin)
        canvas.save(args.out)
        source_size = list(source.size)
    receipt = {"schema_version": 1, "status": "PASS-COMPILED-POSE-CANDIDATE",
               "scope": "僅單姿勢編製；不是正式動畫、原版對拍或正常 GUI",
               "inventory_sha256": sha(args.inventory), "resource": args.resource, "frame": args.frame,
               "master": {"file": args.master.name, "sha256": sha(args.master), "size": source_size},
               "native_frame": frame, "native_descriptor_rectangle": [frame["x"], frame["y"], frame["width"], frame["height"]],
               "policy": "contain_alpha_bbox_center_x_bottom_in_native_descriptor_6x_candidate",
               "alpha_bbox": list(ink), "fit_size": list(fitted.size), "origin_in_frame": list(origin),
               "output": {"file": args.out.name, "sha256": sha(args.out), "size": list(size)},
               "tool_sha256": sha(Path(__file__)), "runtime_registered": False}
    args.receipt.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(receipt, ensure_ascii=False))


if __name__ == "__main__":
    main()
