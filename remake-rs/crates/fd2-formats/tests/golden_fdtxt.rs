//! 與凍結 Go 版 `remake/cmd/fd2-golden-fdtxt` 的黃金差分。
//!
//! 由 `tools/rust_golden_fdtxt.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_FDTXT`。設定
//! `FD2_REQUIRE_GOLDEN=1` 時缺檔即失敗；真實原版檔由 `FD2_FLAME2_DIR`（FLAME2 目錄）與
//! `FD2_SEPARATED_DIR`（`remake/generated-assets/fd2-original-b97caf22`）提供，先核對
//! SHA-256 再比對，`FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗。
//!
//! 比對方式：
//!
//! - 原版 .DAT、真實分離素材、字表與字型的合成輸入，以及格式正確的分離素材與 PNG
//!   （案例 `compare` 為 `exact`）逐欄比對；包裝的檔案系統錯誤只比外層前綴與種類。
//! - 壞 JSON、壞 PNG 與檔案系統邊界（`compare` 為 `accept_reject`）只比對接受或拒收。Rust 版改用
//!   serde_json、png crate 與 `std::path`，刻意不照搬 Go 標準函式庫對壞輸入的寬鬆處理。
//! - 仍不一致的案例逐項列在 [`KNOWN_DIFFERENCES`]，每項都標「刻意不照做」並寫明原因；清單中的
//!   項目若不再出現差異，測試同樣失敗，避免清單過時。

use fd2_formats::archive;
use fd2_formats::fdtxt::{self, FdtxtError, Font, NativeGlyphStyle, NativePresenter, Strings};
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::error::Error;
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

