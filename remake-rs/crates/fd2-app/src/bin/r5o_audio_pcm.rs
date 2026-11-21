//! R5 階段 4：Go／Rust 音訊 PCM 逐樣本比對（docs/goal/121；r5o 建立，r5v 改成比對照 Go 移植的解碼鏈）。只輸出資料。
//!
//! 由 `tools/rust_r5o_audio_pcm.sh` 驅動。讀 Go 工具 `remake/cmd/fd2-r5-audio-pcm` 的 `go-pcm.jsonl` 與
//! `pcm/<id>.{final,native}.s16`，以及 `remake/cmd/fd2-r5v-audio-loop` 的 `loop.jsonl`、`lookup1.json`：
//!
//! | 階段 | Rust | 對照的 Go |
//! |---|---|---|
//! | `final` | `fd2_app::audio::decode_pcm`（正式播放路徑） | `DecodeWithSampleRate`＋`ReadAll` |
//! | `native` | `audio::open_go_stream(…, false)`＋`read_all` | `DecodeWithoutResampling`＋`ReadAll` |
//! | `resample_on_go_native` | 診斷：Rust 的 `Resampling` 輸入 **Go** 的 `native` | `final` |
//!
//! BGM 另外核對循環：Rust 串流讀完一圈後 `Seek(0)` 再讀一圈，兩圈的 SHA-256 與 Go `audio.NewInfiniteLoop` 讀兩圈
//! 的結果比對。`lookup1.json` 核對 `go_vorbis::lookup1_values` 與 Go `lookup1Values` 公式在本機的值。
//!
//! 用法：`r5o_audio_pcm --assets <remake/assets> --sfx <pack/sfx> --go <Go 輸出目錄> --go-loop <Go 循環輸出目錄>
//! --out <輸出根>`；寫出 `<out>/rust/rust-pcm.jsonl`、`<out>/rust/pcm/<id>.final.s16`、`<out>/compare/compare.jsonl`、
//! `summary.json`、`summary.txt`。全部 `final` 與循環都相同、`lookup1` 全部相符時 `summary.json` 的 `verdict` 是
//! `identical`，否則是 `different`（驅動腳本據此失敗）。

use std::collections::BTreeMap;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::Arc;

use fd2_app::audio::{self, Format, OUTPUT_RATE};
use fd2_app::go_ebiten_audio::convert::Resampling;
use fd2_app::go_io::{self, BytesReader, Seek, Whence};
use fd2_app::{go_ebiten_audio, go_oggvorbis, go_vorbis};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

fn sha256_hex(bytes: &[u8]) -> String {
    Sha256::digest(bytes).iter().map(|b| format!("{b:02x}")).collect()
}

fn to_bytes(pcm: &[i16]) -> Vec<u8> {
    pcm.iter().flat_map(|s| s.to_le_bytes()).collect()
}

fn read_s16(path: &Path) -> Vec<i16> {
    let raw = std::fs::read(path).unwrap_or_else(|e| panic!("讀 {path:?}: {e}"));
    audio::bytes_to_i16(&raw)
}

/// 一個階段的比對結果。
#[derive(Default, Clone)]
struct Cmp {
    go_frames: usize,
    rust_frames: usize,
    identical: bool,
    compared: usize,
    mismatched: usize,
    off_by_one: usize,
    max_abs: i32,
    sum_sq: f64,
    first_mismatch: Option<usize>,
}

impl Cmp {
    fn new(go: &[i16], rust: &[i16]) -> Cmp {
        let n = go.len().min(rust.len());
        let mut c = Cmp {
            go_frames: go.len() / 2,
            rust_frames: rust.len() / 2,
            identical: go == rust,
            compared: n,
            ..Default::default()
        };
        for i in 0..n {
            let d = (i32::from(go[i]) - i32::from(rust[i])).abs();
            if d != 0 {
                c.mismatched += 1;
                c.off_by_one += usize::from(d == 1);
                c.max_abs = c.max_abs.max(d);
                c.sum_sq += f64::from(d) * f64::from(d);
                c.first_mismatch.get_or_insert(i);
            }
        }
        c
    }

    fn json(&self) -> Value {
        json!({
            "identical": self.identical,
            "go_frames": self.go_frames, "rust_frames": self.rust_frames,
            "frame_diff": self.rust_frames as i64 - self.go_frames as i64,
            "compared_samples": self.compared, "mismatched_samples": self.mismatched,
            "off_by_one_samples": self.off_by_one, "max_abs_diff": self.max_abs,
            "rms_diff": if self.compared == 0 { 0.0 } else { (self.sum_sq / self.compared as f64).sqrt() },
            "first_mismatch_sample": self.first_mismatch,
        })
    }
}

