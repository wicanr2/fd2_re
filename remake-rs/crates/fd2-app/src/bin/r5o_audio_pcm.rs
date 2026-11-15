//! R5 階段 4 r5o：Go／Rust 音訊 PCM 逐樣本比對（docs/goal/121）。只輸出資料。
//!
//! 由 `tools/rust_r5o_audio_pcm.sh` 驅動。讀 Go 工具 `remake/cmd/fd2-r5-audio-pcm` 的 `go-pcm.jsonl` 與
//! `pcm/<id>.{final,native}.s16`，以同一批來源檔算出四種 Rust 結果，逐檔與 Go 比：
//!
//! | 階段 | Rust | 對照的 Go |
//! |---|---|---|
//! | `final` | `fd2_app::audio::decode_pcm`（正式播放路徑：lewton＋rodio 重取樣） | `DecodeWithSampleRate`＋`ReadAll` |
//! | `native` | lewton 浮點 → `quantize` → 單聲道複製（不重取樣；WAV 是整數樣本） | `DecodeWithoutResampling`＋`ReadAll` |
//! | `ebiten_on_go_native` | 診斷：Ebiten `convert.Resampling` 的移植，輸入是 **Go** 的 `native` | `final` |
//! | `ebiten_on_rust_native` | 診斷：同一個移植，輸入是 **Rust** 的 `native` | `final` |
//!
//! 後兩者只用來分辨差異來自解碼器還是重取樣，沒有接進播放路徑；要不要照 Go 重做由使用者決定。
//! `final_head`、`ebiten_on_rust_native_head` 是同兩個比較只取前段：輸出幀的重取樣窗（±8 個來源樣本）完全落在
//! Go 與 Rust 都有解出樣本的範圍內，排除結尾長度不同造成的差異。
//!
//! OGG 另記 `start_offset`（第一個帶 granule 的頁：granule 減去已解出的樣本數）與 `spec_frames`
//! （最後 granule 減 `start_offset`，Vorbis I 規格要求的解碼長度），用來判斷兩邊結尾處理哪一邊照規格。
//!
//! 用法：`r5o_audio_pcm --assets <remake/assets> --sfx <pack/sfx> --go <Go 輸出目錄> --out <輸出根>`；
//! 寫出 `<out>/rust/rust-pcm.jsonl`、`<out>/rust/pcm/<id>.final.s16`、`<out>/compare/compare.jsonl`、
//! `summary.json`、`summary.txt`。

use std::collections::BTreeMap;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::Arc;

use fd2_app::audio::{self, Format, OUTPUT_RATE};
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
    raw.as_chunks::<2>().0.iter().map(|c| i16::from_le_bytes(*c)).collect()
}

/// OGG 最後一頁的 granule position（oggvorbis `LastPosition`，Go 的 `Length()`）。
fn ogg_last_granule(bytes: &[u8]) -> Option<u64> {
    let (mut pos, mut last) = (0usize, None);
    while pos + 27 <= bytes.len() && &bytes[pos..pos + 4] == b"OggS" {
        let granule = u64::from_le_bytes(bytes[pos + 6..pos + 14].try_into().unwrap());
        let segments = bytes[pos + 26] as usize;
        let table = bytes.get(pos + 27..pos + 27 + segments)?;
        let body: usize = table.iter().map(|&s| s as usize).sum();
        if granule != u64::MAX {
            last = Some(granule);
        }
        pos += 27 + segments + body;
    }
    last
}

/// 原始取樣率下的 Rust 樣本（轉 16 位元、單聲道複製成雙聲道），與 Go `DecodeWithoutResampling` 對照。
fn rust_native(bytes: &[u8]) -> (u16, u32, Vec<i16>) {
    let native = audio::open_native(Arc::from(bytes)).expect("Rust 解碼");
    let (channels, rate) = (native.channels, native.sample_rate);
    let samples: Vec<i16> = match native.integer {
        Some(integer) => integer,
        None => native.samples.map(audio::quantize).collect(),
    };
    let stereo = if channels == 1 { samples.iter().flat_map(|&s| [s, s]).collect() } else { samples };
    (channels, rate, stereo)
}