fn err_text<T>(r: &Result<T, FdtxtError>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

fn err_info(err: Option<&FdtxtError>) -> Map<String, Value> {
    let mut out = Map::new();
    let Some(e) = err else {
        out.insert("ok".into(), json!(true));
        return out;
    };
    let msg = e.to_string();
    let (kind, stable) = match e.source() {
        None => ("none", msg.clone()),
        Some(inner) => {
            let stable = msg.strip_suffix(&inner.to_string()).expect("外層訊息以內層結尾").to_string();
            let kind = match e {
                FdtxtError::SeparatedIo { source, .. }
                | FdtxtError::FontMetadataIo(source)
                | FdtxtError::AtlasOpen(source)
                | FdtxtError::AtlasRead(source) => {
                    if source.kind() == std::io::ErrorKind::NotFound {
                        "not_exist"
                    } else {
                        "io"
                    }
                }
                FdtxtError::SeparatedJson { source, .. } | FdtxtError::FontJson(source) => match source.classify() {
                    serde_json::error::Category::Data => "json_type",
                    serde_json::error::Category::Io => "io",
                    _ => "json_syntax",
                },
                _ => "wrapped",
            };
            (kind, stable)
        }
    };
    out.insert("ok".into(), json!(false));
    out.insert("kind".into(), json!(kind));
    out.insert("stable".into(), json!(stable));
    out.insert("err".into(), if kind == "not_exist" || kind == "io" { Value::Null } else { json!(msg) });
    out
}

fn strings_summary(s: &Strings) -> Value {
    let words: Vec<Vec<u16>> = (0..s.count() as i64).map(|i| s.words(i).unwrap()).collect();
    let mut locate = Vec::new();
    let locate_end = loop {
        match s.locate_logical_utterance(locate.len() as i64) {
            Ok(l) => locate.push([l.raw_string_index, l.raw_utterance_index]),
            Err(e) => break e.to_string(),
        }
    };
    json!({
        "count": s.count(), "words": words,
        "words_neg_err": err_text(&s.words(-1)), "words_end_err": err_text(&s.words(s.count() as i64)),
        "locate": locate, "locate_end_err": locate_end,
        "locate_neg_err": err_text(&s.locate_logical_utterance(-1)),
    })
}

/// Go 版 Parse 遇到奇數結尾會 panic；Rust 移植同樣 panic，這裡接住後記成 `{"panic": true}`。
fn parse_summary(data: &[u8]) -> Value {
    let Ok(result) = std::panic::catch_unwind(|| fdtxt::parse(data)) else { return json!({ "panic": true }) };
    let mut out = err_info(result.as_ref().err());
    if let Ok(s) = &result {
        out.insert("strings".into(), strings_summary(s));
    }
    Value::Object(out)
}

fn font_bits(f: &Font) -> Vec<u8> {
    let mut bits = vec![0u8; f.glyph_count() * fdtxt::GLYPH_BYTES];
    for g in 0..f.glyph_count() {
        for y in 0..16 {
            for x in 0..16 {
                if f.glyph_bit(g as i64, x as i64, y as i64).unwrap() {
                    bits[g * 32 + y * 2 + x / 8] |= 0x80 >> (x % 8);
                }
            }
        }
    }
    bits
}

fn font_summary(f: &Font, with_hex: bool) -> Value {
    let n = f.glyph_count() as i64;
    let probes: Vec<Value> = [(-1, 0, 0), (n, 0, 0), (0, -1, 0), (0, 16, 0), (0, 0, -1), (0, 0, 16)]
        .iter()
        .map(|&(g, x, y)| err_text(&f.glyph_bit(g, x, y)))
        .collect();
    let bits = font_bits(f);
    let mut out = json!({ "glyph_count": n, "bits_sha256": digest(&bits), "probe_errs": probes });
    if with_hex {
        out["bits_hex"] = json!(hex(&bits));
    }
    out
}

fn run_blit(f: &Font, op: &Value) -> Value {
    let mut dst = unhex(op["dst_hex"].as_str().unwrap());
    let i = |k: &str| op[k].as_i64().unwrap();
    let style = &op["style"];
    let style = NativeGlyphStyle {
        foreground: style[0].as_u64().unwrap() as u8,
        shadow: style[1].as_u64().unwrap() as u8,
        background: style[2].as_u64().unwrap() as u8,
    };
    let calls = Arc::new(Mutex::new(Vec::<[i64; 6]>::new()));
    let mode = op["presenter"].as_str().unwrap().to_string();
    let font = match mode.as_str() {
        "none" => f.clone(),
        "nil" => f.with_native_presenter(None),
        _ => {
            let calls = Arc::clone(&calls);
            let present: NativePresenter = Arc::new(move |stride, base, index, s: NativeGlyphStyle| {
                calls.lock().unwrap().push([
                    stride,
                    base,
                    index,
                    i64::from(s.foreground),
                    i64::from(s.shadow),
                    i64::from(s.background),
                ]);
                match mode.as_str() {
                    "true" => true,
                    "parity" => (stride + base + index) % 2 == 0,
                    _ => false,
                }
            });
            f.with_native_presenter(Some(present))
        }
    };
    let result = if op["op"] == "blit" {
        font.blit_glyph(&mut dst, i("stride"), i("base"), i("index"), i("color") as u8)
    } else {
        font.blit_native_glyph(&mut dst, i("stride"), i("base"), i("index"), style)
    };
    let calls = calls.lock().unwrap().clone();
    json!({ "err": err_text(&result), "dst_sha256": digest(&dst), "calls": calls })
}

struct TempDir(PathBuf);

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

fn materialize(case: &Value, tag: &str, index: usize) -> (TempDir, String) {
    let dir = std::env::temp_dir().join(format!("fd2-golden-fdtxt-{tag}-{}-{index}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    for d in case["dirs"].as_array().into_iter().flatten() {
        std::fs::create_dir_all(dir.join(d.as_str().unwrap())).unwrap();
    }
    for f in case["files"].as_array().into_iter().flatten() {
        let path = dir.join(f["path"].as_str().unwrap());
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(path, unhex(f["hex"].as_str().unwrap())).unwrap();
    }
    let root = match case["root"].as_str().unwrap() {
        "empty" => String::new(),
        "tmp" => dir.display().to_string(),
        suffix => format!("{}{suffix}", dir.display()),
    };
    (TempDir(dir), root)
}

fn text_result(root: impl AsRef<Path>, resource: i64) -> Value {
    let result = fdtxt::load_separated_resource(root, resource);
    let mut out = err_info(result.as_ref().err());
    if let Ok(s) = &result {
        out.insert("strings".into(), strings_summary(s));
    }
    Value::Object(out)
}

fn font_result(root: impl AsRef<Path>, real: bool) -> Value {
    let result = fdtxt::load_separated_font(root);
    let mut out = err_info(result.as_ref().err());
    if let Ok(f) = &result {
        out.insert("font".into(), if real { real_font(f) } else { font_summary(f, false) });
    }
    Value::Object(out)
}

fn png_summary(data: &[u8]) -> Value {
    match fdtxt::decode_png_rgba(data) {
        Err(e) => json!({ "ok": false, "err": e.to_string() }),
        Ok(img) => {
            let mut px = Vec::with_capacity(img.width * img.height * 8);
            for y in 0..img.height {
                for x in 0..img.width {
                    for v in img.rgba(x, y) {
                        px.extend_from_slice(&(v as u16).to_be_bytes());
                    }
                }
            }
            json!({ "ok": true, "type": img.go_type, "width": img.width, "height": img.height,
                "rgba_sha256": digest(&px) })
        }
    }
}

fn real_font(f: &Font) -> Value {
    let mut out = font_summary(f, true);
    let blits: Vec<String> = (0..f.glyph_count())
        .map(|g| {
            let mut a: Vec<u8> = (0..20 * 18 + 4).map(|i| (i % 5) as u8).collect();
            let mut b: Vec<u8> = (0..20 * 18 + 4).map(|i| (i % 7) as u8).collect();
            f.blit_glyph(&mut a, 20, 2, g as i64, 0x2a).unwrap();
            let style = NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: (g % 3) as u8 };
            f.blit_native_glyph(&mut b, 20, 21, g as i64, style).unwrap();
            a.extend_from_slice(&b);
            digest(&a)[..16].to_string()
        })
        .collect();
    out["blit_digests"] = json!(blits);
    out
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
        (Value::Array(a), Value::Array(b)) => Some(format!("{path}: length go={} rust={}", a.len(), b.len())),
        _ if want == got => None,
        _ => Some(format!("{path}: go={want} rust={got}")),
    }
}

