//! 與凍結 Go 版 `remake/cmd/fd2-golden-fdicon` 的黃金差分。
//!
//! 由 `tools/rust_golden_fdicon.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_FDICON`。真實原版檔不在
//! 黃金檔裡：`FD2_FDICON_REAL_FLAME2` 指向含 FDICON.B24 的目錄，`FD2_FDICON_REAL_ASSETS`
//! 指向分離素材包根目錄，兩者都先核對黃金檔記錄的 SHA-256。設定 `FD2_REQUIRE_GOLDEN=1`
//! 時，黃金檔或真實檔目錄缺少即失敗，避免把「沒跑到」當成通過。
//!
//! 比對方式：原版 B24、真實分離素材、格式正確的產生輸入以及全部純函式逐欄比對，fdicon 自己
//! 的錯誤文字必須逐字相同。格式錯誤的 JSON、PNG 與檔案系統輸入（見 [`accept_reject_only`]）
//! 只比接受或拒收：Rust 版刻意不照做 Go 標準函式庫對這類輸入的行為與錯誤文字。連接受與否
//! 都不同的案例列在 [`KNOWN_DIFFERENCES`]，標明刻意不照做的原因。

use fd2_formats::fdicon::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

fn digest(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}

fn err_value<T>(r: &Result<T>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

fn sprite_digest(s: &Sprite) -> String {
    let mut h = Sha256::new();
    h.update(&s.pixels);
    h.update(&s.mask);
    h.update(&s.remap_mask);
    h.finalize().iter().map(|x| format!("{x:02x}")).collect()
}

fn bank_result(r: &Result<Bank>) -> Value {
    let mut out = serde_json::Map::new();
    out.insert("err".into(), err_value(r));
    if let Ok(b) = r {
        out.insert("digests".into(), b.sprites.iter().map(sprite_digest).collect());
    }
    Value::Object(out)
}

fn diff(before: &[u8], after: &[u8]) -> Value {
    Value::Array(
        before.iter().zip(after).enumerate().filter(|(_, (a, b))| a != b).map(|(i, (_, b))| json!([i, b])).collect(),
    )
}

/// 對應 Go `protect`：回報執行期 panic（Go 的 err 此時是 nil）。
fn protect(f: impl FnOnce() -> Result<()>) -> (Value, bool) {
    let previous = std::panic::take_hook();
    std::panic::set_hook(Box::new(|_| {}));
    let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(f));
    std::panic::set_hook(previous);
    match r {
        Ok(result) => (err_value(&result), false),
        Err(_) => (Value::Null, true),
    }
}

fn i(v: &Value, key: &str) -> i64 {
    v[key].as_i64().unwrap_or_else(|| panic!("欄位 {key} 不是整數：{}", v[key]))
}

fn u8_(v: &Value, key: &str) -> u8 {
    i(v, key) as u8
}

fn b(v: &Value, key: &str) -> bool {
    v[key].as_bool().unwrap()
}

fn bytes_of(v: &Value, key: &str) -> Vec<u8> {
    match &v[key] {
        Value::Null => Vec::new(),
        s => unhex(s.as_str().unwrap()),
    }
}

fn sprite_from(v: &Value) -> Sprite {
    let a = v.as_array().unwrap();
    let layer = |k: usize| unhex(a[k].as_str().unwrap());
    Sprite { pixels: layer(0), mask: layer(1), remap_mask: layer(2) }
}

fn ident_bank(n: usize) -> Bank {
    Bank {
        sprites: (0..n)
            .map(|k| Sprite { pixels: vec![k as u8, (k >> 8) as u8], mask: Vec::new(), remap_mask: Vec::new() })
            .collect(),
    }
}

fn ident(s: &Sprite) -> i64 {
    s.pixels[0] as i64 | (s.pixels[1] as i64) << 8
}

fn sprite_ident(r: Result<&Sprite>) -> Value {
    let mut out = serde_json::Map::new();
    out.insert("err".into(), err_value(&r));
    if let Ok(s) = r {
        out.insert("sprite".into(), json!(ident(s)));
    }
    Value::Object(out)
}

fn cells_from(v: &Value) -> Vec<NativeTerrainCell> {
    v.as_array()
        .unwrap()
        .iter()
        .map(|c| NativeTerrainCell { tile: c[0].as_u64().unwrap() as u16, blit_mode: c[1].as_u64().unwrap() as u8 })
        .collect()
}

