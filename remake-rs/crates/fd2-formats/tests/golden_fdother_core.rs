//! 與凍結 Go 版 `remake/cmd/fd2-golden-fdother-core` 的黃金差分（fdother 第一組：核心與分離載入器）。
//!
//! 由 `tools/rust_golden_fdother_core.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_FDOTHER_CORE`。真實原版檔
//! 與分離素材包不在黃金檔裡：`FD2_FLAME2_DIR` 指向原版 FLAME2 目錄，`FD2_ASSET_PACK` 指向分離素材包
//! 根目錄，兩者都先核對黃金檔記錄的 SHA-256。設定 `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實檔目錄
//! 缺少即失敗，避免把「沒跑到」當成通過。
//!
//! 每個案例的 `compare` 欄決定比對方式：
//! - `exact`：結果逐欄相同，錯誤文字逐字相同。
//! - `err_prefix`：Go 錯誤包了標準函式庫錯誤；Rust 必須同樣拒收，`FdotherError::go_prefix()` 必須等於
//!   黃金檔的 `prefix`，內層文字不比，其餘欄位逐欄比。
//! - `accept_reject`：格式錯誤的 PNG，只比接受或拒收；都接受時結果仍逐欄比。
//!
//! 連接受與否都和 Go 不同的案例種類列在 [`KNOWN_DIFFERENCES`]，一律是刻意不照做 Go 標準函式庫的部分。

use fd2_formats::fdicon::{Bank, Sprite};
use fd2_formats::fdother::*;
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};

fn digest(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}

fn pattern(n: i64, seed: i64) -> Vec<u8> {
    (0..n.max(0)).map(|i| (i * 13 + seed) as u8).collect()
}

fn lut_of(n: i64, mul: i64, add: i64) -> Vec<u8> {
    (0..n).map(|i| (i * mul + add) as u8).collect()
}

fn i(v: &Value, key: &str) -> i64 {
    v[key].as_i64().unwrap_or_else(|| panic!("欄位 {key} 不是整數：{}", v[key]))
}

fn s<'a>(v: &'a Value, key: &str) -> &'a str {
    v[key].as_str().unwrap_or_else(|| panic!("欄位 {key} 不是字串：{}", v[key]))
}

fn bytes_of(v: &Value) -> Vec<u8> {
    match v {
        Value::Null => Vec::new(),
        x => unhex(x.as_str().unwrap()),
    }
}

/// 一個案例的結果與 Rust 端錯誤的 Go 前綴。
struct Out {
    v: Map<String, Value>,
    prefix: Option<String>,
}

fn out(v: Value) -> Out {
    let Value::Object(v) = v else { panic!("結果必須是物件") };
    Out { v, prefix: None }
}

/// 對應 Go `protectErr`：加上 `err` 與 `panic`。
fn finish<T>(v: Value, r: &Result<T>) -> Out {
    let mut o = out(v);
    o.v.insert("err".into(), r.as_ref().err().map_or(Value::Null, |e| e.to_string().into()));
    o.v.insert("panic".into(), false.into());
    o.prefix = r.as_ref().err().map(FdotherError::go_prefix);
    o
}

/// 對應 Go 沒包 protect 的路徑 API：只加 `err`。
fn finish_plain<T>(v: Value, r: &Result<T>) -> Out {
    let mut o = finish(v, r);
    o.v.remove("panic");
    o
}

/// 對應 Go `protectErr` 的 panic 分支。
fn protect(f: impl FnOnce() -> Out) -> Out {
    let previous = std::panic::take_hook();
    std::panic::set_hook(Box::new(|_| {}));
    let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(f));
    std::panic::set_hook(previous);
    r.unwrap_or_else(|_| out(json!({"err": null, "panic": true})))
}

const LAYER_LIMIT: i64 = 1 << 22;

fn frame_info(f: &Frame) -> Value {
    let mut m = json!({"x": f.x, "y": f.y, "w": f.width, "h": f.height,
        "pixels": digest(&f.pixels), "indexed": digest(&f.indexed), "mask": digest(&f.mask)});
    if f.width > 0 && f.height > 0 && f.width.wrapping_mul(f.height) <= LAYER_LIMIT {
        let layers = protect(|| {
            let r = f.indexed_layers();
            let v = match &r {
                Ok((ix, mk)) => json!({"sha": digest(&[ix.as_slice(), mk.as_slice()].concat())}),
                Err(_) => json!({}),
            };
            let mut o = finish(v, &r);
            o.prefix = None;
            o
        });
        m["layers"] = Value::Object(layers.v);
    }
    m
}

fn lmi_info(e: &LMI1Entry) -> Value {
    json!([e.width, e.height, digest(&e.pixels)])
}

fn raw_info(c: &RawCell) -> Value {
    json!([c.width, c.height, digest(&c.pixels)])
}

fn sprite_digest(s: &Sprite) -> String {
    let mut h = Sha256::new();
    h.update(&s.pixels);
    h.update(&s.mask);
    h.update(&s.remap_mask);
    h.finalize().iter().map(|x| format!("{x:02x}")).collect()
}