/// 覆蓋統計：成功，或錯誤訊息（數字換成 N）。
fn outcome(result: &Value) -> String {
    if result.get("panic").is_some() {
        return "panic".into();
    }
    if result["ok"] == true || (result.get("ok").is_none() && result["err"].is_null()) {
        return match result["type"].as_str() {
            Some(t) => format!("ok {t}"),
            None => "ok".into(),
        };
    }
    let text = match result["kind"].as_str() {
        Some(kind @ ("not_exist" | "io" | "json_syntax" | "json_type")) => {
            format!("{}<{kind}>", result["stable"].as_str().unwrap())
        }
        _ => result["err"].as_str().unwrap_or_default().to_string(),
    };
    // 十進位數字換成 N、`0x` 十六進位換成 0xN。
    let bytes = text.as_bytes();
    let mut out = String::new();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i..].starts_with(b"0x") && bytes.get(i + 2).is_some_and(u8::is_ascii_hexdigit) {
            out.push_str("0xN");
            i += 2;
            while bytes.get(i).is_some_and(u8::is_ascii_hexdigit) {
                i += 1;
            }
        } else if bytes[i].is_ascii_digit() {
            out.push('N');
            while bytes.get(i).is_some_and(u8::is_ascii_digit) {
                i += 1;
            }
        } else {
            let ch = text[i..].chars().next().unwrap();
            out.push(ch);
            i += ch.len_utf8();
        }
    }
    out
}

/// 已知差異：Go 標準函式庫對壞輸入的處理與 serde_json／png crate／`std::path` 不同的案例。全部只出現在
/// `accept_reject` 案例，每一項都是刻意不照做。`cases` 的格式為「段落:種類」；種類以 `~` 開頭時表示
/// 種類字串含該標記（手工 PNG 的種類由多個破壞標記組成）。清單中每個樣式都必須至少命中一次。
struct KnownDifference {
    cases: &'static [&'static str],
    reason: &'static str,
}