/// 測試用的暫存根目錄；Go 端的 `$TMP` 對應到這裡。
struct Workspace {
    tmp: PathBuf,
    blobs: BTreeMap<String, Vec<u8>>,
    trees: serde_json::Map<String, Value>,
    banks: BTreeMap<String, Bank>,
    counter: usize,
}

impl Workspace {
    fn materialize(&self, name: &str, dir: &Path) {
        let t = &self.trees[name];
        if let Some(base) = t.get("base").and_then(Value::as_str) {
            self.materialize(base, dir);
        }
        let list = |key: &str| t.get(key).and_then(Value::as_array).cloned().unwrap_or_default();
        for p in list("remove") {
            let path = dir.join(p.as_str().unwrap());
            if path.is_dir() {
                std::fs::remove_dir_all(&path).unwrap();
            } else if path.exists() {
                std::fs::remove_file(&path).unwrap();
            }
        }
        for p in list("dirs") {
            std::fs::create_dir_all(dir.join(p.as_str().unwrap())).unwrap();
        }
        for f in list("files") {
            let path = dir.join(f[0].as_str().unwrap());
            std::fs::create_dir_all(path.parent().unwrap()).unwrap();
            std::fs::write(&path, &self.blobs[f[1].as_str().unwrap()]).unwrap();
        }
        for p in list("chmod000") {
            use std::os::unix::fs::PermissionsExt;
            std::fs::set_permissions(dir.join(p.as_str().unwrap()), std::fs::Permissions::from_mode(0o000)).unwrap();
        }
    }

    /// 建立樹、執行 f，錯誤訊息中的樹根換成 `$ROOT`，之後刪除。
    fn with_tree(&mut self, name: &str, f: impl FnOnce(&str) -> Value) -> Value {
        self.counter += 1;
        let dir = self.tmp.join(format!("t{:04}", self.counter));
        self.materialize(name, &dir);
        let root = dir.to_str().unwrap().to_string();
        let mut res = f(&root);
        if let Some(Value::String(e)) = res.get_mut("err") {
            *e = e.replace(&root, "$ROOT");
        }
        let _ = std::process::Command::new("chmod").args(["-R", "u+rwx", &root]).status();
        std::fs::remove_dir_all(&dir).unwrap();
        res
    }
}

struct Real {
    flame2: Option<String>,
    assets: Option<String>,
}

fn clean_real(mut r: Value, root: &str) -> Value {
    if let Some(Value::String(e)) = r.get_mut("err") {
        *e = e.replace(root, "$REAL");
    }
    r
}

fn fdshap_result(r: Result<(Bank, Vec<u8>)>) -> Value {
    match r {
        Ok((bank, controls)) => {
            let mut v = bank_result(&Ok(bank));
            v["controls"] = json!(hex(&controls));
            v
        }
        Err(e) => bank_result(&Err(e)),
    }
}

fn tree_digest(root: &Path) -> (String, usize) {
    fn walk(base: &Path, dir: &Path, lines: &mut Vec<String>) {
        for entry in std::fs::read_dir(dir).unwrap() {
            let entry = entry.unwrap();
            let kind = entry.file_type().unwrap();
            if kind.is_dir() {
                walk(base, &entry.path(), lines);
            } else if kind.is_file() {
                let rel = entry.path().strip_prefix(base).unwrap().to_str().unwrap().to_string();
                lines.push(format!("{rel}\0{}\n", digest(&std::fs::read(entry.path()).unwrap())));
            }
        }
    }
    let mut lines = Vec::new();
    walk(root, root, &mut lines);
    lines.sort();
    (digest(lines.concat().as_bytes()), lines.len())
}

