#!/usr/bin/env python3
"""Update selected modern FDICON frame hashes without reformatting the catalog.

The catalog is intentionally edited as text: JSON serialization would create a
large, noisy diff and could hide an accidental change to unrelated metadata.
Only the ``frame_sha256`` array inside explicitly selected group objects may
change.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import struct
from pathlib import Path

GROUP_ID = re.compile(r"^modern\.fdicon\.group_(\d{3})\.style_a$")
def _png_size_and_alpha(path: Path) -> None:
    raw = path.read_bytes()
    if len(raw) < 33 or raw[:8] != b"\x89PNG\r\n\x1a\n" or raw[12:16] != b"IHDR":
        raise ValueError(f"{path}: 不是有效 PNG")
    width, height, depth, color_type = struct.unpack(">IIBB", raw[16:26])
    if (width, height) != (24, 24):
        raise ValueError(f"{path}: 尺寸必須是 24x24，實際為 {width}x{height}")
    # Color type 6 is RGBA.  The full pixel-level binary-alpha check is done
    # through Pillow below; rejecting other color types here avoids silently
    # accepting palette/grayscale data as RGBA.
    if depth != 8 or color_type != 6:
        raise ValueError(f"{path}: 必須是 8-bit RGBA PNG")
    try:
        from PIL import Image
    except ImportError as exc:  # pragma: no cover - tool environment failure
        raise ValueError("需要 Pillow 才能驗證 PNG alpha") from exc
    with Image.open(path) as image:
        if image.size != (24, 24) or image.mode != "RGBA":
            raise ValueError(f"{path}: 必須解碼為 24x24 RGBA")
        alpha = image.getchannel("A")
        values = set(alpha.getdata())
    if not values.issubset({0, 255}):
        raise ValueError(f"{path}: alpha 必須是二值（0 或 255）")


def _object_span(text: str, start: int) -> tuple[int, int]:
    """Return the JSON object span containing a known asset_id string."""
    opening = text.rfind("{", 0, start)
    if opening < 0:
        raise ValueError("找不到資產物件起點")
    depth = 0
    quoted = False
    escaped = False
    for pos in range(opening, len(text)):
        char = text[pos]
        if quoted:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                quoted = False
            continue
        if char == '"':
            quoted = True
        elif char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return opening, pos + 1
    raise ValueError("資產物件 JSON 未結束")


def _array_span(object_text: str, key: str) -> tuple[int, int]:
    match = re.search(rf'"{re.escape(key)}"\s*:\s*\[', object_text)
    if not match:
        raise ValueError(f"資產物件缺少 {key}")
    opening = object_text.find("[", match.start())
    depth = 0
    quoted = False
    escaped = False
    for pos in range(opening, len(object_text)):
        char = object_text[pos]
        if quoted:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                quoted = False
            continue
        if char == '"':
            quoted = True
        elif char == "[":
            depth += 1
        elif char == "]":
            depth -= 1
            if depth == 0:
                return opening, pos + 1
    raise ValueError(f"{key} 陣列未結束")


def _hashes(asset: dict, asset_root: Path) -> list[str]:
    files = asset.get("files")
    contract = asset.get("consumer_contract")
    expected_count = {
        "fdicon_map_sprite_12x24_v1": 12,
        "fdicon_static_sprite_3x24_v1": 3,
    }.get(contract)
    if expected_count is None:
        raise ValueError(f"不支援的 selector consumer_contract：{contract!r}")
    if asset.get("frame_count") != expected_count:
        raise ValueError(f"selector frame_count 與 consumer_contract 不一致：{asset.get('frame_count')!r}")
    if not isinstance(files, list) or len(files) != expected_count or len(set(files)) != expected_count:
        raise ValueError(f"selector 必須正好包含 {expected_count} 個 files")
    if asset.get("width") != 24 or asset.get("height") != 24:
        raise ValueError("selector catalog 尺寸必須是 24x24")
    result: list[str] = []
    for name in files:
        if not isinstance(name, str) or Path(name).name != name or Path(name).suffix.lower() != ".png":
            raise ValueError(f"不安全或非 PNG 檔名：{name!r}")
        path = asset_root / name
        if not path.is_file():
            raise ValueError(f"找不到 selector 素材：{path}")
        _png_size_and_alpha(path)
        result.append(hashlib.sha256(path.read_bytes()).hexdigest())
    return result


def update(catalog_path: Path, asset_root: Path, groups: list[int]) -> None:
    if not groups or len(groups) != len(set(groups)):
        raise ValueError("必須提供不重複的 --group N")
    if any(group < 0 or group > 95 for group in groups):
        raise ValueError("--group 必須介於 0 與 95")
    text = catalog_path.read_text(encoding="utf-8")
    replacements: list[tuple[int, int, str]] = []
    for group in groups:
        needle = f'"asset_id": "modern.fdicon.group_{group:03d}.style_a"'
        marker = text.find(needle)
        if marker < 0:
            raise ValueError(f"catalog 缺少 group_{group:03d}")
        start, end = _object_span(text, marker)
        asset_text = text[start:end]
        asset = json.loads(asset_text)
        match = GROUP_ID.fullmatch(asset.get("asset_id", ""))
        if not match or int(match.group(1)) != group or asset.get("role") != "map_sprite_set":
            raise ValueError(f"group_{group:03d} 不是 selector map_sprite_set")
        hashes = _hashes(asset, asset_root)
        array_start, array_end = _array_span(asset_text, "frame_sha256")
        old_array = asset_text[array_start:array_end]
        indent_match = re.search(r"\n(\s*)\"", old_array)
        indent = indent_match.group(1) if indent_match else "  "
        replacement = "[\n" + "\n".join(
            f'{indent}"{digest}"' + ("," if index < len(hashes) - 1 else "")
            for index, digest in enumerate(hashes)
        ) + f"\n{indent[:-2] if len(indent) >= 2 else ''}]"
        replacements.append((start + array_start, start + array_end, replacement))
    updated = text
    for start, end, replacement in reversed(replacements):
        updated = updated[:start] + replacement + updated[end:]
    # Ensure the replacement itself remains valid JSON before touching disk.
    json.loads(updated)
    catalog_path.write_text(updated, encoding="utf-8", newline="")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=Path, required=True)
    parser.add_argument("--asset-root", type=Path, required=True)
    parser.add_argument("--group", type=int, action="append", required=True)
    args = parser.parse_args()
    try:
        update(args.catalog, args.asset_root, args.group)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        parser.error(str(exc))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
