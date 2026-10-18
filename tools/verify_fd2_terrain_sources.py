#!/usr/bin/env python3
"""獨立逐像素核對 Go 匯出的本機地形製圖來源；不作遊戲對拍。"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def check_binding(root, entry):
    path = Path(root) / entry["file"]
    if sha(path) != entry["sha256"]:
        raise ValueError(f"來源或產物雜湊不符：{path}")
    return path


def verify(pack, maps, sources):
    manifest = json.loads((sources / "manifest.json").read_text())
    if (manifest["schema_version"], manifest["kind"], manifest["frame_size"], manifest["columns"]) != (
            1, "fd2_native_terrain_art_sources", 24, 16):
        raise ValueError("製圖來源契約不符")
    if not manifest["private_local_only"] or manifest["hd_art_complete"]:
        raise ValueError("來源不能宣稱HD美術已完成或可公開")
    palette_path = check_binding(pack, manifest["palette"])
    dac = json.loads(palette_path.read_text())["dac_6bit_components"]
    palette = [tuple((value << 2) | (value >> 4) for value in dac[i:i + 3])
               for i in range(0, 768, 3)]
    frames = pixels = selectors = remapped = 0
    unique = set()
    if [item["map"] for item in manifest["maps"]] != list(range(33)):
        raise ValueError("原生地圖清單不完整")
    for item in manifest["maps"]:
        metadata = json.loads(check_binding(pack, item["bank"]).read_text())
        data = json.loads(check_binding(maps, item["map_data"]).read_text())
        count = len(metadata["sprites"])
        if count != item["frame_count"] or [frame["frame"] for frame in item["frames"]] != list(range(count)):
            raise ValueError("原生幀索引被刪除或重排")
        if item["selector_checks"] != data["w"] * data["h"] * 8:
            raise ValueError("selector未檢查全部格子與相位")
        if any(not 0 <= frame < count for frame in
               item["selected_terrain_frames"] + item["selected_foreground_frames"]):
            raise ValueError("selector索引越界")
        outputs = {entry["file"].split("-")[-1].split(".")[0]:
                   Image.open(check_binding(sources, entry)) for entry in item["outputs"]}
        if set(outputs) != {"rgba", "indices", "mask", "remap"}:
            raise ValueError("製圖來源缺少圖層")
        if any(image.size != (384, ((count + 15) // 16) * 24) for image in outputs.values()):
            raise ValueError("圖層尺寸或幀數不符")
        rgba = outputs["rgba"].convert("RGBA")
        for frame in item["frames"]:
            index = frame["frame"]
            layer_paths = [check_binding(pack, entry) for entry in frame["input_layers"]]
            if len(layer_paths) != 3 or [path.name for path in layer_paths] != ["frame.png", "mask.png", "remap_mask.png"]:
                raise ValueError("來源圖層順序不符")
            original, opaque, remap = [Image.open(path) for path in layer_paths]
            expected = [original.tobytes(), opaque.tobytes(), remap.tobytes()]
            native_content = expected[0] + bytes(bool(value) for value in expected[1]) + bytes(bool(value) for value in expected[2])
            content_hash = hashlib.sha256(native_content).hexdigest()
            if content_hash != frame["native_content_sha256"]:
                raise ValueError("原生解碼內容不符")
            unique.add(content_hash)
            rect = (index % 16 * 24, index // 16 * 24, (index % 16 + 1) * 24, (index // 16 + 1) * 24)
            for key, value in zip(("indices", "mask", "remap"), expected):
                if outputs[key].crop(rect).tobytes() != value:
                    raise ValueError(f"map{item['map']} frame{index} {key}位元組不同")
            for raw, mask, mode3, pixel in zip(*expected, rgba.crop(rect).getdata()):
                if mask and mode3:
                    raise ValueError("opaque與remap遮罩重疊")
                want = palette[raw] + (255,) if mask else (0, 0, 0, 0)
                if pixel != want:
                    raise ValueError("基準RGBA與原生索引／opaque遮罩不同")
            frames += 1
            pixels += 576
            remapped += sum(bool(value) for value in expected[2])
        selectors += item["selector_checks"]
    if len(unique) != manifest["unique_native_frames"]:
        raise ValueError("去重統計不符")
    for entry in manifest["previews"]:
        check_binding(sources, entry)
    return {"status": "PASS", "banks": 33, "frames": frames, "pixels_per_layer": pixels,
            "unique_native_frames": len(unique), "native_selector_checks": selectors,
            "destination_remap_pixels_preserved": remapped,
            "manifest_sha256": sha(sources / "manifest.json"),
            "scope": "逐像素圖源回讀，非HD美術／GPU／正常玩家／原版對拍"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--pack", type=Path, required=True)
    parser.add_argument("--maps", type=Path, required=True)
    parser.add_argument("--sources", type=Path, required=True)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()
    if args.report.exists():
        raise ValueError("不得覆寫既有核對收據")
    report = verify(args.pack, args.maps, args.sources)
    report["verifier_sha256"] = sha(Path(__file__))
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