/// 依 op 用 Rust 重算結果；`None` 表示真實檔不可用而略過。
fn run(ws: &mut Workspace, real: &Real, real_bank: &mut Option<Bank>, op: &str, v: &Value) -> Option<Value> {
    Some(match op {
        "frame_index" => {
            let r = native_frame_index(i(v, "motion"), b(v, "force"), i(v, "idle"), i(v, "moving"));
            json!({"err": err_value(&r), "v": r.unwrap_or(0)})
        }
        "placement" => {
            let r = native_placement_offset(
                i(v, "x"),
                i(v, "y"),
                i(v, "cx"),
                i(v, "cy"),
                i(v, "pose"),
                i(v, "motion"),
                i(v, "shift"),
                b(v, "force"),
            );
            json!({"err": err_value(&r), "v": r.unwrap_or(0)})
        }
        "binary_tick" => {
            let state = NativeBinaryTickState { value: i(v, "value"), last_timer_tick: i(v, "last") };
            let r = advance_native_binary_tick(state, i(v, "raw"));
            let got = r.clone().unwrap_or(state);
            json!({"err": err_value(&r), "value": got.value, "last": got.last_timer_tick})
        }
        "sprite_cycles" => {
            let state =
                NativeMapSpriteCycleState { idle: i(v, "idle"), moving: i(v, "moving"), last_timer_tick: i(v, "last") };
            let got = advance_native_map_sprite_cycles(state, i(v, "raw"));
            json!({"idle": got.idle, "moving": got.moving, "last": got.last_timer_tick})
        }
        "cursor_info" => {
            let r = native_terrain_cursor_info_for_cell(
                i(v, "tile_word") as u16,
                i(v, "event_word") as u16,
                &bytes_of(v, "controls"),
            );
            let info = r.clone().unwrap_or_default();
            json!({"err": err_value(&r), "tile": info.tile, "event_low": info.event_low, "control": hex(&info.control)})
        }
        "hud_layout" => {
            let r = native_map_hud_layout_for(i(v, "anchor"), i(v, "stride"));
            let l = r.clone().unwrap_or_default();
            json!({"err": err_value(&r), "layout": [l.frame, l.terrain, l.ap, l.dp, l.unit, l.hp]})
        }
        "hud_unit_frame" => {
            let r = native_map_hud_unit_frame_index(i(v, "group"), i(v, "state"));
            json!({"err": err_value(&r), "v": r.unwrap_or(0)})
        }
        "command_bg" => {
            let targets: Vec<_> = v["targets"]
                .as_array()
                .unwrap()
                .iter()
                .map(|t| NativeCommandBackgroundTarget {
                    gate: t[0].as_bool().unwrap(),
                    control: unhex(t[1].as_str().unwrap()).try_into().unwrap(),
                })
                .collect();
            json!({"v": native_command_background_selector(u8_(v, "initial"), &targets)})
        }
        "fg_eligible" => {
            json!({"v": native_foreground_redraw_eligible(b(v, "inactive"), u8_(v, "unit7"), u8_(v, "race"), u8_(v, "class"))})
        }
        "fg_cells" => {
            let (cells, n) = native_foreground_redraw_cells(i(v, "x"), i(v, "y"), u8_(v, "pose"), i(v, "motion"));
            json!({"n": n, "cells": cells.iter().map(|c| [c.x, c.y]).collect::<Vec<_>>()})
        }
        "terrain_frame" => {
            let r = native_terrain_frame_index(i(v, "tile"), u8_(v, "flags"), i(v, "flip"), i(v, "cycle"));
            json!({"err": err_value(&r), "v": r.unwrap_or(0)})
        }
        "fg_frame" => {
            let r = native_foreground_frame_index(i(v, "tile"), u8_(v, "flags"), i(v, "flip"));
            let (fv, present) = r.clone().unwrap_or((0, false));
            json!({"err": err_value(&r), "v": fv, "present": present})
        }
        "source_groups" => {
            json!({"v": (i(v, "from")..=i(v, "to")).map(native_source_group_has_pixels).collect::<Vec<_>>()})
        }
        "constants" => json!({"v": [NATIVE_SIZE, NATIVE_MAP_STRIDE, NATIVE_UNIT_ORIGIN_BYTES, NATIVE_MOTION_PIXELS,
            SEPARATED_SPRITE_COUNT, NATIVE_SOURCE_GROUP_COUNT, SEPARATED_FDSHAP_MAP_COUNT]}),
        "selector_cache" => {
            let mut cache = NativeSelectorCache::default();
            let mut clone: Option<NativeSelectorCache> = None;
            let mut results = Vec::new();
            for step in v["ops"].as_array().unwrap() {
                let arg = || step[1].as_i64().unwrap();
                results.push(match step[0].as_str().unwrap() {
                    "slot" => {
                        let r = cache.slot_for(arg());
                        json!({"v": r.clone().unwrap_or(0), "err": err_value(&r)})
                    }
                    "key" => {
                        let r = cache.key_for_slot(arg());
                        json!({"v": r.clone().unwrap_or(0), "err": err_value(&r)})
                    }
                    "clone" => {
                        clone = Some(cache.clone());
                        json!({})
                    }
                    "cslot" => {
                        let r = clone.as_mut().unwrap().slot_for(arg());
                        json!({"v": r.clone().unwrap_or(0), "err": err_value(&r)})
                    }
                    "ckeys" => json!({"v": hex(&clone.as_ref().unwrap().keys())}),
                    "keys" => json!({"v": hex(&cache.keys())}),
                    "from_keys" => {
                        let r = NativeSelectorCache::from_keys(&unhex(step[1].as_str().unwrap()));
                        match r {
                            Ok(c) => json!({"err": null, "keys": hex(&c.keys())}),
                            Err(e) => json!({"err": e.to_string()}),
                        }
                    }
                    other => panic!("未知的 selector op {other}"),
                });
            }
            json!({"results": results})
        }
        "sprite_for" => {
            let bank = ident_bank(i(v, "bank_len") as usize);
            sprite_ident(bank.sprite_for(i(v, "group"), i(v, "pose"), i(v, "cycle")))
        }
        "sprite_for_native_slot" => {
            let bank = ident_bank(i(v, "bank_len") as usize);
            let cache = NativeSelectorCache::from_keys(&bytes_of(v, "keys")).unwrap();
            sprite_ident(bank.sprite_for_native_slot(&cache, i(v, "slot"), i(v, "pose"), i(v, "cycle")))
        }
        "sprite_blit" => {
            let s = sprite_from(&v["sprite"]);
            let before = bytes_of(v, "dst");
            let mut dst = before.clone();
            let lut = bytes_of(v, "lut");
            let (stride, x, y, offset) = (i(v, "stride"), i(v, "x"), i(v, "y"), i(v, "offset"));
            let (err, panicked) = protect(|| match v["variant"].as_str().unwrap() {
                "at" => s.blit_at(&mut dst, stride, x, y),
                "band" => s.blit_palette_band(&mut dst, stride, x, y),
                "flags" => s.blit_for_native_flags(&mut dst, stride, x, y, u8_(v, "flags")),
                "flags_offset" => s.blit_for_native_flags_at_offset(&mut dst, stride, offset, u8_(v, "flags")),
                "lut" => s.blit_lut(&mut dst, stride, x, y, &lut),
                "lut_t" => s.blit_lut_transparent(&mut dst, stride, x, y, &lut),
                "lut_t_offset" => s.blit_lut_transparent_at_offset(&mut dst, stride, offset, &lut),
                "const" => s.blit_constant_mask_at(&mut dst, stride, x, y, u8_(v, "index")),
                _ => s.blit_native_damage_blend_at(&mut dst, stride, x, y, i(v, "blend"), u8_(v, "raw_base")),
            });
            json!({"err": err, "panic": panicked, "diff": diff(&before, &dst)})
        }
        "copy_region" => {
            let before = bytes_of(v, "dst");
            let mut dst = before.clone();
            let src = bytes_of(v, "src");
            let (err, panicked) = protect(|| {
                copy_native_indexed_region(
                    &mut dst,
                    i(v, "dst_stride"),
                    &src,
                    i(v, "src_stride"),
                    i(v, "width"),
                    i(v, "height"),
                )
            });
            json!({"err": err, "panic": panicked, "diff": diff(&before, &dst)})
        }
        "terrain_cell" | "fg_cell" => {
            let bank = &ws.banks[v["bank"].as_str().unwrap()];
            let before = bytes_of(v, "dst");
            let mut dst = before.clone();
            let lut = bytes_of(v, "lut");
            let (stride, x, y, tile) = (i(v, "stride"), i(v, "x"), i(v, "y"), i(v, "tile"));
            let (flags, mode, flip) = (u8_(v, "flags"), u8_(v, "blit_mode"), i(v, "flip"));
            let (err, panicked) = protect(|| {
                if op == "fg_cell" {
                    bank.blit_native_foreground_cell(&mut dst, stride, x, y, tile, flags, mode, flip, &lut)
                } else {
                    bank.blit_native_terrain_cell(&mut dst, stride, x, y, tile, flags, mode, flip, i(v, "cycle"), &lut)
                }
            });
            json!({"err": err, "panic": panicked, "diff": diff(&before, &dst)})
        }
        "terrain_region" => {
            let bank = &ws.banks[v["bank"].as_str().unwrap()];
            let before = bytes_of(v, "dst");
            let mut dst = before.clone();
            let (lut, cells, controls) = (bytes_of(v, "lut"), cells_from(&v["cells"]), bytes_of(v, "controls"));
            let (err, panicked) = protect(|| {
                bank.blit_native_terrain_region(
                    &mut dst,
                    i(v, "stride"),
                    i(v, "dst_x"),
                    i(v, "dst_y"),
                    i(v, "map_width"),
                    &cells,
                    &controls,
                    i(v, "map_x"),
                    i(v, "map_y"),
                    i(v, "width"),
                    i(v, "height"),
                    i(v, "flip"),
                    i(v, "cycle"),
                    &lut,
                )
            });
            json!({"err": err, "panic": panicked, "diff": diff(&before, &dst)})
        }
        "unit_layer" => {
            let bank = &ws.banks[v["bank"].as_str().unwrap()];
            let cache = NativeSelectorCache::from_keys(&bytes_of(v, "keys")).unwrap();
            let before = vec![0u8; i(v, "dst_len") as usize];
            let mut dst = before.clone();
            let units: Vec<_> = v["units"]
                .as_array()
                .unwrap()
                .iter()
                .map(|u| NativeUnitLayerEntry {
                    x: u[0].as_i64().unwrap(),
                    y: u[1].as_i64().unwrap(),
                    slot: u[2].as_i64().unwrap(),
                    pose: u[3].as_i64().unwrap(),
                    motion_offset: u[4].as_i64().unwrap(),
                    flags: u[5].as_u64().unwrap() as u8,
                    force_base: u[6].as_bool().unwrap(),
                    inactive: u[7].as_bool().unwrap(),
                })
                .collect();
            let (err, panicked) = protect(|| {
                bank.blit_native_unit_layer_with_base_shift(
                    &mut dst,
                    i(v, "stride"),
                    &cache,
                    &units,
                    i(v, "cx"),
                    i(v, "cy"),
                    i(v, "vx"),
                    i(v, "vy"),
                    i(v, "idle"),
                    i(v, "moving"),
                    i(v, "shift"),
                    i(v, "base_shift"),
                )
            });
            json!({"err": err, "panic": panicked, "diff": diff(&before, &dst)})
        }
        "fg_layer" => {
            let bank = &ws.banks[v["bank"].as_str().unwrap()];
            let before = vec![0u8; i(v, "dst_len") as usize];
            let mut dst = before.clone();
            let units: Vec<_> = v["units"]
                .as_array()
                .unwrap()
                .iter()
                .map(|u| NativeForegroundLayerEntry {
                    x: u[0].as_i64().unwrap(),
                    y: u[1].as_i64().unwrap(),
                    pose: u[2].as_u64().unwrap() as u8,
                    motion_offset: u[3].as_i64().unwrap(),
                    inactive: u[4].as_bool().unwrap(),
                    unit7: u[5].as_u64().unwrap() as u8,
                    race: u[6].as_u64().unwrap() as u8,
                    class: u[7].as_u64().unwrap() as u8,
                })
                .collect();
            let (lut, cells, controls) = (bytes_of(v, "lut"), cells_from(&v["cells"]), bytes_of(v, "controls"));
            let (err, panicked) = protect(|| {
                bank.blit_native_foreground_layer(
                    &mut dst,
                    i(v, "stride"),
                    &units,
                    i(v, "map_width"),
                    &cells,
                    &controls,
                    i(v, "cx"),
                    i(v, "cy"),
                    i(v, "vx"),
                    i(v, "vy"),
                    i(v, "flip"),
                    &lut,
                )
            });
            json!({"err": err, "panic": panicked, "diff": diff(&before, &dst)})
        }
        "parse" => bank_result(&parse(&bytes_of(v, "raw"))),
        "decode_file" => {
            if let Some(name) = v.get("real").and_then(Value::as_str) {
                let root = real.flame2.as_ref()?;
                let r = decode_file(format!("{root}/{name}"));
                let out = clean_real(bank_result(&r), root);
                *real_bank = r.ok();
                out
            } else {
                let tmp = ws.tmp.to_str().unwrap().to_string();
                let rel = v["tmp"].as_str().unwrap();
                let path = if rel.is_empty() { tmp.clone() } else { format!("{tmp}/{rel}") };
                let r = decode_file(&path);
                json!({"err": err_value(&r).as_str().map(|e| e.replace(&tmp, "$TMP"))})
            }
        }
        "real_sprite_for" => {
            let bank = real_bank.as_ref()?;
            let r = bank.sprite_for(i(v, "group"), i(v, "pose"), i(v, "cycle"));
            let mut out = serde_json::Map::new();
            out.insert("err".into(), err_value(&r));
            if let Ok(s) = r {
                out.insert("digest".into(), json!(sprite_digest(s)));
            }
            Value::Object(out)
        }
        "load_bank" => {
            if let Some(rel) = v.get("real").and_then(Value::as_str) {
                let root = real.assets.as_ref()?;
                clean_real(bank_result(&load_separated_bank(format!("{root}/{rel}"))), root)
            } else {
                let suffix = v["suffix"].as_str().unwrap().to_string();
                ws.with_tree(v["tree"].as_str().unwrap(), |root| {
                    bank_result(&load_separated_bank(format!("{root}{suffix}")))
                })
            }
        }
        "load_bank_empty_root" => json!({"err": err_value(&load_separated_bank(""))}),
        "load_fdshap" => {
            let map_index = i(v, "map_index");
            if let Some(rel) = v.get("real").and_then(Value::as_str) {
                let root = real.assets.as_ref()?;
                clean_real(fdshap_result(load_separated_fdshap_bank(format!("{root}/{rel}"), map_index)), root)
            } else {
                let suffix = v["suffix"].as_str().unwrap().to_string();
                ws.with_tree(v["tree"].as_str().unwrap(), |root| {
                    fdshap_result(load_separated_fdshap_bank(format!("{root}{suffix}"), map_index))
                })
            }
        }
        "load_fdshap_empty_root" => {
            json!({"err": err_value(&load_separated_fdshap_bank("", i(v, "map_index")).map(|_| ()))})
        }
        other => panic!("未知的 op {other}"),
    })
}