/// 各類別、各階段的彙總。
#[derive(Default)]
struct Agg {
    files: usize,
    identical: usize,
    same_length: usize,
    compared: usize,
    mismatched: usize,
    off_by_one: usize,
    max_abs: i32,
    sum_sq: f64,
    frame_diff_min: i64,
    frame_diff_max: i64,
}

impl Agg {
    fn add(&mut self, c: &Cmp) {
        let fd = c.rust_frames as i64 - c.go_frames as i64;
        if self.files == 0 {
            (self.frame_diff_min, self.frame_diff_max) = (fd, fd);
        }
        self.files += 1;
        self.identical += usize::from(c.identical);
        self.same_length += usize::from(fd == 0);
        self.compared += c.compared;
        self.mismatched += c.mismatched;
        self.off_by_one += c.off_by_one;
        self.max_abs = self.max_abs.max(c.max_abs);
        self.sum_sq += c.sum_sq;
        self.frame_diff_min = self.frame_diff_min.min(fd);
        self.frame_diff_max = self.frame_diff_max.max(fd);
    }

    fn json(&self) -> Value {
        json!({
            "files": self.files, "identical_files": self.identical, "same_length_files": self.same_length,
            "compared_samples": self.compared, "mismatched_samples": self.mismatched,
            "off_by_one_samples": self.off_by_one, "max_abs_diff": self.max_abs,
            "rms_diff": if self.compared == 0 { 0.0 } else { (self.sum_sq / self.compared as f64).sqrt() },
            "frame_diff_min": self.frame_diff_min, "frame_diff_max": self.frame_diff_max,
        })
    }
}

const STAGES: [&str; 3] = ["final", "native", "resample_on_go_native"];

/// WAV 標頭的聲道數與取樣率（只供記錄）。
fn wav_format(bytes: &[u8]) -> (u16, u32) {
    let mut pos = 12;
    while pos + 8 <= bytes.len() {
        let size = u32::from_le_bytes(bytes[pos + 4..pos + 8].try_into().unwrap()) as usize;
        if &bytes[pos..pos + 4] == b"fmt " {
            let b = &bytes[pos + 8..];
            return (u16::from_le_bytes([b[2], b[3]]), u32::from_le_bytes([b[4], b[5], b[6], b[7]]));
        }
        pos += 8 + size;
    }
    (0, 0)
}

/// 讀完一圈、`Seek(0)`、再讀一圈（Go `InfiniteLoop` 每圈結尾的動作）。
fn two_passes(bytes: &[u8]) -> (i64, String, String) {
    let mut s = audio::open_go_stream(Arc::from(bytes), true).expect("open");
    let (p1, e1) = go_io::read_all(&mut s);
    assert!(e1.is_none(), "{e1:?}");
    s.seek(0, Whence::Start).expect("seek");
    let (p2, e2) = go_io::read_all(&mut s);
    assert!(e2.is_none(), "{e2:?}");
    (s.length(), sha256_hex(&p1), sha256_hex(&p2))
}