const KNOWN_DIFFERENCES: &[KnownDifference] = &[
    KnownDifference {
        cases: &[
            "separated-text:fold-upper",
            "separated-text:fold-long-s",
            "separated-text:fold-long-s-raw",
            "separated-text:fold-kelvin",
            "separated-text:fold-mixed",
            "separated-text:fold-tokens",
            "separated-text:fold-glyph-index",
            "separated-font:doc-fold",
        ],
        reason: "刻意不照做：Go encoding/json 鍵名不分大小寫（ſ、K 也算），serde 只認精確鍵名",
    },
    KnownDifference {
        cases: &[
            "separated-text:string-null-dup",
            "separated-text:dup-later-wins",
            "separated-text:dup-source-merge",
            "separated-text:dup-source-merge-keeps",
            "separated-text:stale-merge-fields",
            "separated-text:stale-merge-tokens",
            "separated-text:stale-truncate",
            "separated-text:stale-null-elem",
            "separated-text:stale-regrow",
            "separated-text:stale-tokens-empty",
            "separated-font:doc-dup",
        ],
        reason: "刻意不照做：Go 對重複鍵依序覆寫並把陣列合併進舊元素，serde 對重複欄位直接拒收",
    },
    KnownDifference {
        cases: &["separated-text:deep-10001"],
        reason: "刻意不照做：Go 掃描器限 10000 層巢狀；serde_json 略過未知欄位時不受此限而接受",
    },
    KnownDifference {
        cases: &[
            "separated-text:text-lone-high",
            "separated-text:text-lone-high-then-u",
            "separated-text:text-lone-low",
            "separated-text:text-high-high",
            "separated-text:text-invalid-utf8",
            "separated-text:text-surrogate-utf8",
            "separated-text:text-overlong",
            "separated-text:text-f4-90",
            "separated-text:text-cont-only",
            "separated-text:text-raw-vs-escape-ffd",
        ],
        reason: "刻意不照做：Go 把無效 UTF-8 與孤立代理項換成 U+FFFD，serde_json 拒收",
    },
    KnownDifference {
        cases: &["separated-text:fs-clean-dotdot", "separated-text:fs-clean-deep", "separated-font:fs-clean-dotdot"],
        reason: "刻意不照做：Go filepath.Join 先字面化簡 x/..，std::path 交給作業系統，不存在的中間目錄即失敗",
    },
    KnownDifference {
        cases: &["png:~-adler", "png:~-zflip"],
        reason: "刻意不照做：png crate 不驗 zlib 結尾的 adler32，Go 驗到串流結束",
    },
    KnownDifference {
        cases: &["png:~-long"],
        reason: "刻意不照做：解壓後多出的像素資料 png crate 不理會，Go 報 too much pixel data",
    },
    KnownDifference {
        cases: &["png:~bufio-"],
        reason: "刻意不照做：zlib 串流之後的 IDAT 位元組 png crate 不理會，Go 只放過 bufio 已預讀的部分",
    },
    KnownDifference {
        cases: &["png:~-badtrns"],
        reason: "刻意不照做：tRNS 長度或取值不合色彩型別（含帶 alpha 的型別）時 png crate 不拒收，Go 拒收",
    },
    KnownDifference {
        cases: &["png:~-fdict"],
        reason: "刻意不照做：zlib 預設字典旗標在字典 ID 為 1 時 Go 照常解碼，png crate 一律拒收",
    },
    KnownDifference {
        cases: &["png:~-crc", "png:~-flip", "png:fixed-iend-len"],
        reason: "刻意不照做：png crate 讀完影像即停，不驗 IEND 與輔助區塊的 CRC 及 IEND 長度，Go 逐塊驗到 IEND",
    },
];

fn known_pattern(section: &str, kind: &str) -> Vec<&'static str> {
    KNOWN_DIFFERENCES
        .iter()
        .flat_map(|k| k.cases.iter().copied())
        .filter(|pattern| {
            let (sec, pat) = pattern.split_once(':').unwrap();
            sec == section
                && match pat.strip_prefix('~') {
                    Some(tag) => kind.contains(tag),
                    None => kind == pat,
                }
        })
        .collect()
}

#[derive(Default)]
struct Report {
    failures: Vec<String>,
    coverage: BTreeMap<String, BTreeMap<String, usize>>,
    known_hits: BTreeMap<&'static str, usize>,
}

fn accepted(result: &Value) -> bool {
    result["ok"] == true
}