/// 回傳第一個不同的 JSON 路徑，方便定位。
fn first_diff(path: &str, want: &Value, got: &Value) -> Option<String> {
    match (want, got) {
        (Value::Object(a), Value::Object(b)) => {
            for key in a.keys().chain(b.keys()) {
                if a.contains_key(key) != b.contains_key(key) {
                    return Some(format!(
                        "{path}.{key}: present go={} rust={}",
                        a.contains_key(key),
                        b.contains_key(key)
                    ));
                }
                if let Some(d) = first_diff(&format!("{path}.{key}"), &a[key], &b[key]) {
                    return Some(d);
                }
            }
            None
        }
        (Value::Array(a), Value::Array(b)) if a.len() == b.len() => {
            a.iter().zip(b).enumerate().find_map(|(i, (x, y))| first_diff(&format!("{path}[{i}]"), x, y))
        }
        _ if want == got => None,
        _ => Some(format!("{path}: go={want} rust={got}")),
    }
}

/// 格式錯誤的 JSON／PNG／檔案系統輸入：只比接受或拒收。
fn accept_reject_only(kind: &str) -> bool {
    const MALFORMED: [&str; 13] = [
        "json-syntax",
        "json-type",
        "json-top",
        "json-fuzz",
        "json-fold",
        "json-dup",
        "json-escape",
        "json-unknown",
        "fs-",
        "png-fuzz",
        "fdicon-json",
        "fdicon-file",
        "fdicon-fs",
    ];
    // 由 Go `image/png` 編碼器或本工具依 PNG 規範產生的檔案逐欄比對。
    const WELL_FORMED_PNG: [&str; 27] = [
        "png-frame-valid",
        "png-frame-filters",
        "png-frame-interlaced",
        "png-frame-interlaced-filters",
        "png-frame-plte200",
        "png-frame-depth4",
        "png-frame-size25",
        "png-frame-size23",
        "png-frame-rgba",
        "png-frame-rgb",
        "png-frame-rgb-plte",
        "png-frame-gray",
        "png-frame-ga16",
        "png-frame-rgb16-trns",
        "png-frame-gray16",
        "png-frame-rgba16-interlaced",
        "png-frame-stored",
        "png-frame-level1",
        "png-frame-level9-filters",
        "png-frame-huffman-only",
        "png-frame-split-1",
        "png-frame-split-7",
        "png-frame-split-zero",
        "png-frame-1x1-interlaced",
        "png-frame-3x2-interlaced",
        "png-frame-unknown-anc-post",
        "png-mask-",
    ];
    if kind.starts_with("png-remap-")
        || WELL_FORMED_PNG.iter().any(|k| kind == *k || (k.ends_with('-') && kind.starts_with(k)))
    {
        return false;
    }
    kind.starts_with("png-") || MALFORMED.iter().any(|k| kind.starts_with(k))
}

