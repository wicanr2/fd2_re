//! 與凍結 Go 版 `remake/cmd/fd2-golden-afm` 的黃金差分。
//!
//! 由 `tools/rust_golden_afm.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_AFM`；原版檔目錄以
//! `FD2_AFM_ORG_DIR`（含 ANI.DAT）與 `FD2_AFM_ANIMATIONS_DIR`（分離動畫）傳入。
//! 設定 `FD2_REQUIRE_GOLDEN=1` 時缺任何一項即失敗，避免把「沒跑到」當成通過。
//!
//! 每個案例在暫存目錄重建同一棵檔案樹（合成檔案取自 JSON 的 hex，衍生檔案讀原版檔後套用
//! 相同變形），呼叫同一個公開 API，結果逐欄比對。錯誤字串內的暫存路徑換成 `$ROOT`。
//!
//! 比對規則：
//! - ANI.DAT、VM、容器與 decode 案例完全比對（含 Go 端 panic）；只有不以 `afm:` 開頭的作業系統
//!   錯誤原文不比。
//! - 原版分離素材完全比對；其他分離動畫案例中 Go 接受者，重製版必須接受且欄位與像素雜湊相同。
//! - 兩邊都拒收時，afm 自己的錯誤全文要相同；包了 JSON／PNG／OS 原因的錯誤只比拒收。
//! - 一邊接受一邊拒收的案例必須逐條列在 [`KNOWN_DIFFERENCES`]（刻意不照做）。

use fd2_formats::afm::*;
use serde_json::{Value, json};
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

/// 測試端自備的 CRC-32（IEEE），與函式庫內部實作無關。
fn crc32(data: &[u8]) -> u32 {
    let mut c = 0xffff_ffffu32;
    for &b in data {
        c ^= b as u32;
        for _ in 0..8 {
            c = if c & 1 != 0 { 0xedb8_8320 ^ (c >> 1) } else { c >> 1 };
        }
    }
    !c
}

fn be32(b: &[u8]) -> u32 {
    u32::from_be_bytes([b[0], b[1], b[2], b[3]])
}

fn le32(b: &[u8]) -> usize {
    u32::from_le_bytes([b[0], b[1], b[2], b[3]]) as usize
}

/// 與 Go 匯出工具 `fixCRC` 相同：依序重算每個完整區塊的 CRC。
fn fix_crc(b: &mut [u8]) {
    let mut pos = 8;
    while pos + 8 <= b.len() {
        let n = be32(&b[pos..]);
        let end = pos + 8 + n as usize;
        if n > 0x7fff_ffff || end + 4 > b.len() {
            return;
        }
        let crc = crc32(&b[pos + 4..end]);
        b[end..end + 4].copy_from_slice(&crc.to_be_bytes());
        pos = end + 4;
    }
}

fn extract_resource(b: &[u8], res: usize) -> Vec<u8> {
    let n = (le32(&b[6..]) - 6) / 4;
    let start = le32(&b[6 + 4 * res..]);
    let end = if res + 1 < n { le32(&b[6 + 4 * (res + 1)..]) } else { b.len() };
    b[start..end].to_vec()
}

fn container(resource: &[u8]) -> Vec<u8> {
    let mut out = b"LLLLLL".to_vec();
    out.extend_from_slice(&10u32.to_le_bytes());
    out.extend_from_slice(resource);
    out
}

fn apply_ops(data: &[u8], ops: &[Value]) -> Vec<u8> {
    let mut b = data.to_vec();
    for op in ops {
        let num = |k: &str| op[k].as_u64().unwrap() as usize;
        match op["op"].as_str().unwrap() {
            "xor" => b[num("at")] ^= num("val") as u8,
            "set16" => {
                let at = num("at");
                b[at..at + 2].copy_from_slice(&(num("val") as u16).to_le_bytes());
            }
            "truncate" => b.truncate(num("n")),
            "insert" => {
                let at = num("at");
                let ins = unhex(op["hex"].as_str().unwrap());
                b.splice(at..at, ins);
            }
            "delete" => {
                let at = num("at");
                b.drain(at..at + num("n"));
            }
            "fixcrc" => fix_crc(&mut b),
            "extract" => b = extract_resource(&b, num("n")),
            "wrap" => b = container(&b),
            other => panic!("未知變形 {other}"),
        }
    }
    b
}