impl Report {
    fn check(&mut self, section: &str, label: &str, want: &Value, got: &Value) {
        self.compare(section, label, "", "exact", want, got);
    }

    fn compare(&mut self, section: &str, label: &str, kind: &str, mode: &str, want: &Value, got: &Value) {
        let diff = match mode {
            "exact" => first_diff("result", want, got),
            "accept_reject" => (accepted(want) != accepted(got))
                .then(|| format!("accept go={} rust={}（rust: {}）", accepted(want), accepted(got), got)),
            other => Some(format!("未知的比對模式 {other}")),
        };
        *self.coverage.entry(section.into()).or_default().entry(outcome(want)).or_default() += 1;
        let Some(d) = diff else { return };
        let patterns = if mode == "accept_reject" { known_pattern(section, kind) } else { Vec::new() };
        if patterns.is_empty() {
            self.failures.push(format!("{section} {label}: {d}"));
        }
        for pattern in patterns {
            *self.known_hits.entry(pattern).or_default() += 1;
        }
    }

    fn case(&mut self, section: &str, case: &Value, got: &Value) {
        let kind = case["kind"].as_str().unwrap();
        let label = format!("case {} ({kind})", case["index"]);
        let mode = case["compare"].as_str().unwrap_or("exact");
        self.compare(section, &label, kind, mode, &case["result"], got);
    }
}

fn require_original() -> bool {
    std::env::var_os("FD2_REQUIRE_ORIGINAL").is_some()
}

fn read_verified(dir: &Path, rel: &str, sha: &Value, report: &mut Report) -> Option<Vec<u8>> {
    let data = std::fs::read(dir.join(rel)).ok()?;
    if digest(&data) != sha.as_str().unwrap() {
        report.failures.push(format!("real {rel}: SHA-256 與黃金檔記錄不同"));
        return None;
    }
    Some(data)
}

fn check_real(real: &Value, report: &mut Report) {
    match std::env::var_os("FD2_FLAME2_DIR").map(PathBuf::from) {
        None => assert!(!require_original(), "FD2_REQUIRE_ORIGINAL=1 但沒有 FD2_FLAME2_DIR"),
        Some(dir) => {
            let dat = &real["fdtxt_dat"];
            if let Some(data) = read_verified(&dir, "FDTXT.DAT", &dat["sha256"], report) {
                let resources = dat["resources"].as_array().unwrap();
                let count = archive::archive_data_resource_count(&data).unwrap();
                if count != resources.len() {
                    report.failures.push(format!("real FDTXT.DAT: 資源數 go={} rust={count}", resources.len()));
                }
                for row in resources {
                    let r = row["resource"].as_u64().unwrap() as usize;
                    let entry = archive::archive_entry(&data, r).unwrap();
                    if digest(entry) != row["data_sha256"].as_str().unwrap() {
                        report.failures.push(format!("real FDTXT.DAT #{r}: 資源 SHA-256 不同"));
                    }
                    report.check("real-fdtxt", &format!("#{r}"), &row["result"], &parse_summary(entry));
                }
            }
            let font = &real["fdother_font"];
            if read_verified(&dir, "FDOTHER.DAT", &font["sha256"], report).is_some() {
                let data = archive::read_resource(dir.join("FDOTHER.DAT"), 4).unwrap();
                assert_eq!(digest(&data), font["data_sha256"].as_str().unwrap());
                let got = real_font(&fdtxt::parse_font(&data).unwrap());
                report.check("real-fdother-font", "#4", &font["result"], &got);
            }
        }
    }
    match std::env::var_os("FD2_SEPARATED_DIR").map(PathBuf::from) {
        None => assert!(!require_original(), "FD2_REQUIRE_ORIGINAL=1 但沒有 FD2_SEPARATED_DIR"),
        Some(dir) => {
            for row in real["separated_text"].as_array().unwrap() {
                let rel = row["path"].as_str().unwrap();
                match row.get("sha256") {
                    Some(sha) => {
                        read_verified(&dir, rel, sha, report);
                    }
                    None => {
                        if dir.join(rel).exists() {
                            report.failures.push(format!("real {rel}: Go 端沒有這個檔案"));
                        }
                    }
                }
                let r = row["resource"].as_i64().unwrap();
                report.check("real-separated-text", rel, &row["result"], &text_result(dir.join("text"), r));
            }
            let font = &real["separated_font"];
            let json_ok = read_verified(&dir, font["json_path"].as_str().unwrap(), &font["json_sha256"], report);
            let atlas_ok = read_verified(&dir, font["atlas_path"].as_str().unwrap(), &font["atlas_sha256"], report);
            if json_ok.is_some() && atlas_ok.is_some() {
                report.check("real-separated-font", "fonts", &font["result"], &font_result(dir.join("fonts"), true));
            }
            for row in real["png"].as_array().unwrap() {
                let rel = row["path"].as_str().unwrap();
                if let Some(data) = read_verified(&dir, rel, &row["sha256"], report) {
                    report.check("real-png", rel, &row["result"], &png_summary(&data));
                }
            }
        }
    }
}