fn bank_info(b: &Bank) -> Value {
    b.sprites.iter().map(sprite_digest).collect()
}

fn palette_digest(p: &[Nrgba]) -> String {
    let mut bytes = Vec::new();
    for c in p {
        let (r, g, b, a) = c.rgba();
        for v in [r, g, b, a] {
            bytes.extend_from_slice(&v.to_le_bytes());
        }
    }
    digest(&bytes)
}

fn frame_from(d: &Value) -> Frame {
    let mut f = Frame::default();
    f.x = i(d, "x");
    f.y = i(d, "y");
    f.width = i(d, "w");
    f.height = i(d, "h");
    f.pixels = d.get("pixels").map(bytes_of).unwrap_or_default();
    f.indexed = d.get("indexed").map(bytes_of).unwrap_or_default();
    f.mask = d.get("mask").map(bytes_of).unwrap_or_default();
    f
}

fn entry_from(v: &Value) -> LMI1Entry {
    LMI1Entry { width: v[0].as_i64().unwrap(), height: v[1].as_i64().unwrap(), pixels: bytes_of(&v[2]) }
}

fn cell_from(v: &Value) -> RawCell {
    RawCell { width: v[0].as_i64().unwrap(), height: v[1].as_i64().unwrap(), pixels: bytes_of(&v[2]) }
}

fn frames_info(frames: &[Frame]) -> Value {
    frames.iter().map(frame_info).collect()
}

// ---- 原始資料解析 ----

fn parse_op(op: &str, data: &[u8], index: i64) -> Out {
    match op {
        "parse_frames" => {
            let r = parse_frames(data);
            let v = r.as_ref().map_or(json!({}), |f| json!({"frames": frames_info(f)}));
            finish(v, &r)
        }
        "parse_single_frame" | "parse_lmi1_frame_entry" => {
            let r =
                if op == "parse_single_frame" { parse_single_frame(data) } else { parse_lmi1_frame_entry(data, index) };
            let v = r.as_ref().map_or(json!({}), |f| json!({"frame": frame_info(f)}));
            finish(v, &r)
        }
        "parse_lmi1" => {
            let r = parse_lmi1(data);
            let v = r.as_ref().map_or(json!({}), |e| json!({"entries": e.iter().map(lmi_info).collect::<Vec<_>>()}));
            finish(v, &r)
        }
        "parse_opaque_run_cell" | "parse_lmi1_opaque_entry" => {
            let r = if op == "parse_opaque_run_cell" {
                parse_opaque_run_cell(data)
            } else {
                parse_lmi1_opaque_entry(data, index)
            };
            let v = r.as_ref().map_or(json!({}), |e| json!({"entry": lmi_info(e)}));
            finish(v, &r)
        }
        "parse_raw_cell" | "parse_lmi1_raw_entry" => {
            let r = if op == "parse_raw_cell" { parse_raw_cell(data) } else { parse_lmi1_raw_entry(data, index) };
            let v = r.as_ref().map_or(json!({}), |c| json!({"cell": raw_info(c)}));
            finish(v, &r)
        }
        "parse_raw_cell_bank" => {
            let r = parse_raw_cell_bank(data);
            let v = r.as_ref().map_or(json!({}), |c| json!({"cells": c.iter().map(raw_info).collect::<Vec<_>>()}));
            finish(v, &r)
        }
        "parse_lut_bank" => {
            let r = parse_lut_bank(data);
            let v = match &r {
                Ok(luts) => {
                    let mut v = json!({"luts": luts.iter().map(|l| digest(l)).collect::<Vec<_>>()});
                    match native_unit_present_bridge_lut(luts) {
                        Ok(b) => v["bridge"] = digest(&b).into(),
                        Err(e) => v["bridge_err"] = e.to_string().into(),
                    }
                    v
                }
                Err(_) => json!({}),
            };
            finish(v, &r)
        }
        "parse_vga_palette" => {
            let r = parse_vga_palette(data);
            let v = r.as_ref().map_or(json!({}), |p| json!({"palette": palette_digest(p)}));
            finish(v, &r)
        }
        _ => panic!("未知解析器 {op}"),
    }
}

// ---- 執行環境 ----

struct Env {
    flame2: Option<PathBuf>,
    assets: Option<PathBuf>,
    tmp: PathBuf,
    blobs: BTreeMap<String, Vec<u8>>,
    trees: Map<String, Value>,
    banks: BTreeMap<String, Bank>,
    archives: BTreeMap<String, Vec<u8>>,
}

impl Env {
    fn archive(&mut self, name: &str) -> Option<&Vec<u8>> {
        let dir = self.flame2.clone()?;
        Some(self.archives.entry(name.to_string()).or_insert_with(|| std::fs::read(dir.join(name)).unwrap()))
    }

