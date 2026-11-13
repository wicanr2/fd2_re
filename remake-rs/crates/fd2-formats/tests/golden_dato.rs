//! 與凍結 Go 版 `remake/cmd/fd2-golden-dato` 的黃金差分。
//!
//! 由 `tools/rust_golden_dato.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_DATO`；真實原版檔從
//! `FD2_FLAME2_DIR`（DATO.DAT）與 `FD2_PORTRAITS_DIR`（分離頭像）讀，SHA-256 必須與黃金檔
//! 記錄相同。設定 `FD2_REQUIRE_GOLDEN=1` 時缺檔即失敗，避免把「沒跑到」當成通過。
//!
//! 錯誤文字逐字比對；只有包住作業系統 I/O 錯誤的情形（Go 以 `%w` 包裝 `*PathError`，內層文字
//! 依平台而異）改比去掉內層後的前綴，並另外計數。分離頭像的畸形 PNG 案例只比接受或拒收，
//! 已知不同的類別列在 `KNOWN_DIFFERENCES`。

use fd2_formats::archive::read_resource;
use fd2_formats::dato::*;
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::panic::{AssertUnwindSafe, catch_unwind};
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

/// 錯誤的比對方式：`exact` 逐字比，`prefix` 只比 Go `%w` 外層前綴。
struct ErrView {
    text: String,
    prefix: Option<String>,
    foreign: bool,
    class: String,
}

fn err_view(e: &DatoError) -> ErrView {
    let text = e.to_string();
    let (prefix, foreign, class) = match e {
        DatoError::FramePixels { frame, source } => {
            (Some(format!("dato: frame {frame}: ")), false, format!("frame-pixels:{source}"))
        }
        DatoError::SeparatedOpen { resource, mouth, .. } => {
            (Some(format!("dato: separated portrait {resource} frame {mouth}: ")), true, "separated-open".into())
        }
        DatoError::SeparatedDecode { resource, mouth, source } => {
            let prefix = format!("dato: separated portrait {resource} frame {mouth} decode: ");
            let (foreign, class) = match source {
                ImageError::UnknownFormat => (false, "decode-unknown-format".to_string()),
                ImageError::Png(_) => (false, "decode-png".to_string()),
                ImageError::Io(_) => (true, "decode-io".to_string()),
            };
            (Some(prefix), foreign, class)
        }
        DatoError::Archive(a) => {
            (None, matches!(a, fd2_formats::archive::ArchiveError::Io { .. }), format!("archive:{a}"))
        }
        other => (None, false, variant_class(other)),
    };
    ErrView { text, prefix, foreign, class }
}

fn variant_class(e: &DatoError) -> String {
    let s = format!("{e:?}");
    s.split(['(', ' ', '{']).next().unwrap_or("").to_string()
}

#[derive(Default)]
struct Stats {
    exact_err: usize,
    prefix_err: usize,
    coverage: BTreeMap<String, BTreeMap<String, usize>>,
}

impl Stats {
    fn count(&mut self, section: &str, class: &str) {
        *self.coverage.entry(section.into()).or_default().entry(class.into()).or_default() += 1;
    }
}

fn frames_json(frames: &[Frame]) -> Value {
    Value::Array(
        frames
            .iter()
            .map(|f| {
                let mut row = Map::new();
                row.insert("w".into(), json!(f.width));
                row.insert("h".into(), json!(f.height));
                row.insert("len".into(), json!(f.pixels.len()));
                row.insert("sha256".into(), json!(digest(&f.pixels)));
                if f.pixels.len() <= 256 {
                    row.insert("hex".into(), json!(hex(&f.pixels)));
                }
                Value::Object(row)
            })
            .collect(),
    )
}