/// Ebiten v2.6.6 `audio/internal/convert/resampling.go` 的移植（診斷用）。`size` 是 Go 傳給 `NewResampling` 的
/// 位元組數（OGG：最後 granule × 4；WAV：`data` 宣告長度換算成雙聲道 16 位元）；超出實際樣本的位置讀到 0，
/// 與 Go 讀到 EOF 後區塊補零相同。cos 表以 Rust `f64::cos` 建（Go `math.Cos`）。
struct EbitenResampling<'a> {
    source: &'a [i16],
    size: i64,
    from: i64,
    to: i64,
    cos: Vec<f64>,
}

impl EbitenResampling<'_> {
    fn fast_cos01(&self, x: f64) -> f64 {
        let x = x.abs();
        let n = self.cos.len() as i64;
        let mut i = (4.0 * n as f64 * x) as i64;
        if 4 * n < i {
            i %= 4 * n;
        }
        let mut sign = 1.0;
        if i < n {
        } else if i < n * 2 {
            i = n * 2 - i;
            sign = -1.0;
        } else if i < n * 3 {
            i -= n * 2;
            sign = -1.0;
        } else {
            i = n * 4 - i;
        }
        if i == n {
            return 0.0;
        }
        sign * self.cos[i as usize]
    }

    fn sinc01(&self, x: f64) -> f64 {
        if x.abs() < 1e-8 {
            return 1.0;
        }
        self.fast_cos01(x - 0.25) / (x * 2.0 * std::f64::consts::PI)
    }

    fn src(&self, i: i64) -> (f64, f64) {
        if i < 0 || self.size / 4 <= i {
            return (0.0, 0.0);
        }
        let i = i as usize;
        let at = |k: usize| self.source.get(k).map_or(0.0, |&s| f64::from(s) / 32767.0);
        (at(2 * i), at(2 * i + 1))
    }

    fn length(&self) -> i64 {
        let s = (self.size as f64 * self.to as f64 / self.from as f64) as i64;
        s / 4 * 4
    }

    fn at(&self, t: i64) -> (f64, f64) {
        let window = 8.0;
        let t_in_src = t as f64 * self.from as f64 / self.to as f64;
        let mut start = (t_in_src - window) as i64;
        if start < 0 {
            start = 0;
        }
        if self.size / 4 <= start {
            start = self.size / 4 - 1;
        }
        let mut end = (t_in_src + window) as i64;
        if self.size / 4 <= end {
            end = self.size / 4 - 1;
        }
        let (mut lv, mut rv) = (0.0, 0.0);
        for n in start..=end {
            let (l, r) = self.src(n);
            let d = t_in_src - n as f64;
            let w = 0.5 + 0.5 * self.fast_cos01(d / (window * 2.0 + 1.0));
            let s = self.sinc01(d / 2.0) * w;
            lv += l * s;
            rv += r * s;
        }
        (lv.clamp(-1.0, 1.0), rv.clamp(-1.0, 1.0))
    }

    fn run(source: &[i16], size: i64, from: u32, to: u32) -> Vec<i16> {
        let cos = (0..65536).map(|i| (i as f64 * std::f64::consts::PI / 2.0 / 65536.0).cos()).collect();
        let r = EbitenResampling { source, size, from: i64::from(from), to: i64::from(to), cos };
        let frames = r.length() / 4;
        let mut out = Vec::with_capacity(frames as usize * 2);
        for t in 0..frames {
            let (l, rr) = r.at(t);
            out.push((l * 32767.0) as i16);
            out.push((rr * 32767.0) as i16);
        }
        out
    }
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

const STAGES: [&str; 6] =
    ["final", "native", "ebiten_on_go_native", "ebiten_on_rust_native", "final_head", "ebiten_on_rust_native_head"];

/// 最後三頁的 granule、旗標（4 = 串流結束）與片段數，以及全檔頁數。
fn ogg_tail_pages(bytes: &[u8]) -> Value {
    let (mut pos, mut pages) = (0usize, Vec::new());
    while pos + 27 <= bytes.len() && &bytes[pos..pos + 4] == b"OggS" {
        let granule = u64::from_le_bytes(bytes[pos + 6..pos + 14].try_into().unwrap());
        let segments = bytes[pos + 26] as usize;
        let Some(table) = bytes.get(pos + 27..pos + 27 + segments) else { break };
        let body: usize = table.iter().map(|&s| s as usize).sum();
        pages.push(json!({"granule": granule as i64, "flags": bytes[pos + 5], "segments": segments, "body": body}));
        pos += 27 + segments + body;
    }
    let n = pages.len();
    json!({"pages": n, "trailing_bytes": bytes.len() - pos, "tail": pages[n.saturating_sub(3)..]})
}