struct Env {
    org: String,
    anim: String,
}

impl Env {
    fn root(&self, name: &str) -> &str {
        match name {
            "org" => &self.org,
            "anim" => &self.anim,
            other => panic!("未知原版目錄 {other}"),
        }
    }

    fn norm(&self, base: &str, s: &str) -> String {
        s.replace(base, "$ROOT").replace(&self.org, "$ORG").replace(&self.anim, "$ANIM")
    }

    fn expand(&self, base: &str, p: &str) -> String {
        p.replace("$ROOT", base).replace("$ORG", &self.org).replace("$ANIM", &self.anim)
    }

    fn materialize(&self, base: &Path, files: &[Value]) {
        std::fs::create_dir_all(base).unwrap();
        for f in files {
            let full = base.join(f["path"].as_str().unwrap());
            if f["dir"].as_bool().unwrap_or(false) {
                std::fs::create_dir_all(&full).unwrap();
                continue;
            }
            std::fs::create_dir_all(full.parent().unwrap()).unwrap();
            let bytes = if f["from"].is_object() {
                let from = &f["from"];
                let src = Path::new(self.root(from["root"].as_str().unwrap())).join(from["path"].as_str().unwrap());
                apply_ops(&std::fs::read(src).unwrap(), from["ops"].as_array().unwrap())
            } else {
                unhex(f["hex"].as_str().unwrap_or(""))
            };
            std::fs::write(&full, bytes).unwrap();
        }
    }
}

fn clip_json(c: &Clip) -> Value {
    let frames: Vec<Value> = c
        .frames
        .iter()
        .enumerate()
        .map(|(i, f)| {
            json!({
                "x0": 0, "y0": 0, "w": f.width, "h": f.height, "stride": 4 * f.width,
                "rgba": digest(&f.pix), "indexed": digest(&c.indexed_frames[i]), "palette": digest(&c.palettes[i]),
            })
        })
        .collect();
    json!({
        "title_hex": hex(&c.title), "header_frames": c.header_frames,
        "n_frames": c.frames.len(), "n_indexed": c.indexed_frames.len(), "n_palettes": c.palettes.len(),
        "frames": frames,
    })
}

fn guarded(f: impl FnOnce() -> Value) -> Value {
    catch_unwind(AssertUnwindSafe(f)).unwrap_or_else(|_| json!({ "panic": true }))
}