/// 把 Rust 結果轉成 Go `framesResult` 的形狀；`Err` 對應 Go 的 nil 切片。錯誤欄位依
/// [`ErrView`] 與 Go 端調和後寫入，讓 `first_diff` 只在真正不一致時報錯。
fn frames_result(
    section: &str,
    want: &Value,
    got: &std::result::Result<Vec<Frame>, DatoError>,
    stats: &mut Stats,
) -> Value {
    let mut out = Map::new();
    match got {
        Ok(frames) => {
            out.insert("err".into(), Value::Null);
            out.insert("nil".into(), json!(false));
            out.insert("frames".into(), frames_json(frames));
            stats.count(section, "ok");
        }
        Err(e) => {
            out.insert("nil".into(), json!(true));
            insert_err(&mut out, want, e, stats);
            stats.count(section, &err_view(e).class);
        }
    }
    Value::Object(out)
}

fn insert_err(out: &mut Map<String, Value>, want: &Value, e: &DatoError, stats: &mut Stats) {
    let view = err_view(e);
    if let Some(p) = &view.prefix {
        out.insert("err_prefix".into(), json!(p));
    }
    let go_err = want["err"].as_str();
    if go_err == Some(view.text.as_str()) {
        stats.exact_err += 1;
        out.insert("err".into(), json!(view.text));
        return;
    }
    // 外部錯誤只比前綴：Go 的 err_prefix 與 Rust 的前綴相同時，採用 Go 的全文。
    // Archive 的 I/O 錯誤沒有 Rust 前綴，只要求兩邊都是 `open ` 開頭的錯誤。
    let prefix_ok = match (&view.prefix, want["err_prefix"].as_str()) {
        (Some(p), Some(gp)) => p == gp,
        (None, _) => go_err.is_some_and(|g| g.starts_with("open ")) && view.text.starts_with("open "),
        _ => false,
    };
    if view.foreign && prefix_ok {
        stats.prefix_err += 1;
        out.insert("err".into(), want["err"].clone());
    } else {
        out.insert("err".into(), json!(view.text));
    }
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

fn i(v: &Value) -> i64 {
    v.as_i64().unwrap_or_else(|| panic!("不是整數：{v}"))
}

fn check_parse(cases: &[Value], stats: &mut Stats, failures: &mut Vec<String>) {
    for case in cases {
        let got = parse_resource(&unhex(case["data_hex"].as_str().unwrap()));
        let result = frames_result("parse", &case["result"], &got, stats);
        if let Some(d) = first_diff("result", &case["result"], &result) {
            failures.push(format!("parse case {} ({}): {d}", case["index"], case["kind"]));
        }
    }
}

fn check_blit(cases: &[Value], stats: &mut Stats, failures: &mut Vec<String>) {
    let hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(|_| {}));
    for case in cases {
        let input = &case["input"];
        let frame = Frame {
            width: i(&input["w"]),
            height: i(&input["h"]),
            pixels: unhex(input["pixels_hex"].as_str().unwrap()),
        };
        let mut dst = unhex(input["dst_hex"].as_str().unwrap());
        let (stride, x, y, offset) = (i(&input["stride"]), i(&input["x"]), i(&input["y"]), i(&input["offset"]));
        let op = input["op"].as_str().unwrap().to_string();
        let run = catch_unwind(AssertUnwindSafe(|| match op.as_str() {
            "at" => frame.blit_at(&mut dst, stride, x, y),
            "offset" => frame.blit_at_offset(&mut dst, stride, offset),
            _ => frame.blit_right_to_left_at_offset(&mut dst, stride, offset),
        }));
        let mut out = Map::new();
        match run {
            Err(_) => {
                out.insert("err".into(), Value::Null);
                out.insert("panic".into(), json!(true));
                stats.count("blit", &format!("{op}:panic"));
            }
            Ok(Ok(())) => {
                out.insert("err".into(), Value::Null);
                out.insert("panic".into(), json!(false));
                stats.count("blit", &format!("{op}:ok"));
            }
            Ok(Err(e)) => {
                insert_err(&mut out, &case["result"], &e, stats);
                out.insert("panic".into(), json!(false));
                stats.count("blit", &format!("{op}:{}", err_view(&e).class));
            }
        }
        out.insert("dst_hex".into(), json!(hex(&dst)));
        if let Some(d) = first_diff("result", &case["result"], &Value::Object(out)) {
            failures.push(format!("blit case {} ({}): {d}", case["index"], case["kind"]));
        }
    }
    std::panic::set_hook(hook);
}