/// lewton 逐封包解到底：封包數、幀數，以及停下的原因（`end` 或錯誤訊息）；只用來記錄結尾處理。
fn lewton_end(bytes: &[u8]) -> Value {
    let Ok(mut reader) = lewton::inside_ogg::OggStreamReader::new(std::io::Cursor::new(bytes)) else {
        return json!({"error": "headers"});
    };
    let channels = usize::from(reader.ident_hdr.audio_channels);
    let (mut packets, mut frames, mut last_packet_frames) = (0usize, 0usize, 0usize);
    // 每頁結束時「已解出幀數 − 該頁 granule」：0 表示兩者一致；記第一次與最後一次不為 0 的頁。
    let (mut last_gp, mut first_drift, mut last_drift, mut page) = (None, None, None, 0usize);
    let reason = loop {
        match reader.read_dec_packet_generic::<lewton::samples::InterleavedSamples<f32>>() {
            Ok(Some(p)) => {
                packets += 1;
                last_packet_frames = p.samples.len() / channels;
                frames += last_packet_frames;
                let gp = reader.get_last_absgp();
                if gp != last_gp {
                    page += 1;
                    let drift = frames as i64 - gp.unwrap_or(0) as i64;
                    if drift != 0 {
                        first_drift.get_or_insert((page, drift));
                        last_drift = Some((page, drift));
                    }
                    last_gp = gp;
                }
            }
            Ok(None) => break String::from("end"),
            Err(e) => break format!("{e:?}"),
        }
    };
    json!({"packets": packets, "frames": frames, "last_packet_frames": last_packet_frames, "stop": reason,
           "last_absgp": reader.get_last_absgp(), "pages_with_granule": page,
           "first_drift": first_drift, "last_drift": last_drift})
}