#[test]
fn matches_frozen_go_golden() {
    let Ok(path) = std::env::var("FD2_GOLDEN_FDTXT") else {
        assert!(std::env::var("FD2_REQUIRE_GOLDEN").is_err(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_FDTXT");
        eprintln!("略過：未設定 FD2_GOLDEN_FDTXT");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-fdtxt/1");
    let label = |c: &Value| format!("case {} ({})", c["index"], c["kind"].as_str().unwrap());
    let mut report = Report::default();

    // Parse 的 panic 案例會觸發預設 panic hook；這一段暫時靜音。
    let hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(|_| {}));
    let parse_results: Vec<(String, Value, Value)> = doc["parse_cases"]
        .as_array()
        .unwrap()
        .iter()
        .map(|c| (label(c), c["result"].clone(), parse_summary(&unhex(c["data_hex"].as_str().unwrap()))))
        .collect();
    std::panic::set_hook(hook);
    for (l, want, got) in &parse_results {
        report.check("parse", l, want, got);
    }

    for c in doc["font_cases"].as_array().unwrap() {
        let result = fdtxt::parse_font(&unhex(c["data_hex"].as_str().unwrap()));
        let mut got = err_info(result.as_ref().err());
        if let Ok(f) = &result {
            got.insert("font".into(), font_summary(f, true));
            for (k, op) in c["blits"].as_array().unwrap().iter().enumerate() {
                report.check("blit", &format!("{} op {k}", label(c)), &op["result"], &run_blit(f, op));
            }
        }
        report.check("font", &label(c), &c["result"], &Value::Object(got));
    }

    for (i, c) in doc["separated_text_cases"].as_array().unwrap().iter().enumerate() {
        let (_dir, root) = materialize(c, "text", i);
        let got = text_result(&root, c["resource"].as_i64().unwrap());
        report.case("separated-text", c, &got);
    }

    for (i, c) in doc["separated_font_cases"].as_array().unwrap().iter().enumerate() {
        let (_dir, root) = materialize(c, "font", i);
        report.case("separated-font", c, &font_result(&root, false));
    }

    for c in doc["png_cases"].as_array().unwrap() {
        let got = png_summary(&unhex(c["png_hex"].as_str().unwrap()));
        report.case("png", c, &got);
    }

    check_real(&doc["real"], &mut report);

    for (section, outcomes) in &report.coverage {
        let total: usize = outcomes.values().sum();
        eprintln!("== {section}：{total} 案例");
        for (o, n) in outcomes {
            eprintln!("   {n:5}  {o}");
        }
    }
    for known in KNOWN_DIFFERENCES {
        eprintln!("已知差異（{}）", known.reason);
        for pattern in known.cases {
            match report.known_hits.get(pattern) {
                Some(n) => eprintln!("   {n:5}  {pattern}"),
                None => report.failures.push(format!("已知差異 {pattern} 沒有發生，請從清單移除")),
            }
        }
    }
    let shown: Vec<&String> = report.failures.iter().take(80).collect();
    assert!(
        report.failures.is_empty(),
        "{} 個比對不符（列出前 {}）：\n{}",
        report.failures.len(),
        shown.len(),
        shown.iter().map(|s| s.as_str()).collect::<Vec<_>>().join("\n")
    );
}