/// 連接受與否都和 Go 不同的案例種類：(kind, 刻意不照做的原因)。
/// 一律是刻意不照做 Go 標準函式庫的部分；fdicon 自己的檢查沒有差異。
const KNOWN_DIFFERENCES: &[(&str, &str)] = &[
    ("json-fold", "Go 的欄位名稱不分大小寫（含 U+212A、U+017F），這裡只比對原字"),
    ("json-dup", "重複鍵時 Go 沿用既有 slice 元素並合併欄位，serde_json 以後者整個取代"),
    ("json-type-first", "Go 在重複鍵覆蓋之前仍回報先前的型別錯誤，serde_json 只看最後的值"),
    ("json-unknown", "未知欄位裡的無效 UTF-8 或超過 128 層的巢狀，Go 略過，serde_json 拒收"),
    ("json-syntax", "字串裡的無效 UTF-8，Go 換成 U+FFFD，serde_json 拒收"),
    ("png-frame-plte16-idx255", "Go 依像素最大索引延伸調色盤長度（索引超出 PLTE，規範外）"),
    ("png-frame-interlaced-idx255", "Go 依像素最大索引延伸調色盤長度（索引超出 PLTE，規範外）"),
    ("png-frame-plte16-trns256", "Go 依 tRNS 項數延伸調色盤長度（tRNS 比 PLTE 長，規範外），這裡只取 PLTE 項數"),
    ("png-frame-depth2-trns256", "Go 依 tRNS 項數延伸調色盤長度（tRNS 比 PLTE 長，規範外），這裡只取 PLTE 項數"),
    ("png-frame-depth1-trns256", "Go 依 tRNS 項數延伸調色盤長度（tRNS 比 PLTE 長，規範外），這裡只取 PLTE 項數"),
    (
        "png-frame-depth1-trns256-interlaced",
        "Go 依 tRNS 項數延伸調色盤長度（tRNS 比 PLTE 長，規範外），這裡只取 PLTE 項數",
    ),
    ("png-frame-trns-before-plte", "png crate 不檢查 tRNS 的區塊順序"),
    ("png-frame-trns-after-idat", "png crate 不檢查 tRNS 的區塊順序"),
    ("png-frame-trns-dup", "png crate 不拒收重複的 tRNS"),
    ("png-frame-trns-long", "png crate 不檢查 tRNS 長度上限"),
    ("png-frame-fdict-1", "Go 接受字典 ID 等於空字典 Adler-32（1）的 FDICT zlib 串流"),
    ("png-frame-unknown-crit", "Go 略過未知的關鍵區塊，png crate 拒收"),
    ("png-frame-unknown-badcrc", "png crate 不驗未知區塊的 CRC"),
    ("png-frame-trunc-adler", "png crate 不要求完整的 Adler-32"),
    ("png-frame-trunc-adler-full", "png crate 不要求 Adler-32"),
    ("png-frame-extra-row", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-frame-extra-byte", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-frame-extra-row-bad-adler", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-frame-junk-big", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-frame-junk-exact-4096", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-frame-junk-4097", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-frame-stored-junk-4097", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-frame-stored-junk-big", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
    ("png-fuzz-crcfix", "Go 拒收 zlib 串流之後的多餘資料（too much pixel data），png crate 忽略"),
];

/// 覆蓋統計用：把錯誤訊息中的數字與十六進位換成 N，路徑留下樣板。
fn classify(result: &Value) -> String {
    if result.get("panic").and_then(Value::as_bool) == Some(true) {
        return "panic".to_string();
    }
    match result.get("err") {
        Some(Value::String(e)) => {
            let mut out = String::new();
            let mut last_digit = false;
            for ch in e.chars() {
                if ch.is_ascii_digit() {
                    if !last_digit {
                        out.push('N');
                    }
                    last_digit = true;
                } else {
                    out.push(ch);
                    last_digit = false;
                }
            }
            out
        }
        Some(Value::Null) => "ok".to_string(),
        _ => "value".to_string(),
    }
}

#[test]
fn matches_frozen_go_golden() {
    let required = std::env::var("FD2_REQUIRE_GOLDEN").is_ok();
    let Ok(path) = std::env::var("FD2_GOLDEN_FDICON") else {
        assert!(!required, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_FDICON");
        eprintln!("略過：未設定 FD2_GOLDEN_FDICON");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-fdicon/1");

    let real = Real {
        flame2: std::env::var("FD2_FDICON_REAL_FLAME2").ok(),
        assets: std::env::var("FD2_FDICON_REAL_ASSETS").ok(),
    };
    if required {
        assert!(real.flame2.is_some(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_FDICON_REAL_FLAME2");
        assert!(real.assets.is_some(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_FDICON_REAL_ASSETS");
    }
    if let (Some(dir), Some(rec)) = (&real.flame2, doc["real"].get("fdicon_b24")) {
        let got = digest(&std::fs::read(format!("{dir}/{}", rec["path"].as_str().unwrap())).expect("讀 FDICON.B24"));
        assert_eq!(got, rec["sha256"].as_str().unwrap(), "FDICON.B24 與黃金檔記錄的雜湊不同");
    }
    if let Some(dir) = &real.assets {
        for key in ["fdicon_pack", "fdshap_pack"] {
            let Some(rec) = doc["real"].get(key) else { continue };
            let (got, files) = tree_digest(&Path::new(dir).join(rec["path"].as_str().unwrap()));
            assert_eq!(files as u64, rec["files"].as_u64().unwrap(), "{key} 檔案數不同");
            assert_eq!(got, rec["tree_sha256"].as_str().unwrap(), "{key} 檔案清單摘要不同");
        }
    }

    let tmp = std::env::temp_dir().join(format!("fd2-golden-fdicon-{}", std::process::id()));
    std::fs::create_dir_all(&tmp).unwrap();
    let mut ws = Workspace {
        tmp: tmp.clone(),
        blobs: doc["blobs"]
            .as_object()
            .unwrap()
            .iter()
            .map(|(k, z)| {
                let data = unhex(z.as_str().unwrap());
                assert_eq!(&digest(&data), k, "blob 內容與雜湊不符");
                (k.clone(), data)
            })
            .collect(),
        trees: doc["trees"].as_object().unwrap().clone(),
        banks: doc["banks"]
            .as_object()
            .unwrap()
            .iter()
            .map(|(k, rows)| (k.clone(), Bank { sprites: rows.as_array().unwrap().iter().map(sprite_from).collect() }))
            .collect(),
        counter: 0,
    };

    let cases = doc["cases"].as_array().unwrap();
    assert!(!cases.is_empty());
    let mut failures = Vec::new();
    let mut skipped = 0;
    let mut kinds = BTreeMap::<String, usize>::new();
    let mut coverage = BTreeMap::<String, BTreeMap<String, usize>>::new();
    let mut real_bank = None;
    let mut loose = BTreeMap::<String, usize>::new();
    let mut known = Vec::new();
    for case in cases {
        let index = case["index"].as_u64().unwrap();
        let op = case["op"].as_str().unwrap();
        *kinds.entry(case["kind"].as_str().unwrap().to_string()).or_default() += 1;
        let Some(got) = run(&mut ws, &real, &mut real_bank, op, &case["input"]) else {
            skipped += 1;
            continue;
        };
        *coverage.entry(op.to_string()).or_default().entry(classify(&case["result"])).or_default() += 1;
        let kind = case["kind"].as_str().unwrap();
        let want = &case["result"];
        let mismatch = if accept_reject_only(kind) {
            let (go_ok, rust_ok) = (want["err"].is_null(), got["err"].is_null());
            if go_ok != rust_ok {
                Some(format!("result.err: 接受與否不同 go={} rust={}", want["err"], got["err"]))
            } else if go_ok {
                first_diff("result", want, &got)
            } else {
                *loose.entry(kind.to_string()).or_default() += 1;
                None
            }
        } else {
            first_diff("result", want, &got)
        };
        if let Some(d) = mismatch {
            if let Some((_, reason)) = KNOWN_DIFFERENCES.iter().find(|(k, _)| *k == kind) {
                known.push(format!("case {index} ({kind} {op})：刻意不照做，{reason}；{d}"));
            } else {
                failures.push(format!("case {index} ({kind} {op}): {d}"));
            }
        }
    }
    let _ = std::fs::remove_dir_all(&tmp);
    eprintln!("黃金案例 {}（略過 {skipped}）：{kinds:?}", cases.len());
    eprintln!("只比接受／拒收且雙方都拒收：{}", loose.values().sum::<usize>());
    eprintln!("已知差異 {} 個：", known.len());
    for k in &known {
        eprintln!("  {k}");
    }
    for (op, paths) in &coverage {
        eprintln!("覆蓋 {op}:");
        for (path, n) in paths {
            eprintln!("  {n:5}  {path}");
        }
    }
    assert!(!required || skipped == 0, "FD2_REQUIRE_GOLDEN=1 但有 {skipped} 個真實檔案例未執行");
    assert!(failures.is_empty(), "{} 個案例不符：\n{}", failures.len(), failures.join("\n"));
}
