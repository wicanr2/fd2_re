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
KEY_RE = re.compile(r"^[a-z][a-z0-9]*(?:\.[a-z0-9_]+)+$")
VAR_RE = re.compile(r"^%(?:%|[-+0-9.#]*[a-zA-Z])$")
APPROVED_SOURCE = {
    "battle.attack.miss": "legacy.go.remake.cmd.fd2.main.l5973-c22",
    "battle.attack.hit": "legacy.go.remake.cmd.fd2.main.l5975-c25",
    "battle.attack.critical_suffix": "legacy.go.remake.cmd.fd2.main.l5977-c14",
    "battle.attack.exp_suffix": "legacy.go.remake.cmd.fd2.main.l5980-c26",
    "battle.mp.insufficient": "legacy.go.remake.cmd.fd2.main.l5091-c14",
    "battle.command.unavailable": "legacy.go.remake.cmd.fd2.main.l5187-c12",
    "battle.attack.choose_target": "legacy.go.remake.cmd.fd2.main.l5193-c13",
    "battle.command.native_unavailable": "legacy.go.remake.cmd.fd2.main.l5201-c14",
    "battle.spell.sealed": "legacy.go.remake.cmd.fd2.main.l5211-c14",
    "battle.spell.none": "legacy.go.remake.cmd.fd2.main.l5218-c13",
    "battle.item.choose_slot": "legacy.go.remake.cmd.fd2.main.l5225-c13",
    "battle.spell.target_prompt": "legacy.go.remake.cmd.fd2.main.l5168-c14",
    "battle.spell.blocked": "legacy.go.remake.cmd.fd2.main.l6170-c12",
    "battle.unit.paralyzed": "legacy.go.remake.cmd.fd2.main.l6145-c12",
    "battle.spell.result_damage": "legacy.go.remake.cmd.fd2.main.l6194-c10",
    "battle.spell.result_heal": "legacy.go.remake.cmd.fd2.main.l6194-c10",
    "battle.spell.miss_suffix": "legacy.go.remake.cmd.fd2.main.l6196-c13",
    "battle.treasure.gold": "legacy.go.remake.cmd.fd2.main.l5396-c13",
    "battle.treasure.item": "legacy.go.remake.cmd.fd2.main.l5398-c13",
    "battle.treasure.inventory_full": "legacy.go.remake.cmd.fd2.main.l5402-c12",
    "battle.reward.item": "legacy.go.remake.cmd.fd2.main.l5457-c12",
    "battle.reward.item_full": "legacy.go.remake.cmd.fd2.main.l5459-c12",
    "battle.reward.gold": "legacy.go.remake.cmd.fd2.main.l5463-c11",
    "battle.spell.menu_title": "legacy.go.remake.cmd.fd2.main.l8687-c22",
    "church.transfer.empty": "legacy.go.remake.cmd.fd2.native_church_input.l209-c12",
    "church.transfer.success": "legacy.go.remake.cmd.fd2.native_church_input.l271-c12",
    "church.revive.success": "legacy.go.remake.cmd.fd2.main.l4711-c10",
    "church.class_change.success": "legacy.go.remake.cmd.fd2.main.l4674-c10",
    "church.class_change.confirm_title": "legacy.go.remake.cmd.fd2.main.l9090-c13",
    "church.class_change.empty": "legacy.go.remake.cmd.fd2.main.l9094-c24",
    "church.class_change.target": "legacy.go.remake.cmd.fd2.main.l9100-c35",
    "common.yes": "legacy.go.remake.cmd.fd2.main.l9101-c36",
    "common.no": "legacy.go.remake.cmd.fd2.main.l9101-c41",
    "system.locale.changed": "runtime.settings.locale.changed",
    "system.audio.changed": "runtime.settings.audio.changed",
    "save.unsupported": "runtime.save.unsupported",
    "save.postbattle_blocked": "runtime.save.postbattle_blocked",
    "save.saved": "runtime.save.saved",
    "save.none": "runtime.save.none",
    "save.node_missing": "runtime.save.node_missing",
    "save.loaded": "runtime.save.loaded",
    "shop.greeting.weapon": "FDTXT_000/string_0440",
    "shop.greeting.item": "FDTXT_000/string_0501",
    "shop.purchase.question.weapon": "FDTXT_000/string_0439",
    "shop.purchase.question.item": "FDTXT_000/string_0502",
    "shop.purchase.insufficient.weapon": "FDTXT_000/string_0438",
    "shop.purchase.insufficient.item": "FDTXT_000/string_0504",
    "shop.purchase.no_recipient.weapon": "FDTXT_000/string_0437",
    "shop.purchase.no_recipient.item": "FDTXT_000/string_0505",
    "shop.purchase.equip_question": "FDTXT_000/string_0507",
    "shop.recipient.full": "FDTXT_000/string_0506",
    "shop.transfer.destination_prompt": "FDTXT_000/string_0510",
    "shop.transfer.empty_source": "FDTXT_000/string_0511",
    "shop.transfer.source_prompt": "FDTXT_000/string_0512",
    "shop.purchase.success": "legacy.go.remake.cmd.fd2.main.l4468-c17",
    "shop.sell.success": "legacy.go.remake.cmd.fd2.main.l4506-c15",
    "shop.purchase.equip_prompt": "legacy.go.remake.cmd.fd2.main.l4555-c15",
    "shop.recipient.none": "legacy.go.remake.cmd.fd2.main.l4576-c13",
    "shop.purchase.equip_prompt.title": "legacy.go.remake.cmd.fd2.main.l8864-c24",
    "shop.purchase.equip_prompt.controls": "legacy.go.remake.cmd.fd2.main.l8865-c24",
    "shop.sell.item_title": "legacy.go.remake.cmd.fd2.main.l8874-c24",
    "shop.item.equipped_label": "legacy.go.remake.cmd.fd2.main.l8883-c14",
    "shop.sell.roster_title": "legacy.go.remake.cmd.fd2.main.l8891-c23",
    "shop.sell.inventory_count": "legacy.go.remake.cmd.fd2.main.l8898-c24",
    "shop.purchase.recipient_title": "legacy.go.remake.cmd.fd2.main.l8905-c23",
    "shop.purchase.recipient_inventory": "legacy.go.remake.cmd.fd2.main.l8912-c24",
    "shop.panel.title_controls": "legacy.go.remake.cmd.fd2.main.l8919-c22",
    "hotel.title.fallback": "legacy.go.remake.cmd.fd2.main.l8957-c12",
    "hotel.controls": "legacy.go.remake.cmd.fd2.main.l8971-c23",
    "preparation.title": "legacy.go.remake.cmd.fd2.main.l8981-c23",
    "preparation.save_prompt": "legacy.go.remake.cmd.fd2.main.l8985-c14",
    "preparation.deploy.controls": "legacy.go.remake.cmd.fd2.main.l9006-c23",
    "preparation.unknown_character": "legacy.go.remake.cmd.fd2.main.l9020-c13",
    "preparation.enter_confirm": "legacy.go.remake.cmd.fd2.main.l9032-c23",
    "preparation.save_hint": "legacy.go.remake.cmd.fd2.main.l9053-c22",
    "battle.result.win": "legacy.go.remake.cmd.fd2.main.l8410-c9",
    "battle.result.lose": "legacy.go.remake.cmd.fd2.main.l8413-c9",
    "battle.result.continue": "legacy.go.remake.cmd.fd2.main.l8418-c24",
    "postbattle.preparation.title": "legacy.go.remake.cmd.fd2.main.l8857-c12",
    "title.load.controls": "legacy.go.remake.cmd.fd2.title.l826-c22",
    "title.load.slot": "legacy.go.remake.cmd.fd2.title.l834-c13",
    "save.slot.empty": "legacy.go.remake.cmd.fd2.title.l836-c14",
    "save.slot.present": "legacy.go.remake.cmd.fd2.title.l838-c14",
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
        if entry["source_string_id"] != APPROVED_SOURCE.get(key): fail(f"{path}: source is not an approved player-visible ID for {key}")
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