fn state_json(
    err: Option<&DatoError>,
    want: &Value,
    open: bool,
    countdown: i64,
    frame: usize,
    stats: &mut Stats,
) -> Value {
    let mut out = Map::new();
    match err {
        Some(e) => insert_err(&mut out, want, e, stats),
        None => {
            out.insert("err".into(), Value::Null);
        }
    }
    out.insert("open".into(), json!(open));
    out.insert("countdown".into(), json!(countdown));
    out.insert("frame".into(), json!(frame));
    Value::Object(out)
}

/// 嘴型狀態機。Go 的 `Tick` 出錯時回傳零值結構，序列從零值繼續；Rust 端照做。
fn check_mouth(cases: &[Value], stats: &mut Stats, failures: &mut Vec<String>) {
    for case in cases {
        let samples: Vec<i64> = case["samples"].as_array().unwrap().iter().map(i).collect();
        let want_steps = case["steps"].as_array().cloned().unwrap_or_default();
        let mut got = Map::new();
        let mut steps = Vec::new();
        let machine = case["machine"].as_str().unwrap();
        if machine == "mouth" {
            let mut s = MouthState {
                open: case["start"]["open"].as_bool().unwrap(),
                countdown: i(&case["start"]["countdown"]),
            };
            for (k, &sample) in samples.iter().enumerate() {
                let r = s.tick(sample);
                let class = r.as_ref().err().map_or("ok".to_string(), |e| err_view(e).class);
                stats.count("mouth", &class);
                s = r.as_ref().copied().unwrap_or_default();
                steps.push(state_json(r.as_ref().err(), &want_steps[k], s.open, s.countdown, s.frame_index(), stats));
            }
        } else {
            let mut s = if case.get("new").is_some() {
                let r = ConfirmationMouthState::new(i(&case["new_sample"]));
                let class = r.as_ref().err().map_or("new-ok".to_string(), |e| err_view(e).class);
                stats.count("confirmation", &class);
                let s = r.as_ref().copied().unwrap_or_default();
                got.insert(
                    "new".into(),
                    state_json(r.as_ref().err(), &case["new"], s.open, s.countdown, s.frame_index(), stats),
                );
                s
            } else {
                ConfirmationMouthState {
                    open: case["start"]["open"].as_bool().unwrap(),
                    countdown: i(&case["start"]["countdown"]),
                }
            };
            for (k, &sample) in samples.iter().enumerate() {
                let r = s.tick(sample);
                let class = r.as_ref().err().map_or("ok".to_string(), |e| err_view(e).class);
                stats.count("confirmation", &class);
                s = r.as_ref().copied().unwrap_or_default();
                steps.push(state_json(r.as_ref().err(), &want_steps[k], s.open, s.countdown, s.frame_index(), stats));
            }
        }
        got.insert("steps".into(), Value::Array(steps));
        let mut want = Map::new();
        want.insert("steps".into(), case["steps"].clone());
        if let Some(n) = case.get("new") {
            want.insert("new".into(), n.clone());
        }
        if let Some(d) = first_diff("case", &Value::Object(want), &Value::Object(got)) {
            failures.push(format!("mouth case {} ({} {}): {d}", case["index"], machine, case["kind"]));
        }
    }
}

/// 合法輸入的案例類別：逐欄比對（含像素與錯誤文字）。其餘是刻意畸形的 PNG，只比接受或拒收；
/// 兩邊都接受時仍要求解出的幀相同。命名規則見 `remake/cmd/fd2-golden-dato`。
fn exact_kind(kind: &str) -> bool {
    ["input-", "ok-", "open-", "type-", "nonpng-"].iter().any(|p| kind.starts_with(p))
        || matches!(kind, "random-clean" | "stream-clean")
}

/// 不一致的方向：`GoOnly` 是 Go 接受而 Rust 拒收，`RustOnly` 是 Go 拒收而 Rust 接受。
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
enum Side {
    GoOnly,
    RustOnly,
}