    fn mirror(&self, real_rel: &str, tree_rel: &str, dir: &Path) {
        let src = self.assets.as_ref().unwrap().join(real_rel);
        let dst = if tree_rel.is_empty() { dir.to_path_buf() } else { dir.join(tree_rel) };
        if !src.is_dir() {
            std::fs::create_dir_all(dst.parent().unwrap()).unwrap();
            std::os::unix::fs::symlink(&src, &dst).unwrap();
            return;
        }
        std::fs::create_dir_all(&dst).unwrap();
        for entry in std::fs::read_dir(&src).unwrap() {
            let entry = entry.unwrap();
            let name = entry.file_name();
            let rel = format!("{real_rel}/{}", name.to_str().unwrap());
            let sub = dst.join(&name);
            if entry.file_type().unwrap().is_dir() {
                std::fs::create_dir_all(&sub).unwrap();
                self.mirror(&rel, sub.strip_prefix(dir).unwrap().to_str().unwrap(), dir);
            } else {
                std::os::unix::fs::symlink(entry.path(), &sub).unwrap();
            }
        }
    }

    fn materialize(&self, name: &str, dir: &Path) {
        let t = &self.trees[name];
        std::fs::create_dir_all(dir).unwrap();
        let list = |key: &str| t.get(key).and_then(Value::as_array).cloned().unwrap_or_default();
        for l in list("links") {
            self.mirror(l[0].as_str().unwrap(), l[1].as_str().unwrap(), dir);
        }
        for p in list("remove") {
            let path = dir.join(p.as_str().unwrap());
            if path.is_dir() && !path.is_symlink() {
                std::fs::remove_dir_all(&path).unwrap();
            } else if path.symlink_metadata().is_ok() {
                std::fs::remove_file(&path).unwrap();
            }
        }
        for p in list("dirs") {
            std::fs::create_dir_all(dir.join(p.as_str().unwrap())).unwrap();
        }
        for f in list("files") {
            let path = dir.join(f[0].as_str().unwrap());
            let _ = std::fs::remove_file(&path);
            std::fs::create_dir_all(path.parent().unwrap()).unwrap();
            std::fs::write(&path, &self.blobs[f[1].as_str().unwrap()]).unwrap();
        }
    }
}

// ---- 分離載入器 ----

fn string_keyed<T>(m: &BTreeMap<i64, T>, info: impl Fn(&T) -> Value) -> Value {
    Value::Object(m.iter().map(|(k, v)| (k.to_string(), info(v))).collect())
}

fn run_loader(op: &str, root: &Path, a: &Value) -> Out {
    match op {
        "load_single_frame" => {
            let r = load_separated_single_frame(root, s(a, "source"), i(a, "resource"));
            finish(r.as_ref().map_or(json!({}), |f| json!({"frame": frame_info(f)})), &r)
        }
        "load_nested_single_frame" => {
            let r = load_separated_nested_single_frame(root, i(a, "resource"), i(a, "nested"));
            finish(r.as_ref().map_or(json!({}), |f| json!({"frame": frame_info(f)})), &r)
        }
        "load_palette" => {
            let r = load_separated_fdother_palette(root, i(a, "resource"));
            finish(r.as_ref().map_or(json!({}), |(d, p)| json!({"dac": digest(d), "palette": palette_digest(p)})), &r)
        }
        "load_item_panel" => {
            let r = load_separated_item_panel_entries(root);
            let v = r.as_ref().map_or(json!({}), |e| {
                json!({"opaque": string_keyed(&e.opaque, lmi_info), "raw": string_keyed(&e.raw, raw_info),
                    "frames": string_keyed(&e.frames, frame_info)})
            });
            finish(v, &r)
        }
        "load_system_info" => {
            let r = load_separated_system_info_panels(root);
            finish(r.as_ref().map_or(json!({}), |p| json!({"panels": frames_info(p)})), &r)
        }
        "load_lmi1_bank" => {
            let r = load_separated_lmi1_bank(root, i(a, "resource"));
            finish(r.as_ref().map_or(json!({}), |e| json!({"entries": e.iter().map(lmi_info).collect::<Vec<_>>()})), &r)
        }
        "load_load_slots" => {
            let r = load_separated_load_slots_frame(root);
            finish(r.as_ref().map_or(json!({}), |e| json!({"entry": lmi_info(e)})), &r)
        }
        "load_action_cells" => {
            let r = load_separated_action_cells(root);
            finish(r.as_ref().map_or(json!({}), |c| json!({"cells": c.iter().map(raw_info).collect::<Vec<_>>()})), &r)
        }
        "load_church" => {
            let r = load_separated_church_ui_assets(root);
            let v = r.as_ref().map_or(json!({}), |c| {
                json!({"background": frame_info(&c.background), "entries": c.entries.iter().map(lmi_info).collect::<Vec<_>>(),
                    "price": raw_info(&c.price_cell), "revive": frames_info(&c.revive_fx)})
            });
            finish(v, &r)
        }
        "load_chapter_aux" | "load_chapter_aux_resource" => {
            let r = if op == "load_chapter_aux" {
                load_separated_chapter_aux_surface(root)
            } else {
                let c = &a["contract"];
                let contract = NativeChapterAuxSurfaceContract {
                    resource: c[0].as_i64().unwrap(),
                    raw_md5: c[1].as_str().unwrap().to_string().into(),
                    raw_sha256: c[2].as_str().unwrap().to_string().into(),
                };
                load_separated_chapter_aux_surface_resource(root, &contract)
            };
            finish(r.as_ref().map_or(json!({}), |s| json!({"pixels": digest(&s.pixels)})), &r)
        }
        "load_sound" => {
            let r = load_separated_sound_bank(root, i(a, "resource"));
            let v = r.as_ref().map_or(
                json!({}),
                |b| json!({"resource": b.resource, "encoded": string_keyed(&b.encoded, |e| digest(e).into())}),
            );
            finish(v, &r)
        }
        "load_lut_bank" => {
            let r = load_separated_lut_bank(root);
            finish(
                r.as_ref().map_or(json!({}), |l| json!({"luts": l.iter().map(|x| digest(x)).collect::<Vec<_>>()})),
                &r,
            )
        }
        "load_range_overlay" => {
            let r = load_separated_range_overlay_bank(root);
            finish(r.as_ref().map_or(json!({}), |b| json!({"bank": bank_info(b)})), &r)
        }
        "load_frame_bank" => {
            let r = match s(a, "which") {
                "event61" => load_separated_event61_frames(root),
                "ending_prefix" => load_separated_ending_prefix_frames(root),
                "ch20_sky_key" => load_separated_ch20_sky_key_frames(root),
                _ => load_separated_pending_code1_frames(root),
            };
            finish(r.as_ref().map_or(json!({}), |f| json!({"frames": frames_info(f)})), &r)
        }
        "load_parallax" => {
            let r = load_separated_native_map_parallax(root);
            finish(r.as_ref().map_or(json!({}), |s| json!({"pixels": digest(&s.pixels)})), &r)
        }
        _ => panic!("未知載入器 {op}"),
    }
}