/// `lookup1.json` 與 `go_vorbis::lookup1_values` 比對：回傳（表雜湊是否相同、整數次方的不同筆數、整數次方筆數）。
fn check_lookup1(path: &Path) -> Value {
    let doc: Value = serde_json::from_slice(&std::fs::read(path).expect("lookup1.json")).unwrap();
    let mut h = Sha256::new();
    for dim in 1..=doc["table_dims"].as_u64().unwrap() as u32 {
        for entries in 1..=doc["table_entries"].as_i64().unwrap() {
            h.update(go_vorbis::lookup1_values(entries, dim).to_le_bytes());
        }
    }
    let table: String = h.finalize().iter().map(|b| format!("{b:02x}")).collect();
    let table_match = table == doc["table_sha256"].as_str().unwrap();
    let powers = doc["powers"].as_array().unwrap();
    let mismatches: Vec<&Value> = powers
        .iter()
        .filter(|c| {
            go_vorbis::lookup1_values(c["entries"].as_i64().unwrap(), c["dim"].as_u64().unwrap() as u32)
                != c["value"].as_i64().unwrap()
        })
        .collect();
    json!({"table_sha256_match": table_match, "rust_table_sha256": table, "powers": powers.len(),
           "power_mismatches": mismatches})
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let arg = |key: &str| -> PathBuf {
        let i = args.iter().position(|a| a == key).unwrap_or_else(|| panic!("缺少 {key}"));
        PathBuf::from(&args[i + 1])
    };
    let (assets, sfx, go, go_loop, out) = (arg("--assets"), arg("--sfx"), arg("--go"), arg("--go-loop"), arg("--out"));
    std::fs::create_dir_all(out.join("rust/pcm")).unwrap();
    std::fs::create_dir_all(out.join("compare")).unwrap();
    let go_lines = std::fs::read_to_string(go.join("go-pcm.jsonl")).expect("go-pcm.jsonl");
    let go_loops: BTreeMap<String, Value> = std::fs::read_to_string(go_loop.join("loop.jsonl"))
        .expect("loop.jsonl")
        .lines()
        .map(|l| {
            let v: Value = serde_json::from_str(l).unwrap();
            (v["id"].as_str().unwrap().to_string(), v)
        })
        .collect();
    let mut rust_log = std::fs::File::create(out.join("rust/rust-pcm.jsonl")).unwrap();
    let mut cmp_log = std::fs::File::create(out.join("compare/compare.jsonl")).unwrap();
    let mut aggs: BTreeMap<(String, &str), Agg> = BTreeMap::new();
    let mut errors = 0usize;
    let (mut loops_checked, mut loops_identical) = (0usize, 0usize);
    let mut floor_types: BTreeMap<u16, usize> = BTreeMap::new();

    for line in go_lines.lines() {
        let g: Value = serde_json::from_str(line).unwrap();
        let (id, kind, rel) = (g["id"].as_str().unwrap(), g["kind"].as_str().unwrap(), g["path"].as_str().unwrap());
        let path = match kind {
            "sfx" => sfx.join(rel.strip_prefix("sfx/").unwrap()),
            _ => assets.join(rel.strip_prefix("assets/").unwrap()),
        };
        let bytes = std::fs::read(&path).unwrap_or_else(|e| panic!("讀 {path:?}: {e}"));
        assert_eq!(sha256_hex(&bytes), g["source_sha256"].as_str().unwrap(), "{id} 來源雜湊");
        if g["final"].get("error").is_some() || g["native"].get("error").is_some() {
            errors += 1;
            writeln!(cmp_log, "{}", json!({"id": id, "go_error": [g["final"]["error"], g["native"]["error"]]}))
                .unwrap();
            continue;
        }
        let stem = id.replace('/', "__");
        let go_final = read_s16(&go.join(format!("pcm/{stem}.final.s16")));
        let go_native_bytes = std::fs::read(go.join(format!("pcm/{stem}.native.s16"))).unwrap();
        let go_native = audio::bytes_to_i16(&go_native_bytes);
        assert_eq!(sha256_hex(&to_bytes(&go_final)), g["final"]["sha256"].as_str().unwrap(), "{id} Go final");

        let rust_final = match audio::decode_pcm(&bytes) {
            Ok(pcm) => pcm,
            Err(e) => {
                errors += 1;
                writeln!(cmp_log, "{}", json!({"id": id, "rust_error": e.to_string()})).unwrap();
                continue;
            }
        };
        let mut native_stream = audio::open_go_stream(Arc::from(bytes.as_slice()), false).expect("native");
        let native_size = native_stream.length();
        let (native_bytes, native_err) = go_io::read_all(&mut native_stream);
        assert!(native_err.is_none(), "{id} native: {native_err:?}");
        let rust_native = audio::bytes_to_i16(&native_bytes);

        let (channels, rate, granule, floors) = match audio::sniff(&bytes).unwrap() {
            Format::Ogg => {
                let r = go_oggvorbis::Reader::new(Arc::from(bytes.as_slice()), true).expect("oggvorbis");
                let floors = go_ebiten_audio::vorbis::floor_types(Arc::from(bytes.as_slice())).expect("floor types");
                for &t in &floors {
                    *floor_types.entry(t).or_default() += 1;
                }
                (r.channels() as u16, r.sample_rate() as u32, Some(r.length()), Some(floors))
            }
            Format::Wav => {
                let (c, r) = wav_format(&bytes);
                (c, r, None, None)
            }
        };
        let resampled_go_native = if rate == OUTPUT_RATE {
            go_native.clone()
        } else {
            let mut r = Resampling::new(
                Box::new(BytesReader::new(Arc::from(go_native_bytes.as_slice()))),
                native_size,
                i64::from(rate),
                i64::from(OUTPUT_RATE),
            );
            let (b, err) = go_io::read_all(&mut r);
            assert!(err.is_none(), "{id} resample: {err:?}");
            audio::bytes_to_i16(&b)
        };
        let results = [
            Cmp::new(&go_final, &rust_final),
            Cmp::new(&go_native, &rust_native),
            Cmp::new(&go_final, &resampled_go_native),
        ];
        let final_bytes = to_bytes(&rust_final);
        let final_sha = sha256_hex(&final_bytes);
        let loop_rec = (kind == "bgm").then(|| {
            let (length, p1, p2) = two_passes(&bytes);
            let gl = go_loops.get(id).unwrap_or_else(|| panic!("{id} 不在 loop.jsonl"));
            let same = gl["length"].as_i64() == Some(length)
                && gl["pass1_sha256"].as_str() == Some(p1.as_str())
                && gl["pass2_sha256"].as_str() == Some(p2.as_str())
                && p1 == final_sha
                && p1 == p2;
            loops_checked += 1;
            loops_identical += usize::from(same);
            json!({"identical": same, "length": length, "rust_pass1_sha256": p1, "rust_pass2_sha256": p2,
                   "go_pass1_sha256": gl["pass1_sha256"], "go_pass2_sha256": gl["pass2_sha256"],
                   "go_length": gl["length"]})
        });
        std::fs::write(out.join(format!("rust/pcm/{stem}.final.s16")), &final_bytes).unwrap();
        writeln!(
            rust_log,
            "{}",
            json!({
                "id": id, "kind": kind, "path": rel, "source_channels": channels, "source_rate": rate,
                "last_granule": granule, "floor_types": floors,
                "final": { "bytes": final_bytes.len(), "frames": rust_final.len() / 2, "channels": 2,
                           "sample_rate": OUTPUT_RATE, "sha256": final_sha },
                "native": { "bytes": native_bytes.len(), "declared_bytes": native_size,
                            "sha256": sha256_hex(&native_bytes) },
            })
        )
        .unwrap();
        let mut rec = json!({"id": id, "kind": kind, "source_channels": channels, "source_rate": rate,
                             "last_granule": granule, "loop": loop_rec});
        for (stage, c) in STAGES.iter().zip(&results) {
            rec[*stage] = c.json();
            aggs.entry((kind.to_string(), stage)).or_default().add(c);
        }
        writeln!(cmp_log, "{rec}").unwrap();
    }

    let lookup1 = check_lookup1(&go_loop.join("lookup1.json"));
    let final_all = aggs.iter().filter(|((_, s), _)| *s == "final").all(|(_, a)| a.identical == a.files);
    let final_files: usize = aggs.iter().filter(|((_, s), _)| *s == "final").map(|(_, a)| a.files).sum();
    let identical = errors == 0
        && final_all
        && final_files > 0
        && loops_checked == go_loops.len()
        && loops_identical == loops_checked
        && lookup1["table_sha256_match"] == true
        && lookup1["power_mismatches"].as_array().is_some_and(Vec::is_empty);
    let mut summary = json!({
        "schema": "fd2-r5v-audio-pcm-compare/1", "errors": errors, "groups": {},
        "loops": {"checked": loops_checked, "identical": loops_identical, "go_records": go_loops.len()},
        "floor_types": floor_types.iter().map(|(t, n)| (t.to_string(), json!(n))).collect::<serde_json::Map<_, _>>(),
        "lookup1": lookup1,
        "verdict": if identical { "identical" } else { "different" },
    });
    let mut text = String::from(
        "類別\t階段\t檔數\t整檔相同\t長度相同\t幀數差(最小..最大)\t比較樣本\t不同樣本\t差1\t最大差\tRMS\n",
    );
    for ((kind, stage), a) in &aggs {
        summary["groups"][format!("{kind}/{stage}")] = a.json();
        let j = a.json();
        text += &format!(
            "{kind}\t{stage}\t{}\t{}\t{}\t{}..{}\t{}\t{}\t{}\t{}\t{:.4}\n",
            a.files,
            a.identical,
            a.same_length,
            a.frame_diff_min,
            a.frame_diff_max,
            a.compared,
            a.mismatched,
            a.off_by_one,
            a.max_abs,
            j["rms_diff"].as_f64().unwrap()
        );
    }
    text += &format!("BGM 循環兩圈：{loops_identical}／{loops_checked} 與 Go 相同（Go 紀錄 {} 筆）\n", go_loops.len());
    text += &format!(
        "lookup1：表 {}、整數次方 {} 筆中不同 {} 筆\n",
        if summary["lookup1"]["table_sha256_match"] == true { "相同" } else { "不同" },
        summary["lookup1"]["powers"],
        summary["lookup1"]["power_mismatches"].as_array().map_or(0, Vec::len)
    );
    text += &format!("floor 型別：{}\n", summary["floor_types"]);
    text += &format!("錯誤 {errors}\n結論 {}\n", summary["verdict"].as_str().unwrap());
    std::fs::write(out.join("compare/summary.json"), serde_json::to_vec_pretty(&summary).unwrap()).unwrap();
    std::fs::write(out.join("compare/summary.txt"), &text).unwrap();
    print!("{text}");
}