fn run_case(env: &Env, scratch: &Path, index: usize, case: &Value) -> Value {
    let base_path = scratch.join(format!("case-{index:04}"));
    env.materialize(&base_path, case["files"].as_array().unwrap());
    let base = base_path.to_str().unwrap().to_string();
    let path = env.expand(&base, case["path"].as_str().unwrap());
    let resources: Vec<i64> = case["resources"].as_array().unwrap().iter().map(|v| v.as_i64().unwrap()).collect();
    let err_value = |e: &AfmError| Value::String(env.norm(&base, &e.to_string()));
    let mut result = serde_json::Map::new();
    match case["call"].as_str().unwrap() {
        "archive" => {
            let count = archive_resource_count(&path);
            result.insert(
                "count".into(),
                match &count {
                    Ok(n) => json!({ "n": n, "err": null }),
                    Err(e) => json!({ "n": 0, "err": err_value(e) }),
                },
            );
            let mut reads = Vec::new();
            let mut decodes = Vec::new();
            for &res in &resources {
                let read = read_archive_resource(&path, res);
                let raw = read.as_deref().unwrap_or(&[]);
                reads.push(json!({
                    "res": res, "len": raw.len(), "sha256": digest(raw),
                    "err": read.as_ref().err().map_or(Value::Null, err_value),
                }));
                decodes.push(guarded(|| match decode_resource(&path, res) {
                    Ok(clip) => json!({ "res": res, "err": null, "clip": clip_json(&clip) }),
                    Err(e) => json!({ "res": res, "err": err_value(&e) }),
                }));
            }
            result.insert("read".into(), Value::Array(reads));
            result.insert("decode".into(), Value::Array(decodes));
        }
        "separated" => {
            let loads = resources
                .iter()
                .map(|&res| {
                    guarded(|| match load_separated_resource(&path, res) {
                        Ok(clip) => json!({ "res": res, "err": null, "clip": clip_json(&clip) }),
                        Err(e) => json!({ "res": res, "err": err_value(&e) }),
                    })
                })
                .collect();
            result.insert("load".into(), Value::Array(loads));
        }
        other => panic!("未知呼叫 {other}"),
    }
    std::fs::remove_dir_all(&base_path).unwrap();
    Value::Object(result)
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

/// 錯誤字串只比 afm 自己的部分：`afm: separated <metadata|frame|palette> N: <原因>` 的原因、
/// 以及不以 `afm:` 開頭的作業系統錯誤，都來自 JSON／PNG／OS 函式庫，換成佔位字。
fn owned(err: &str) -> String {
    if !err.starts_with("afm:") {
        return "<os>".into();
    }
    for head in ["afm: separated metadata ", "afm: separated frame ", "afm: separated palette "] {
        if let Some(rest) = err.strip_prefix(head) {
            let digits = rest.bytes().take_while(|b| b.is_ascii_digit() || *b == b'-').count();
            if rest[digits..].starts_with(": ") {
                return format!("{head}{}: <cause>", &rest[..digits]);
            }
        }
    }
    err.to_string()
}

fn normalize(v: &Value) -> Value {
    match v {
        Value::Object(m) => Value::Object(
            m.iter()
                .map(|(k, x)| match (k.as_str(), x) {
                    ("err", Value::String(s)) => (k.clone(), Value::String(owned(s))),
                    _ => (k.clone(), normalize(x)),
                })
                .collect(),
        ),
        Value::Array(a) => Value::Array(a.iter().map(normalize).collect()),
        other => other.clone(),
    }
}

/// 刻意不照做：分離動畫的畸形輸入上，Go 標準庫（encoding/json、image/png、filepath）與重製版
/// （serde_json、png crate、std::path）接受與否不同的案例。依原因分組，每組列出 (案例種類,
/// 案例說明)；未列入的接受／拒收差異、或列了卻已一致的案例都算失敗。說明見 `afm/separated.rs`。
const KNOWN_DIFFERENCES: &[(&str, &[(&str, &str)])] = &[
    (
        "刻意不照做：孤立 UTF-16 代理與無效 UTF-8，Go 換成 U+FFFD 後接受，serde_json 拒收",
        &[
            ("separated-metadata", r#"title raw "\"\\ud800\"""#),
            ("separated-metadata", r#"title raw "\"\\udc00x\"""#),
            ("separated-metadata", r#"title raw "\"\\ud800A\"""#),
            ("separated-metadata", r#"title raw "\"\\ud800\\ud800\"""#),
            ("separated-metadata", r#"title raw "\"\xff\xfe\"""#),
            ("separated-metadata", r#"title raw "\"\xe4\xb8\"""#),
            ("separated-metadata", r#"title raw "\"\xf0\x9f\x98\"""#),
            ("separated-metadata", r#"title raw "\"\xed\xa0\x80\"""#),
            ("separated-metadata", r#"title raw "\"\xc0\xaf\"""#),
        ],
    ),
    (
        "刻意不照做：-0，Go 當整數 0 接受，serde_json 視為浮點數拒收",
        &[("separated-metadata", "frames[0].source_frame=-0"), ("separated-palette", "component=-0")],
    ),
    (
        "刻意不照做：null，Go 保留原值（零值）接受，serde 拒收",
        &[("separated-metadata", "frames[0].source_frame=null"), ("separated-palette", "component=null")],
    ),
    (
        "刻意不照做：鍵名不分大小寫與 Unicode 摺疊（ſ→S、K→K），Go 對上欄位，serde 精確比對",
        &[
            ("separated-metadata", "key SCHEMA_VERSION"),
            ("separated-metadata", "key Kind"),
            ("separated-metadata", "key long-s status"),
            ("separated-metadata", "key escaped long-s"),
            ("separated-metadata", "key kelvin kind"),
            ("separated-metadata", "dup kind folded later"),
            ("separated-palette", "key folded"),
        ],
    ),
    (
        "刻意不照做：重複鍵，Go 後者覆蓋、巢狀合併並重用切片舊元素，serde 拒收",
        &[
            ("separated-metadata", "dup kind bad then good"),
            ("separated-metadata", "dup source partial merge"),
            ("separated-metadata", "dup source empty"),
            ("separated-metadata", "dup source null"),
            ("separated-metadata", "stale slice elements resurface"),
            ("separated-metadata", "frames of nulls after full"),
            ("separated-palette", "stale components"),
        ],
    ),
    (
        "刻意不照做：巢狀深度，Go 超過 10000 層拒收，serde_json 略過未知欄位時不限深度",
        &[("separated-metadata", "deep unknown 10001")],
    ),
    (
        "刻意不照做：路徑，Go filepath.Join 先字面消去 sub/..，std::path 交給作業系統",
        &[("separated-fs", "root with sub/..")],
    ),
    (
        "刻意不照做：IHDR 前的輔助區塊與未知關鍵區塊，Go 略過，png crate 拒收",
        &[("separated-png", "unknown critical chunk"), ("separated-png", "chunk before IHDR")],
    ),
    ("刻意不照做：zlib 字典 id=1，Go 視同無字典接受，png crate 拒收", &[("separated-png", "zlib FDICT dictid 1")]),
    (
        "刻意不照做：PLTE／tRNS 的順序與長度，Go 拒收，png crate 接受（重製版不用 PNG 調色盤）",
        &[
            ("separated-png", "tRNS 257"),
            ("separated-png", "tRNS before PLTE"),
            ("separated-png", "tRNS twice"),
            ("separated-png", "IDAT before PLTE"),
            ("separated-png", "no PLTE"),
            ("separated-png", "p1 PLTE 3 entries"),
        ],
    ),
    (
        "刻意不照做：zlib 串流後的多餘資料或多餘像素列，Go 回 too much pixel data，png crate 忽略",
        &[
            ("separated-png", "too much pixel data +1"),
            ("separated-png", "too much pixel data +7"),
            ("separated-png", "junk 4095 after zlib split []"),
            ("separated-png", "junk 4095 after zlib split [3000]"),
            ("separated-png", "junk 4095 after zlib split [4096]"),
            ("separated-png", "junk 4096 after zlib split []"),
            ("separated-png", "junk 4096 after zlib split [3000]"),
            ("separated-png", "junk 4096 after zlib split [4096]"),
            ("separated-png", "junk 5000 after zlib split []"),
            ("separated-png", "junk 5000 after zlib split [3000]"),
            ("separated-png", "junk 5000 after zlib split [4096]"),
            ("separated-png-random", "xor@5196,fixcrc"),
        ],
    ),
    ("刻意不照做：Adler-32 只剩 2 位元組，Go 拒收，png crate 接受", &[("separated-png", "zlib adler cut")]),
];

fn known_reason(kind: &str, note: &str) -> Option<&'static str> {
    KNOWN_DIFFERENCES.iter().find(|(_, cases)| cases.contains(&(kind, note))).map(|(reason, _)| *reason)
}

enum Outcome {
    Same,
    /// 兩邊都拒收，但一邊是函式庫錯誤（JSON／PNG／OS），另一邊是 afm 契約錯誤。
    ReasonOnly,
    AcceptReject(String),
}

/// 分離動畫案例逐列比對。兩邊都接受時所有欄位與像素雜湊必須相同；兩邊都拒收時，若兩邊都是
/// afm 自己的錯誤（不帶下層原因）就比全文，否則只比拒收；一邊接受一邊拒收回 `AcceptReject`。
fn compare_separated(want: &Value, got: &Value) -> std::result::Result<Outcome, String> {
    let (w, g) = (want["load"].as_array().unwrap(), got["load"].as_array().unwrap());
    if w.len() != g.len() {
        return Err("load 列數不同".into());
    }
    let mut outcome = Outcome::Same;
    for (i, (w, g)) in w.iter().zip(g).enumerate() {
        if w["panic"].is_null() != g["panic"].is_null() {
            return Err(format!("load[{i}] panic 不同"));
        }
        match (w["err"].as_str(), g["err"].as_str()) {
            (None, None) => {
                if let Some(d) = first_diff(&format!("load[{i}]"), w, g) {
                    return Err(format!("兩邊都接受但內容不同：{d}"));
                }
            }
            (Some(we), Some(ge)) => {
                let (wo, go) = (owned(we), owned(ge));
                let layer = |s: &str| s.ends_with("<cause>") || s == "<os>";
                if wo != go {
                    if !layer(&wo) && !layer(&go) {
                        return Err(format!("load[{i}] afm 錯誤文字不同：go={we:?} rust={ge:?}"));
                    }
                    if !matches!(outcome, Outcome::AcceptReject(_)) {
                        outcome = Outcome::ReasonOnly;
                    }
                }
            }
            _ => outcome = Outcome::AcceptReject(format!("load[{i}]: go={} rust={}", w["err"], g["err"])),
        }
    }
    Ok(outcome)
}

/// 覆蓋統計的分類鍵：路徑換成 `<p>`、數字換成 `#`。
fn category(row: &Value) -> String {
    if row["panic"].as_bool() == Some(true) {
        return "panic".into();
    }
    let Some(err) = row["err"].as_str() else {
        return "ok".into();
    };
    let mut out = String::new();
    let mut chars = err.chars().peekable();
    while let Some(c) = chars.next() {
        if c == '$' {
            while chars.peek().is_some_and(|&n| n != ':') {
                chars.next();
            }
            out.push_str("<p>");
        } else if c.is_ascii_digit() {
            while chars.peek().is_some_and(|n| n.is_ascii_digit()) {
                chars.next();
            }
            out.push('#');
        } else {
            out.push(c);
        }
    }
    out
}

fn env_dir(name: &str) -> Option<String> {
    std::env::var(name).ok().filter(|p| Path::new(p).is_dir())
}

#[test]
fn matches_frozen_go_golden() {
    let require = std::env::var("FD2_REQUIRE_GOLDEN").is_ok();
    let Ok(golden) = std::env::var("FD2_GOLDEN_AFM") else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_AFM");
        eprintln!("略過：未設定 FD2_GOLDEN_AFM");
        return;
    };
    let (Some(org), Some(anim)) = (env_dir("FD2_AFM_ORG_DIR"), env_dir("FD2_AFM_ANIMATIONS_DIR")) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但 FD2_AFM_ORG_DIR／FD2_AFM_ANIMATIONS_DIR 不存在");
        eprintln!("略過：缺原版檔目錄");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&golden).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-afm/1");
    let env = Env { org: org.trim_end_matches('/').to_string(), anim: anim.trim_end_matches('/').to_string() };

    // 原版檔必須與 Go 端讀到的是同一份。
    let mut real_count = 0;
    for root in ["org", "anim"] {
        for f in doc["real"][root].as_array().unwrap() {
            let p = Path::new(env.root(root)).join(f["path"].as_str().unwrap());
            let bytes = std::fs::read(&p).unwrap_or_else(|e| panic!("讀原版檔 {}: {e}", p.display()));
            assert_eq!(bytes.len() as u64, f["size"].as_u64().unwrap(), "{} 大小不同", p.display());
            assert_eq!(digest(&bytes), f["sha256"].as_str().unwrap(), "{} SHA-256 不同", p.display());
            real_count += 1;
        }
    }

    let nanos = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_nanos();
    let scratch: PathBuf = std::env::temp_dir().join(format!("fd2-golden-afm-{}-{nanos}", std::process::id()));
    std::fs::create_dir_all(&scratch).unwrap();

    // Go 端 panic 以 recover 記錄；Rust 端以 catch_unwind 對應，期間關閉預設的 panic 訊息。
    let default_hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(|_| {}));
    let cases = doc["cases"].as_array().unwrap();
    let mut failures = Vec::new();
    let mut kinds = BTreeMap::<String, usize>::new();
    let mut coverage = BTreeMap::<String, usize>::new();
    let mut known = BTreeMap::<String, usize>::new();
    let mut reason_only = BTreeMap::<String, usize>::new();
    for case in cases {
        let index = case["index"].as_u64().unwrap() as usize;
        let kind = case["kind"].as_str().unwrap();
        *kinds.entry(kind.to_string()).or_default() += 1;
        let note = case["note"].as_str().unwrap();
        let got = run_case(&env, &scratch, index, case);
        let listed = known_reason(kind, note);
        let label = format!("case {index} ({kind}: {note:?})");
        // 原版分離素材（含複製後變形）與 ANI.DAT、VM、容器案例一律完全比對。
        let strict = case["call"] == "archive" || kind == "separated-real";
        if strict {
            if let Some(d) = first_diff("result", &normalize(&case["result"]), &normalize(&got)) {
                failures.push(format!("{label}: {d}"));
            }
            if listed.is_some() {
                failures.push(format!("{label}: 嚴格比對的案例不得列入差異清單"));
            }
        } else {
            match (compare_separated(&case["result"], &got), listed) {
                (Err(d), _) => failures.push(format!("{label}: {d}")),
                (Ok(Outcome::AcceptReject(d)), None) => {
                    failures.push(format!("{label}: 未列入清單的接受／拒收差異：{d}"))
                }
                (Ok(Outcome::AcceptReject(_)), Some(reason)) => {
                    *known.entry(format!("{kind}｜{reason}")).or_default() += 1
                }
                (Ok(_), Some(_)) => failures.push(format!("{label}: 列在差異清單但接受與否已一致")),
                (Ok(Outcome::ReasonOnly), None) => *reason_only.entry(kind.to_string()).or_default() += 1,
                (Ok(Outcome::Same), None) => {}
            }
        }
        for (call, key) in [("count", "count"), ("read", "read"), ("decode", "decode"), ("load", "load")] {
            let rows: Vec<&Value> = match &case["result"][key] {
                Value::Array(a) => a.iter().collect(),
                Value::Object(_) => vec![&case["result"][key]],
                _ => continue,
            };
            for row in rows {
                *coverage.entry(format!("{call:6} {}", category(row))).or_default() += 1;
            }
        }
    }
    std::panic::set_hook(default_hook);
    std::fs::remove_dir_all(&scratch).unwrap();

    eprintln!("黃金案例 {}：{kinds:?}；原版檔 {real_count} 個", cases.len());
    for (k, n) in &coverage {
        eprintln!("  {n:5}  {k}");
    }
    eprintln!("刻意不照做的接受／拒收差異（{} 組）：", KNOWN_DIFFERENCES.len());
    for (k, n) in &known {
        eprintln!("  {n:3}  {k}");
    }
    eprintln!("兩邊都拒收、只有原因歸屬不同（函式庫錯誤對 afm 契約錯誤）：{reason_only:?}");
    assert!(failures.is_empty(), "{} 個案例不符：\n{}", failures.len(), failures.join("\n"));
}