/// 第一個帶 granule 的頁之前 lewton 解出的幀數與該 granule 的差（正值：串流從非零時間開始）。
fn ogg_start_offset(bytes: &[u8]) -> Option<i64> {
    let mut reader = lewton::inside_ogg::OggStreamReader::new(std::io::Cursor::new(bytes)).ok()?;
    let channels = usize::from(reader.ident_hdr.audio_channels);
    let mut decoded = 0i64;
    while let Ok(Some(p)) = reader.read_dec_packet_generic::<lewton::samples::InterleavedSamples<f32>>() {
        decoded += (p.samples.len() / channels) as i64;
        if let Some(g) = reader.get_last_absgp() {
            return Some(g as i64 - decoded);
        }
    }
    None
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let arg = |key: &str| -> PathBuf {
        let i = args.iter().position(|a| a == key).unwrap_or_else(|| panic!("缺少 {key}"));
        PathBuf::from(&args[i + 1])
    };
    let (assets, sfx, go, out) = (arg("--assets"), arg("--sfx"), arg("--go"), arg("--out"));
    std::fs::create_dir_all(out.join("rust/pcm")).unwrap();
    std::fs::create_dir_all(out.join("compare")).unwrap();
    let go_lines = std::fs::read_to_string(go.join("go-pcm.jsonl")).expect("go-pcm.jsonl");
    let mut rust_log = std::fs::File::create(out.join("rust/rust-pcm.jsonl")).unwrap();
    let mut cmp_log = std::fs::File::create(out.join("compare/compare.jsonl")).unwrap();
    let mut aggs: BTreeMap<(String, &str), Agg> = BTreeMap::new();
    let mut errors = 0usize;

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
        let go_native = read_s16(&go.join(format!("pcm/{stem}.native.s16")));
        assert_eq!(sha256_hex(&to_bytes(&go_final)), g["final"]["sha256"].as_str().unwrap(), "{id} Go final");

        let rust_final = match audio::decode_pcm(&bytes) {
            Ok(pcm) => pcm,
            Err(e) => {
                errors += 1;
                writeln!(cmp_log, "{}", json!({"id": id, "rust_error": e.to_string()})).unwrap();
                continue;
            }
        };
        let (channels, rate, native) = rust_native(&bytes);
        // Go 交給 NewResampling 的 size（位元組，雙聲道 16 位元）。
        let (size, granule) = match audio::sniff(&bytes).unwrap() {
            Format::Ogg => {
                let granule = ogg_last_granule(&bytes).expect("granule");
                (granule as i64 * 4, Some(granule))
            }
            Format::Wav => {
                let wav = audio::parse_wav(&bytes).unwrap();
                // 宣告的 data 長度：parse_wav 只回傳實際讀到的樣本，這裡重讀標頭的宣告值。
                let declared = wav_declared_data(&bytes);
                let per = if wav.channels == 1 { 2 } else { 1 } * if wav_bits(&bytes) == 8 { 2 } else { 1 };
                (declared as i64 * per, None)
            }
        };
        let resample = |src: &[i16]| {
            if rate == OUTPUT_RATE { src.to_vec() } else { EbitenResampling::run(src, size, rate, OUTPUT_RATE) }
        };
        let port_rust = resample(&native);
        // 前段：輸出幀 t 的窗 [t·from/to − 8, t·from/to + 8] 都在兩邊實際解出的樣本內。
        let both = (go_native.len().min(native.len()) / 2) as f64;
        let head_frames = (((both - 9.0) * f64::from(OUTPUT_RATE) / f64::from(rate)).max(0.0)) as usize;
        let head = |v: &[i16]| v[..v.len().min(head_frames * 2)].to_vec();
        let results = [
            Cmp::new(&go_final, &rust_final),
            Cmp::new(&go_native, &native),
            Cmp::new(&go_final, &resample(&go_native)),
            Cmp::new(&go_final, &port_rust),
            Cmp::new(&head(&go_final), &head(&rust_final)),
            Cmp::new(&head(&go_final), &head(&port_rust)),
        ];
        let start_offset = granule.and_then(|_| ogg_start_offset(&bytes));
        let spec_frames = granule.zip(start_offset).map(|(g, o)| g as i64 - o);
        let final_bytes = to_bytes(&rust_final);
        std::fs::write(out.join(format!("rust/pcm/{stem}.final.s16")), &final_bytes).unwrap();
        writeln!(
            rust_log,
            "{}",
            json!({
                "id": id, "kind": kind, "path": rel, "source_channels": channels, "source_rate": rate,
                "last_granule": granule, "rust_native_frames": native.len() / 2,
                "final": { "bytes": final_bytes.len(), "frames": rust_final.len() / 2, "channels": 2,
                           "sample_rate": OUTPUT_RATE, "sha256": sha256_hex(&final_bytes) },
            })
        )
        .unwrap();
        let mut rec = json!({"id": id, "kind": kind, "source_channels": channels, "source_rate": rate,
                             "last_granule": granule, "start_offset": start_offset, "spec_frames": spec_frames,
                             "go_native_frames": go_native.len() / 2, "rust_native_frames": native.len() / 2,
                             "go_resampling_size": size, "head_frames": head_frames,
                             "lewton_end": granule.map(|_| lewton_end(&bytes)),
                             "ogg_tail": granule.map(|_| ogg_tail_pages(&bytes))});
        for (stage, c) in STAGES.iter().zip(&results) {
            rec[*stage] = c.json();
            aggs.entry((kind.to_string(), stage)).or_default().add(c);
        }
        writeln!(cmp_log, "{rec}").unwrap();
    }

    let mut summary = json!({"schema": "fd2-r5o-audio-pcm-compare/1", "errors": errors, "groups": {}});
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
    text += &format!("錯誤 {errors}\n");
    std::fs::write(out.join("compare/summary.json"), serde_json::to_vec_pretty(&summary).unwrap()).unwrap();
    std::fs::write(out.join("compare/summary.txt"), &text).unwrap();
    print!("{text}");
}

fn wav_chunk<'a>(bytes: &'a [u8], name: &[u8; 4]) -> Option<(usize, &'a [u8])> {
    let mut pos = 12;
    while pos + 8 <= bytes.len() {
        let size = u32::from_le_bytes(bytes[pos + 4..pos + 8].try_into().unwrap()) as usize;
        if &bytes[pos..pos + 4] == name {
            return Some((size, &bytes[pos + 8..]));
        }
        pos += 8 + size;
    }
    None
}

fn wav_declared_data(bytes: &[u8]) -> usize {
    wav_chunk(bytes, b"data").map_or(0, |(size, _)| size)
}

fn wav_bits(bytes: &[u8]) -> u16 {
    wav_chunk(bytes, b"fmt ").map_or(0, |(_, b)| u16::from_le_bytes([b[14], b[15]]))
}