/// 刻意不照做：Go 標準庫 `image/png` 與 `png` crate 對下列畸形 PNG 的接受與否不同。理由見
/// `fd2_formats::dato::load_separated_resource` 的說明。名稱以 `*` 結尾者比對前綴；方向也要
/// 相符，清單上的每一項都必須真的出現不一致，避免成為無條件略過。
const KNOWN_DIFFERENCES: &[(&str, Side, &str)] = &[
    ("lenient-unknown-before-ihdr", Side::GoOnly, "刻意不照做：Go 忽略 IHDR 之前的未知區塊，png crate 拒收"),
    (
        "lenient-text-after-complete",
        Side::GoOnly,
        "刻意不照做：Go 解完影像後忽略隔開的 IDAT，png crate 拒收 IDAT 重新開始",
    ),
    ("crc-unknown", Side::RustOnly, "刻意不照做：輔助區塊（tEXt）CRC 錯誤，Go 拒收，png crate 略過"),
    ("missing-plte", Side::RustOnly, "刻意不照做：indexed 影像缺 PLTE，Go 拒收，png crate 不要求調色盤"),
    ("trns-before-plte", Side::RustOnly, "刻意不照做：tRNS 在 PLTE 之前，Go 視為區塊順序錯誤"),
    ("trns-too-long", Side::RustOnly, "刻意不照做：tRNS 超過 256 項，Go 拒收"),
    ("plte-length-4", Side::RustOnly, "刻意不照做：PLTE 長度不是 3 的倍數，Go 拒收"),
    ("plte-over-depth", Side::RustOnly, "刻意不照做：PLTE 項數超過 1<<位元深度，Go 拒收"),
    ("pixel-too-much", Side::RustOnly, "刻意不照做：解壓後多出像素資料，Go 拒收，png crate 忽略"),
    ("zlib-no-adler", Side::RustOnly, "刻意不照做：缺 zlib Adler-32 尾碼，Go 拒收，png crate 輸出填滿即停"),
    ("stream-no-adler", Side::RustOnly, "刻意不照做：同 zlib-no-adler，隨機串流版"),
    ("stream-bitflip", Side::RustOnly, "刻意不照做：Adler-32 不符，Go 拒收，png crate 輸出填滿後不再驗校驗和"),
    ("garbage-*", Side::RustOnly, "刻意不照做：zlib 串流後的垃圾位元組，Go 視 bufio 是否已讀走而拒收，png crate 忽略"),
    ("stream-garbage", Side::RustOnly, "刻意不照做：同 garbage-*，隨機串流版"),
];

fn matches_pattern(pattern: &str, kind: &str) -> bool {
    match pattern.strip_suffix('*') {
        Some(prefix) => kind.starts_with(prefix),
        None => kind == pattern,
    }
}

fn known_difference(kind: &str, side: Side) -> Option<&'static str> {
    KNOWN_DIFFERENCES
        .iter()
        .find_map(|&(pattern, s, reason)| (s == side && matches_pattern(pattern, kind)).then_some(reason))
}

/// 畸形 PNG 的接受與否統計。`by_kind` 的值是（Go 接受 Rust 拒收, Go 拒收 Rust 接受）。
#[derive(Default)]
struct Mismatches {
    by_kind: BTreeMap<String, (usize, usize)>,
    agree: BTreeMap<&'static str, usize>,
}