// ---- 以路徑為參數的 API ----

fn path_op(op: &str, path: &Path, a: i64, b: i64) -> Out {
    let usize_of = |v: i64| usize::try_from(v).unwrap_or(usize::MAX);
    match op {
        "decode_resource" => {
            let r = decode_resource(path, a);
            finish_plain(r.as_ref().map_or(json!({}), |f| json!({"frames": frames_info(f)})), &r)
        }
        "decode_archive_single_frame" => {
            let r = decode_archive_single_frame(path, a);
            finish_plain(r.as_ref().map_or(json!({}), |f| json!({"frame": frame_info(f)})), &r)
        }
        "decode_lmi1_resource" => {
            let r = decode_lmi1_resource(path, a);
            finish_plain(
                r.as_ref().map_or(json!({}), |e| json!({"entries": e.iter().map(lmi_info).collect::<Vec<_>>()})),
                &r,
            )
        }
        "decode_lmi1_frame_resource" => {
            let r = decode_lmi1_frame_resource(path, a, b);
            finish_plain(r.as_ref().map_or(json!({}), |f| json!({"frame": frame_info(f)})), &r)
        }
        "decode_lmi1_raw_entry" => {
            let r = decode_lmi1_raw_entry(path, a, b);
            finish_plain(r.as_ref().map_or(json!({}), |c| json!({"cell": raw_info(c)})), &r)
        }
        "decode_raw_cell_resource" => {
            let r = decode_raw_cell_resource(path, a);
            finish_plain(
                r.as_ref().map_or(json!({}), |c| json!({"cells": c.iter().map(raw_info).collect::<Vec<_>>()})),
                &r,
            )
        }
        "decode_lut_resource" => {
            let r = decode_lut_resource(path, a);
            finish_plain(
                r.as_ref().map_or(json!({}), |l| json!({"luts": l.iter().map(|x| digest(x)).collect::<Vec<_>>()})),
                &r,
            )
        }
        "decode_sprite_bank_resource" => {
            let r = decode_sprite_bank_resource(path, a);
            finish_plain(r.as_ref().map_or(json!({}), |b| json!({"bank": bank_info(b)})), &r)
        }
        "decode_map_terrain_resources" => {
            let r = decode_map_terrain_resources(path, a);
            finish_plain(
                r.as_ref().map_or(json!({}), |(b, c)| json!({"bank": bank_info(b), "controls": digest(c)})),
                &r,
            )
        }
        "decode_native_range_overlay_bank" => {
            let r = decode_native_range_overlay_bank(path);
            finish_plain(r.as_ref().map_or(json!({}), |b| json!({"bank": bank_info(b)})), &r)
        }
        "decode_native_chapter_aux_surface" => {
            let r = decode_native_chapter_aux_surface(path);
            finish_plain(r.as_ref().map_or(json!({}), |s| json!({"pixels": digest(&s.pixels)})), &r)
        }
        "read_resource" => {
            let r = read_resource(path, usize_of(a)).map_err(FdotherError::from);
            finish_plain(r.as_ref().map_or(json!({}), |x| json!({"raw": digest(x)})), &r)
        }
        "read_nested_resource" => {
            let r = read_nested_resource(path, usize_of(a), usize_of(b)).map_err(FdotherError::from);
            finish_plain(r.as_ref().map_or(json!({}), |x| json!({"raw": digest(x)})), &r)
        }
        _ => panic!("未知路徑操作 {op}"),
    }
}

// ---- 結構化案例 ----

