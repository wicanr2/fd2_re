#!/usr/bin/env python3
"""Validate FD2 locale packs and their cross-pack contract."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

OFFICIAL = {"zh-Hant", "zh-Hans", "ja", "en"}
LOCALE_RE = re.compile(r"^[a-z]{2,3}(?:-[A-Z][a-z]{3})?(?:-[A-Z]{2}|-[0-9]{3})?$")
PACK_ID_RE = re.compile(r"^[a-z0-9][a-z0-9._-]+$")
KEY_RE = re.compile(r"^legacy\.go\.remake\.cmd\.fd2\.[a-z0-9_]+\.l[0-9]+-c[0-9]+$")
VAR_RE = re.compile(r"^%(?:%|[-+0-9.#]*[a-zA-Z])$")
ALLOWED_SOURCE = {
    "legacy.go.remake.cmd.fd2.main.l5957-c27", "legacy.go.remake.cmd.fd2.main.l5957-c37",
    "legacy.go.remake.cmd.fd2.main.l5973-c22", "legacy.go.remake.cmd.fd2.main.l5975-c25",
    "legacy.go.remake.cmd.fd2.main.l5977-c14", "legacy.go.remake.cmd.fd2.main.l5980-c26",
}

def fail(message: str) -> None:
    raise ValueError(message)

def validate_one(path: Path) -> dict:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"{path}: invalid JSON: {exc}")
    if not isinstance(data, dict): fail(f"{path}: root must be object")
    allowed_fields = {"schema_version", "pack_id", "locale", "kind", "base_locale", "layout_profile", "font", "entries"}
    extra = set(data) - allowed_fields
    if extra: fail(f"{path}: invalid fields {sorted(extra)}")
    required = {"schema_version", "pack_id", "locale", "kind", "layout_profile", "font", "entries"}
    missing = required - set(data)
    if missing: fail(f"{path}: missing {sorted(missing)}")
    if data["schema_version"] != 1: fail(f"{path}: unsupported schema_version")
    if not isinstance(data.get("pack_id"), str) or not PACK_ID_RE.fullmatch(data["pack_id"]): fail(f"{path}: invalid pack_id")
    locale, kind = data["locale"], data["kind"]
    if not isinstance(locale, str) or not LOCALE_RE.fullmatch(locale): fail(f"{path}: invalid BCP47 locale")
    if kind not in ("official", "community"): fail(f"{path}: invalid kind")
    if kind == "official" and "base_locale" in data: fail(f"{path}: official pack cannot fallback")
    if kind == "community":
        if not isinstance(data.get("base_locale"), str) or not LOCALE_RE.fullmatch(data["base_locale"]): fail(f"{path}: community base_locale invalid")
    if data["layout_profile"] not in ("dos-8x8-latin", "pc98-16x16-cjk"): fail(f"{path}: uncontrolled layout profile")
    font = data["font"]
    if not isinstance(font, dict) or font.get("profile") not in ("fd2-latin-8x8", "fd2-cjk-16x16"): fail(f"{path}: uncontrolled font profile")
    font_path = font.get("path", "")
    if not isinstance(font_path, str) or not re.fullmatch(r"fonts/[a-z0-9][a-z0-9._/-]*\.(?:png|bin|json)", font_path) or ".." in font_path: fail(f"{path}: uncontrolled font path")
    # 受版控官方包的路徑以 remake/assets 為根；不讓一份只寫了字串的
    # manifest 在實際缺字型檔時通過。暫存社群包則由安裝器另驗證包根。
    if kind == "official" and path.parent.parent.name == "locales":
        asset_root = path.parent.parent.parent
        if not (asset_root / font_path).is_file():
            fail(f"{path}: font asset does not exist: {font_path}")
    entries = data["entries"]
    if not isinstance(entries, dict) or not entries: fail(f"{path}: entries must be non-empty object")
    for key, entry in entries.items():
        if not KEY_RE.fullmatch(key) or not isinstance(entry, dict): fail(f"{path}: invalid entry {key!r}")
        if set(entry) != {"text", "variables", "source_string_id"}: fail(f"{path}: invalid fields for {key}")
        if not isinstance(entry["text"], str) or not entry["text"]: fail(f"{path}: empty text for {key}")
        variables = entry["variables"]
        if not isinstance(variables, list) or any(not isinstance(v, str) or not VAR_RE.fullmatch(v) for v in variables): fail(f"{path}: invalid variables for {key}")
        if entry["source_string_id"] != key or entry["source_string_id"] not in ALLOWED_SOURCE: fail(f"{path}: source is not an approved player-visible ID for {key}")
    return data

def validate(paths: list[Path]) -> list[dict]:
    packs = [validate_one(p) for p in paths]
    official = [p for p in packs if p["kind"] == "official"]
    if {p["locale"] for p in official} != OFFICIAL: fail(f"official locales must be exactly {sorted(OFFICIAL)}")
    if len(official) != len(OFFICIAL): fail("duplicate official locale")
    keys = set(official[0]["entries"])
    signatures = {k: tuple(official[0]["entries"][k]["variables"]) for k in keys}
    for pack in official[1:]:
        if set(pack["entries"]) != keys: fail(f"official key set mismatch in {pack['locale']}")
        for key, entry in pack["entries"].items():
            if tuple(entry["variables"]) != signatures[key]: fail(f"variable signature mismatch in {pack['locale']}:{key}")
    for pack in packs:
        if pack["kind"] == "community":
            if pack["base_locale"] not in OFFICIAL: fail(f"community pack {pack['pack_id']} base_locale is not an official locale")
            if any(k not in keys for k in pack["entries"]): fail(f"community pack {pack['pack_id']} contains unknown key")
    return packs

if __name__ == "__main__":
    try:
        validate([Path(arg) for arg in sys.argv[1:]])
    except ValueError as exc:
        print(f"locale validation failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
    print(f"validated {len(sys.argv) - 1} locale pack(s)")