fn check_separated(cases: &[Value], stats: &mut Stats, failures: &mut Vec<String>, mm: &mut Mismatches) {
    let base = std::env::temp_dir().join(format!("fd2-golden-dato-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&base);
    for case in cases {
        let dir = base.join(format!("case{:04}", case["index"].as_u64().unwrap()));
        std::fs::create_dir_all(&dir).unwrap();
        for f in case["files"].as_array().unwrap() {
            let path = dir.join(f["name"].as_str().unwrap());
            if f["dir"].as_bool() == Some(true) {
                std::fs::create_dir_all(&path).unwrap();
            } else {
                std::fs::write(&path, unhex(f["hex"].as_str().unwrap_or(""))).unwrap();
            }
        }
        let root = case["root"].as_str().unwrap();
        let call_root =
            if root.is_empty() { PathBuf::new() } else { PathBuf::from(format!("{}/{root}", dir.display())) };
        let got = load_separated_resource(&call_root, i(&case["resource"]));
        let kind = case["kind"].as_str().unwrap();
        let want = &case["result"];
        if exact_kind(kind) {
            let result = frames_result("separated", want, &got, stats);
            if let Some(d) = first_diff("result", want, &result) {
                failures.push(format!("separated case {} ({kind}): {d}", case["index"]));
            }
            continue;
        }
        // 畸形 PNG：只比接受或拒收。
        let go_accept = want["err"].is_null();
        match &got {
            Ok(_) => stats.count("separated-malformed", "rust-accept"),
            Err(e) => stats.count("separated-malformed", &format!("rust-reject:{}", err_view(e).class)),
        }
        match (go_accept, &got) {
            (true, Ok(frames)) => {
                *mm.agree.entry("both-accept").or_default() += 1;
                if let Some(d) = first_diff("result.frames", &want["frames"], &frames_json(frames)) {
                    failures.push(format!("separated case {} ({kind}) 兩邊都接受但幀不同: {d}", case["index"]));
                }
            }
            (false, Err(_)) => *mm.agree.entry("both-reject").or_default() += 1,
            (go_accept, _) => {
                let slot = mm.by_kind.entry(kind.to_string()).or_default();
                if go_accept {
                    slot.0 += 1;
                } else {
                    slot.1 += 1;
                }
                let side = if go_accept { Side::GoOnly } else { Side::RustOnly };
                if known_difference(kind, side).is_none() {
                    failures.push(format!(
                        "separated case {} ({kind}): 接受與否不同 go={} rust={}，且不在 KNOWN_DIFFERENCES",
                        case["index"],
                        want["err"],
                        got.as_ref().err().map_or("接受".to_string(), |e| e.to_string())
                    ));
                }
            }
        }
    }
    for &(pattern, side, _) in KNOWN_DIFFERENCES {
        let seen = mm.by_kind.iter().any(|(k, &(go_only, rust_only))| {
            matches_pattern(pattern, k) && if side == Side::GoOnly { go_only > 0 } else { rust_only > 0 }
        });
        if !seen {
            failures.push(format!("KNOWN_DIFFERENCES 的 {pattern} 沒有出現不一致，應移除"));
        }
    }
    std::fs::remove_dir_all(&base).unwrap();
}

/// 真實檔案必須與黃金檔記錄的名稱、大小與 SHA-256 相同。
fn verify_file(dir: &Path, info: &Value) -> PathBuf {
    let path = dir.join(info["name"].as_str().unwrap());
    let data = std::fs::read(&path).unwrap_or_else(|e| panic!("讀 {}：{e}", path.display()));
    assert_eq!(data.len() as u64, info["size"].as_u64().unwrap(), "{} 大小", path.display());
    assert_eq!(digest(&data), info["sha256"].as_str().unwrap(), "{} SHA-256", path.display());
    path
}

fn real_dir(var: &str) -> PathBuf {
    PathBuf::from(std::env::var_os(var).unwrap_or_else(|| panic!("黃金檔含真實檔案案例，需要 {var}")))
}

fn check_real_dat(doc: &Value, stats: &mut Stats, failures: &mut Vec<String>) -> PathBuf {
    let info = &doc["real_dat"];
    let path = verify_file(&real_dir("FD2_FLAME2_DIR"), info);
    for row in info["resources"].as_array().unwrap() {
        let resource = i(&row["resource"]);
        let got = decode_resource(&path, resource);
        let result = frames_result("real-dat", &row["result"], &got, stats);
        if let Some(d) = first_diff("result", &row["result"], &result) {
            failures.push(format!("DATO.DAT #{resource}: {d}"));
        }
        if let Some(want) = row.get("entry_sha256") {
            let raw = read_resource(&path, resource as usize).unwrap();
            if digest(&raw) != want.as_str().unwrap() {
                failures.push(format!("DATO.DAT #{resource}: entry_sha256 go={want} rust={}", digest(&raw)));
            }
        }
    }
    path
}

fn check_real_separated(doc: &Value, dat: &Path, stats: &mut Stats, failures: &mut Vec<String>) {
    let info = &doc["real_separated"];
    let dir = real_dir("FD2_PORTRAITS_DIR");
    let mut listed: Vec<String> =
        std::fs::read_dir(&dir).unwrap().map(|e| e.unwrap().file_name().into_string().unwrap()).collect();
    listed.sort();
    let recorded: Vec<String> =
        info["files"].as_array().unwrap().iter().map(|f| f["name"].as_str().unwrap().to_string()).collect();
    assert_eq!(listed, recorded, "分離頭像目錄的檔案清單與黃金檔不同");
    for f in info["files"].as_array().unwrap() {
        verify_file(&dir, f);
    }
    for row in info["resources"].as_array().unwrap() {
        let resource = i(&row["resource"]);
        let got = load_separated_resource(&dir, resource);
        let mut result = frames_result("real-separated", &row["result"], &got, stats);
        if let (Ok(frames), Some(_)) = (&got, row.get("equals_archive")) {
            let equal = decode_resource(dat, resource).is_ok_and(|want| want == *frames);
            result.as_object_mut().unwrap().insert("equals_archive".into(), json!(equal));
        }
        let mut want = row["result"].clone();
        if let Some(e) = row.get("equals_archive") {
            want.as_object_mut().unwrap().insert("equals_archive".into(), e.clone());
        }
        if let Some(d) = first_diff("result", &want, &result) {
            failures.push(format!("portrait #{resource}: {d}"));
        }
    }
}

#[test]
fn matches_frozen_go_golden() {
    let Ok(path) = std::env::var("FD2_GOLDEN_DATO") else {
        assert!(std::env::var("FD2_REQUIRE_GOLDEN").is_err(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_DATO");
        eprintln!("略過：未設定 FD2_GOLDEN_DATO");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-dato/1");
    let mut stats = Stats::default();
    let mut failures = Vec::new();
    let mut kinds = BTreeMap::<String, BTreeMap<String, usize>>::new();
    for section in ["parse", "blit", "mouth", "separated"] {
        let cases = doc[section].as_array().unwrap();
        assert!(!cases.is_empty(), "{section} 沒有案例");
        for case in cases {
            let kind = case["kind"].as_str().unwrap();
            // 隨機類保留完整名稱（看得出變形方式），結構化案例只取前綴歸類。
            let head = kind.split_once('-').map_or(kind, |(head, _)| head);
            let kind = if matches!(head, "random" | "stream" | "shaped") { kind } else { head };
            *kinds.entry(section.into()).or_default().entry(kind.into()).or_default() += 1;
        }
    }
    check_parse(doc["parse"].as_array().unwrap(), &mut stats, &mut failures);
    check_blit(doc["blit"].as_array().unwrap(), &mut stats, &mut failures);
    check_mouth(doc["mouth"].as_array().unwrap(), &mut stats, &mut failures);
    let mut mismatches = Mismatches::default();
    check_separated(doc["separated"].as_array().unwrap(), &mut stats, &mut failures, &mut mismatches);
    let dat = check_real_dat(&doc, &mut stats, &mut failures);
    check_real_separated(&doc, &dat, &mut stats, &mut failures);
    eprintln!("案例類別：{kinds:#?}");
    eprintln!(
        "真實檔：DATO.DAT 資源列 {}，分離頭像檔 {}、資源列 {}",
        doc["real_dat"]["resources"].as_array().unwrap().len(),
        doc["real_separated"]["files"].as_array().unwrap().len(),
        doc["real_separated"]["resources"].as_array().unwrap().len()
    );
    eprintln!("錯誤比對：逐字 {}，只比前綴 {}", stats.exact_err, stats.prefix_err);
    eprintln!("路徑覆蓋：{:#?}", stats.coverage);
    eprintln!("畸形 PNG 接受與否一致：{:?}", mismatches.agree);
    eprintln!("畸形 PNG 接受與否不同（類別: (Go 接受 Rust 拒收, Go 拒收 Rust 接受)）：{:?}", mismatches.by_kind);
    assert!(failures.is_empty(), "{} 個案例不符：\n{}", failures.len(), failures.join("\n"));
}