fn run(env: &mut Env, op: &str, input: &Value) -> Option<Out> {
    let n = |k: &str| i(input, k);
    Some(match op {
        "parse" => {
            let data = if let Some(name) = input.get("real").and_then(Value::as_str) {
                let archive = env.archive(name)?.clone();
                let entry = archive_entry(&archive, n("resource") as usize).unwrap().to_vec();
                match input.get("nested") {
                    Some(nested) => archive_entry(&entry, nested.as_u64().unwrap() as usize).unwrap().to_vec(),
                    None => entry,
                }
            } else {
                bytes_of(&input["data"])
            };
            let index = input.get("index").and_then(Value::as_i64).unwrap_or(0);
            protect(|| parse_op(s(input, "parser"), &data, index))
        }
        "frame_blit" => {
            let f = frame_from(&input["frame"]);
            let mut dst = pattern(n("dst_len"), n("seed"));
            let (stride, base) = (n("stride"), n("base"));
            let lut = input
                .get("lut")
                .map(|l| lut_of(l[0].as_i64().unwrap(), l[1].as_i64().unwrap(), l[2].as_i64().unwrap()));
            protect(|| {
                let r = match &lut {
                    Some(lut) => f.blit_lut_at(&mut dst, stride, base, lut),
                    None => f.blit_at(&mut dst, stride, base, n("transparent")),
                };
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "frame_info" => out(frame_info(&frame_from(&input["frame"]))),
        "text_placeholder" => {
            let f = frame_from(&input["frame"]);
            let calls = Arc::new(Mutex::new(Vec::<[i64; 2]>::new()));
            let sink = Arc::clone(&calls);
            let mut o = protect(|| {
                let r = f.text_placeholder(move |x, y| sink.lock().unwrap().push([x, y]));
                let v = match &r {
                    Ok(p) => {
                        let mut dst = pattern(n("dst_len"), n("seed"));
                        let b = p.blit_at(&mut dst, n("stride"), n("base"), -1);
                        json!({"info": frame_info(p), "blit_err": b.err().map(|e| e.to_string()), "dst": digest(&dst)})
                    }
                    Err(_) => json!({}),
                };
                finish(v, &r)
            });
            o.v.insert("calls".into(), json!(*calls.lock().unwrap()));
            o
        }
        "lmi1_blit" => {
            let e = entry_from(&input["entry"]);
            let mut dst = pattern(n("dst_len"), n("seed"));
            let (stride, x, y, mirror) = (n("stride"), n("x"), n("y"), input["mirror"].as_bool().unwrap());
            protect(|| {
                let r = match s(input, "variant") {
                    "at" => e.blit_at(&mut dst, stride, x, y, mirror),
                    "opaque" => e.blit_opaque_at(&mut dst, stride, x, y, mirror),
                    _ => e.blit_at_clipped(&mut dst, stride, x, y, mirror),
                };
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "raw_blit" => {
            let c = cell_from(&input["cell"]);
            let mut dst = pattern(n("dst_len"), n("seed"));
            let (stride, x, y, offset) = (n("stride"), n("x"), n("y"), n("offset"));
            protect(|| {
                let r = match s(input, "variant") {
                    "at" => c.blit_at(&mut dst, stride, x, y),
                    "clipped" => c.blit_at_clipped(&mut dst, stride, x, y),
                    _ => c.blit_opaque_at_offset(&mut dst, stride, offset),
                };
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "raw_paletted" => {
            let c = cell_from(&input["cell"]);
            let palette: Vec<Nrgba> = input["palette"]
                .as_array()
                .unwrap()
                .iter()
                .map(|p| {
                    let ch = |k: usize| p[k].as_u64().unwrap() as u8;
                    Nrgba { r: ch(0), g: ch(1), b: ch(2), a: ch(3) }
                })
                .collect();
            protect(|| {
                let r = c.paletted(&palette);
                let v = r.as_ref().map_or(json!({}), |im| {
                    json!({"pix": hex(&im.pix), "stride": im.stride, "rect": [im.width, im.height],
                        "palette": im.palette.iter().map(|c| json!([c.r, c.g, c.b, c.a])).collect::<Vec<_>>()})
                });
                finish(v, &r)
            })
        }
        "palette_apply" => {
            let mut dac = bytes_of(&input["dac"]);
            let base = bytes_of(&input["baseline"]);
            let (start, end, delta) = (n("start"), n("end"), n("delta"));
            protect(|| {
                let r = if input["sub"].as_bool().unwrap() {
                    apply_vga_palette_subtraction(&mut dac, &base, start, end, delta)
                } else {
                    apply_vga_palette_delta(&mut dac, &base, start, end, delta)
                };
                finish(json!({"dac": hex(&dac)}), &r)
            })
        }
        "bridge_lut" => {
            let luts: Vec<Vec<u8>> = input["lens"]
                .as_array()
                .unwrap()
                .iter()
                .enumerate()
                .map(|(j, len)| lut_of(len.as_i64().unwrap(), 7 + j as i64, j as i64))
                .collect();
            protect(|| {
                let r = native_unit_present_bridge_lut(&luts);
                finish(json!({"bridge": r.as_ref().map_or(String::new(), |b| hex(b))}), &r)
            })
        }
        "remap" => {
            let spec = &input["spec"];
            let g = |k: &str| i(spec, k);
            let lut = &input["lut"];
            let lut = lut_of(lut[0].as_i64().unwrap(), lut[1].as_i64().unwrap(), lut[2].as_i64().unwrap());
            let mut dst = pattern(n("dst_len"), n("seed"));
            let stride = n("stride");
            protect(|| {
                let r = if input["radial"].as_bool().unwrap() {
                    let spec = RadialLUTRemap {
                        center_x: g("center_x"),
                        center_y: g("center_y"),
                        radius: g("radius"),
                        scale: g("scale"),
                        start_y: g("start_y"),
                        end_y: g("end_y"),
                        clip_width: g("clip_width"),
                    };
                    apply_radial_lut_remap(&mut dst, stride, &lut, spec)
                } else {
                    let spec = CenteredRectLUTRemap {
                        center_x: g("center_x"),
                        horizontal_radius: g("horizontal_radius"),
                        start_y: g("start_y"),
                        end_y: g("end_y"),
                        clip_width: g("clip_width"),
                    };
                    apply_centered_rect_lut_remap(&mut dst, stride, &lut, spec)
                };
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "range_placements" => protect(|| {
            let r = native_range_overlay_placements(n("mode"), n("cx"), n("cy"));
            let list: Vec<Value> =
                r.as_ref().map_or(Vec::new(), |p| p.iter().map(|p| json!([p.x, p.y, p.descriptor])).collect());
            finish(json!({"placements": list}), &r)
        }),
        "mode6_address" => protect(|| {
            let r = native_range_overlay_mode6_byte_address(n("w"), n("cx"), n("cy"));
            finish(json!({"value": r.as_ref().copied().unwrap_or(0)}), &r)
        }),
        "mode6_clear" => {
            let mut field = bytes_of(&input["field"]);
            protect(|| {
                let r = clear_native_range_overlay_mode6_field_byte(&mut field, n("cx"), n("cy"));
                finish(json!({"field": hex(&field)}), &r)
            })
        }
        "range_blit" => {
            let bank = env.banks[s(input, "bank")].clone();
            let mut dst = pattern(n("dst_len"), n("seed"));
            protect(|| {
                let r = blit_native_range_overlay(
                    &bank,
                    &mut dst,
                    n("camera_x"),
                    n("camera_y"),
                    n("visible_width"),
                    n("visible_height"),
                    n("mode"),
                    n("cx"),
                    n("cy"),
                );
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "parallax_build" => {
            let (top, bottom) = (frame_from(&input["top"]), frame_from(&input["bottom"]));
            protect(|| {
                let r = build_native_map_parallax(&top, &bottom);
                finish(r.as_ref().map_or(json!({}), |s| json!({"pixels": digest(&s.pixels)})), &r)
            })
        }
        "parallax_viewport" => {
            let surface = NativeMapParallaxSurface { pixels: pattern(n("surface_len"), 5) };
            let mut dst = pattern(n("dst_len"), n("seed"));
            protect(|| {
                let r = blit_native_map_parallax_viewport(
                    &mut dst,
                    n("stride"),
                    &surface,
                    n("cx"),
                    n("cy"),
                    n("sx"),
                    n("sy"),
                );
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "aux_for" => {
            let c = native_chapter_aux_surface_for(n("chapter"));
            let ok = c.is_some();
            let c = c.unwrap_or_default();
            out(json!({"ok": ok, "contract": [c.resource, c.raw_md5, c.raw_sha256]}))
        }
        "aux_contracts" => {
            let list: Vec<Value> = native_chapter_aux_surface_contracts()
                .iter()
                .map(|c| json!([c.resource, c.raw_md5, c.raw_sha256]))
                .collect();
            out(json!({"contracts": list}))
        }
        "aux_viewport" => {
            let surface = NativeChapterAuxSurface { pixels: pattern(n("surface_len"), 9) };
            let mut dst = pattern(n("dst_len"), n("seed"));
            protect(|| {
                let r = blit_native_chapter_aux_viewport(&mut dst, n("stride"), &surface, n("phase"));
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "split_steps" => {
            let steps: Vec<Value> = native_battle_entry_split_slide_steps()
                .iter()
                .map(|s| json!([s.offset, s.left_x, s.left_y, s.right_x, s.right_y]))
                .collect();
            out(json!({"steps": steps}))
        }
        "split_cell" => {
            let cell = entry_from(&input["cell"]);
            let st = &input["step"];
            let step = NativeBattleEntrySplitSlideStep {
                offset: st[0].as_i64().unwrap(),
                left_x: st[1].as_i64().unwrap(),
                left_y: st[2].as_i64().unwrap(),
                right_x: st[3].as_i64().unwrap(),
                right_y: st[4].as_i64().unwrap(),
            };
            let mut dst = pattern(n("dst_len"), n("seed"));
            protect(|| {
                let r = blit_native_battle_entry_split_slide_cell(&cell, step, &mut dst, n("stride"));
                finish(json!({"dst": digest(&dst)}), &r)
            })
        }
        "split_run" => {
            let cell = entry_from(&input["cell"]);
            let mut dst = pattern(n("dst_len"), n("seed"));
            let events = std::cell::RefCell::new(Vec::<&str>::new());
            let (presents, restores) = (std::cell::Cell::new(0), std::cell::Cell::new(0));
            let (fail_present, fail_restore) = (n("fail_present"), n("fail_restore"));
            let mut o = protect(|| {
                let r = run_native_battle_entry_split_slide::<FdotherError>(
                    &cell,
                    &mut dst,
                    || {
                        events.borrow_mut().push("present");
                        presents.set(presents.get() + 1);
                        if presents.get() == fail_present {
                            return Err(FdotherError::Contract("present failed".into()));
                        }
                        Ok(())
                    },
                    || {
                        events.borrow_mut().push("restore");
                        restores.set(restores.get() + 1);
                        if restores.get() == fail_restore {
                            return Err(FdotherError::Contract("restore failed".into()));
                        }
                        Ok(())
                    },
                );
                finish(json!({"dst": digest(&dst)}), &r)
            });
            o.v.insert("events".into(), json!(events.into_inner()));
            o
        }
        "dialogue_grid" => protect(|| {
            let r = plan_native_dialogue_frame_grid(n("stride"), n("x"), n("y"), n("columns"), n("rows"));
            let list: Vec<Value> = r
                .as_ref()
                .map_or(Vec::new(), |p| p.iter().map(|p| json!([p.resource_index, p.destination_byte])).collect());
            finish(json!({"placements": list}), &r)
        }),
        "terrain_lut" => protect(|| {
            let r = native_terrain_lut_index(n("phase"));
            finish(json!({"value": r.as_ref().copied().unwrap_or(0)}), &r)
        }),
        "terrain_phase" => protect(|| {
            let state = NativeTerrainPhaseState { phase: n("phase"), last_timer_tick: n("last") };
            let r = advance_native_terrain_phase(state, n("tick"), n("override"));
            // Go 錯誤時回傳未改動的輸入狀態。
            let got = r.as_ref().copied().unwrap_or(state);
            finish(json!({"state": [got.phase, got.last_timer_tick]}), &r)
        }),
        _ if op.starts_with("decode_") || op.starts_with("read_") => {
            let dir = env.flame2.clone()?;
            protect_plain(|| path_op(op, &dir.join(s(input, "file")), n("a"), n("b")))
        }
        _ if op.starts_with("load_") => {
            let root = &input["root"];
            if let Some(name) = root.get("tree").and_then(Value::as_str) {
                env.assets.as_ref()?;
                let dir = env.tmp.join(name);
                env.materialize(name, &dir);
                let mut o = protect(|| run_loader(op, &dir, input));
                if let Some(Value::String(e)) = o.v.get_mut("err") {
                    *e = e.replace(dir.to_str().unwrap(), "$ROOT");
                }
                std::fs::remove_dir_all(&dir).unwrap();
                o
            } else if let Some(rel) = root.get("real").and_then(Value::as_str) {
                let dir = env.assets.as_ref()?.join(rel);
                protect(|| run_loader(op, &dir, input))
            } else {
                protect(|| run_loader(op, Path::new(""), input))
            }
        }
        _ => panic!("未知操作 {op}"),
    })
}

/// 路徑 API 在 Go 沒包 protect；panic 時直接讓測試失敗。
fn protect_plain(f: impl FnOnce() -> Out) -> Out {
    f()
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

/// 連接受與否都和 Go 不同的案例種類：(kind, 刻意不照做的原因)。
/// 以 kind 的結尾比對。
const KNOWN_DIFFERENCES: &[(&str, &str)] = &[
    ("-json-fold", "Go 的欄位名稱不分大小寫（KIND 對到 kind），serde 只比對原字，缺欄而拒收"),
    ("-json-utf8", "字串裡的無效 UTF-8，Go 換成 U+FFFD，serde_json 拒收"),
];

/// 覆蓋統計用：把錯誤訊息中的數字換成 N。
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
            out.chars().take(110).collect()
        }
        Some(Value::Null) => "ok".to_string(),
        _ => "value".to_string(),
    }
}

fn tree_digest(root: &Path) -> (String, usize) {
    fn walk(root: &Path, dir: &Path, lines: &mut Vec<String>) {
        for entry in std::fs::read_dir(dir).unwrap() {
            let entry = entry.unwrap();
            let ft = entry.file_type().unwrap();
            if ft.is_dir() {
                walk(root, &entry.path(), lines);
            } else if ft.is_file() {
                let rel = entry.path().strip_prefix(root).unwrap().to_str().unwrap().to_string();
                lines.push(format!("{rel}\0{}\n", digest(&std::fs::read(entry.path()).unwrap())));
            }
        }
    }
    let mut lines = Vec::new();
    walk(root, root, &mut lines);
    lines.sort();
    (digest(lines.concat().as_bytes()), lines.len())
}

#[test]
fn matches_frozen_go_golden() {
    let required = std::env::var("FD2_REQUIRE_GOLDEN").is_ok_and(|v| v == "1");
    let Ok(path) = std::env::var("FD2_GOLDEN_FDOTHER_CORE") else {
        assert!(!required, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_FDOTHER_CORE");
        eprintln!("略過：未設定 FD2_GOLDEN_FDOTHER_CORE");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-fdother-core/1");
    let flame2 = std::env::var_os("FD2_FLAME2_DIR").map(PathBuf::from).filter(|p| p.is_dir());
    let assets = std::env::var_os("FD2_ASSET_PACK").map(PathBuf::from).filter(|p| p.is_dir());
    if required {
        assert!(flame2.is_some(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_FLAME2_DIR");
        assert!(assets.is_some(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_ASSET_PACK");
    }
    if let Some(dir) = &flame2 {
        for name in ["FDOTHER.DAT", "BG.DAT", "TAI.DAT", "FDSHAP.DAT"] {
            let rec = &doc["real"][name];
            let data = std::fs::read(dir.join(name)).unwrap();
            assert_eq!(data.len() as u64, rec["size"].as_u64().unwrap(), "{name} 大小不同");
            assert_eq!(digest(&data), rec["sha256"].as_str().unwrap(), "{name} 與黃金檔記錄的雜湊不同");
        }
    }
    if let Some(dir) = &assets {
        for (rel, rec) in doc["real"]["pack"].as_object().unwrap() {
            let (sum, files) = tree_digest(&dir.join(rel));
            assert_eq!(files as u64, rec["files"].as_u64().unwrap(), "{rel} 檔案數不同");
            assert_eq!(sum, rec["tree_sha256"].as_str().unwrap(), "{rel} 檔案清單摘要不同");
        }
    }
    let tmp = std::env::temp_dir().join(format!("fd2-golden-fdother-core-{}", std::process::id()));
    std::fs::create_dir_all(&tmp).unwrap();
    let mut env = Env {
        flame2,
        assets,
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
            .map(|(k, rows)| {
                let sprites = rows
                    .as_array()
                    .unwrap()
                    .iter()
                    .map(|r| Sprite {
                        pixels: unhex(r[0].as_str().unwrap()),
                        mask: unhex(r[1].as_str().unwrap()),
                        remap_mask: unhex(r[2].as_str().unwrap()),
                    })
                    .collect();
                (k.clone(), Bank { sprites })
            })
            .collect(),
        archives: BTreeMap::new(),
    };

    let cases = doc["cases"].as_array().unwrap();
    assert!(!cases.is_empty());
    let mut failures = Vec::new();
    let mut skipped = 0;
    let mut compares = BTreeMap::<String, usize>::new();
    let mut coverage = BTreeMap::<String, BTreeMap<String, usize>>::new();
    let mut loose = 0;
    let mut known = Vec::new();
    for case in cases {
        let index = case["index"].as_u64().unwrap();
        let (kind, op, compare) = (s(case, "kind"), s(case, "op"), s(case, "compare"));
        let Some(got) = run(&mut env, op, &case["input"]) else {
            skipped += 1;
            continue;
        };
        *compares.entry(compare.to_string()).or_default() += 1;
        *coverage.entry(op.to_string()).or_default().entry(classify(&case["result"])).or_default() += 1;
        let want = &case["result"];
        let got_value = Value::Object(got.v.clone());
        let (go_ok, rust_ok) = (want["err"].is_null(), got_value["err"].is_null());
        let mismatch = match compare {
            "exact" => first_diff("result", want, &got_value),
            "err_prefix" | "accept_reject" if go_ok != rust_ok => {
                Some(format!("result.err: 接受與否不同 go={} rust={}", want["err"], got_value["err"]))
            }
            "err_prefix" if !go_ok => {
                let want_prefix = case["prefix"].as_str().unwrap();
                let got_prefix = got.prefix.clone().unwrap_or_default();
                if want_prefix != got_prefix {
                    Some(format!("prefix: go={want_prefix:?} rust={got_prefix:?}（rust 錯誤 {}）", got_value["err"]))
                } else {
                    let (mut a, mut b) = (want.clone(), got_value.clone());
                    a["err"] = Value::Null;
                    b["err"] = Value::Null;
                    first_diff("result", &a, &b)
                }
            }
            "accept_reject" if !go_ok => {
                loose += 1;
                None
            }
            _ => first_diff("result", want, &got_value),
        };
        if let Some(d) = mismatch {
            if let Some((_, reason)) = KNOWN_DIFFERENCES.iter().find(|(k, _)| kind.ends_with(k)) {
                known.push(format!("case {index} ({kind} {op})：刻意不照做，{reason}；{d}"));
            } else {
                failures.push(format!("case {index} ({kind} {op} {compare}): {d}"));
            }
        }
    }
    let _ = std::fs::remove_dir_all(&tmp);
    eprintln!("黃金案例 {}（略過 {skipped}）：{compares:?}", cases.len());
    eprintln!("只比接受／拒收且雙方都拒收：{loose}");
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
    let shown: Vec<&String> = failures.iter().take(60).collect();
    assert!(
        failures.is_empty(),
        "{} 個案例不符：\n{}",
        failures.len(),
        shown.iter().map(|s| s.as_str()).collect::<Vec<_>>().join("\n")
    );
}
