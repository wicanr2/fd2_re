#!/usr/bin/env python3
"""以實際視窗PNG核對Draw已發布的144地形RGB；不宣稱原版像素對拍。"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--receipt", type=Path, required=True)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()
    if args.report.exists():
        raise ValueError("不得覆寫既有GPU收據")
    receipt = json.loads(args.receipt.read_text())
    if receipt["status"] != "PASS":
        raise ValueError("正常GUI流程尚未通過")
    names = [name for name in receipt["screenshots"] if name.startswith("terrain-hd-")]
    if len(names) < 2:
        raise ValueError("缺少正常移鏡頭與F2返回的房屋畫面")
    label=names[0].removeprefix('terrain-hd-').removesuffix('.png').removesuffix('-restored')
    expectation=receipt.get('terrain_expectation')
    required=set()
    if expectation:
        frames=expectation.get('frames',[])
        if (expectation.get('label')!=label or not frames or any(type(frame) is not int or not 0<=frame<=65535 for frame in frames)
                or len(set(frames))!=len(frames)):
            raise ValueError('明示地形取景範圍不完整或與圖名不符')
        required=set(frames)
    elif label not in {'house','roof-object','props'}:
        raise ValueError('新取景標籤缺少明示frame清單')
    samples = []
    for name in names:
        item = receipt["screenshots"][name]
        path = args.receipt.parent / name
        if digest(path.read_bytes()) != item["sha256"]:
            raise ValueError("GPU圖片被改寫")
        image = Image.open(path).convert("RGBA")
        if image.size != (1920, 1200):
            raise ValueError("不是目標1920×1200視窗")
        states = [item["gpu_before"], item["state"]]
        candidates = {}
        for state in states:
            if not state.get("hd_world"):
                continue
            for tile in state.get("hd_world_terrain", []):
                key = (tile["frame"], tuple(tile["rect"]), tile["rgba_sha256"])
                candidates[key] = tile
        matched, different = [], []
        for (frame, rect, expected), tile in candidates.items():
            x0, y0, x1, y1 = rect
            if x1-x0 != 144 or y1-y0 != 144 or x0 < 24 or y0 < 24 or x1 > 1896 or y1 > 1176:
                continue
            actual = digest(image.crop(rect).tobytes())
            target = {"frame": frame, "rect": list(rect), "rgba_sha256": expected}
            (matched if actual == expected else different).append(target)
        if len(matched) < 8:
            raise ValueError(f"{name}只有{len(matched)}個完整圖塊可比；保留失敗，不推定來源")
        if name.startswith('terrain-hd-props') and not {40,42,43,203,204,205,206,207,208}.issubset({t['frame'] for t in matched}):
            raise ValueError(f"{name}缺少九張新物件的完整GPU相符圖塊")
        if required and not required.issubset({t['frame'] for t in matched}):
            raise ValueError(f'{name}缺少明示範圍的完整GPU相符圖塊')
        samples.append({"file": name, "status": "PASS", "matching_tiles": matched,
                        "matched_pixels": len(matched)*144*144,
                        "not_compared_tiles": different,
                        "limits": "只驗完整可見且未被UI／人物／前景蓋住的相符圖塊；其他候選不當成通過"})
    faithful = args.receipt.parent / ('terrain-faithful-'+label+'.png')
    reference = Image.open(faithful).convert("RGBA")
    first = Image.open(args.receipt.parent/names[0]).convert("RGBA")
    difference = sum(a != b for a, b in zip(first.getdata(), reference.getdata()))
    if difference == 0:
        raise ValueError("兩主題畫面沒有任何差異")
    report = {"status": "PASS", "samples": samples,
              "hd_vs_faithful_difference_pixels": difference,
              "gui_receipt_sha256": digest(args.receipt.read_bytes()),
              "tool_sha256": digest(Path(__file__).read_bytes()),
              "scope": "正常首章移鏡頭與F2、實際GPU呈現所選新RGB；不新增原版oracle或整章E2。"}
    if expectation:
        report['terrain_expectation']=expectation
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({"status": "PASS", "images": len(samples),
                      "matching_tiles": [len(sample["matching_tiles"]) for sample in samples]}, ensure_ascii=False))


if __name__ == "__main__":
    main()
