//! 對應凍結 Go 版 `remake/internal/musiccatalog`（單檔 `catalog.go`）：驗證分離的 FM／MT-32 OGG 音樂包。
//!
//! Go 套件註解：Package musiccatalog validates the separated FM／MT-32 OGG bundle.
//!
//! - JSON 用 serde，`DisallowUnknownFields` → `deny_unknown_fields`；Go 讀 `null` 或缺欄位時留零值，
//!   Rust 以 `default` 加 `null_as_default` 得到相同結果。Go 解碼與讀檔錯誤的內層文字屬標準函式庫，
//!   Rust 只保留本套件的前綴（`music catalog decode: `、`music catalog: ` 等）。
//! - Go 的 `byID`／`root` 是未匯出欄位，JSON 不讀也不寫；Rust 以 `#[serde(skip)]` 保留同樣的形狀。
//! - Go `map` → `BTreeMap`；`Load` 只以固定的 profile 名稱查 map，不走訪，順序不影響結果。
//! - 錯誤用 `fd2_rules::battle::BattleError`（`Display` 與 Go `err.Error()` 逐字相同）。

use crate::go_filepath;
use crate::localization::{StrictDecodeError, decode_strict, go_quote};
use fd2_rules::battle::null_as_default;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;

pub use fd2_rules::battle::BattleError as Error;

/// `musiccatalog` 的結果型別。
pub type Result<T, E = Error> = std::result::Result<T, E>;

fn err(message: impl Into<String>) -> Error {
    Error::new(message)
}

const EXPECTED_TRACKS: [i64; 15] = [1, 3, 4, 6, 8, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19];

/// Go：`catalog.go:18`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Source {
    #[serde(deserialize_with = "null_as_default")]
    pub file: String,
    #[serde(deserialize_with = "null_as_default")]
    pub size: i64,
    #[serde(deserialize_with = "null_as_default")]
    pub md5: String,
    #[serde(deserialize_with = "null_as_default")]
    pub sha256: String,
}

/// Go：`catalog.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Loop {
    #[serde(deserialize_with = "null_as_default")]
    pub mode: String,
    #[serde(deserialize_with = "null_as_default")]
    pub accepted_counts: Vec<i64>,
    #[serde(deserialize_with = "null_as_default")]
    pub seam_evidence: String,
    #[serde(deserialize_with = "null_as_default")]
    pub evidence_level: String,
}

/// Go：`catalog.go:32`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Profile {
    #[serde(deserialize_with = "null_as_default")]
    pub render_pipeline: String,
    #[serde(deserialize_with = "null_as_default")]
    pub provenance_status: String,
    #[serde(deserialize_with = "null_as_default")]
    pub rights_note: String,
}

/// Go：`catalog.go:38`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Render {
    #[serde(deserialize_with = "null_as_default")]
    pub path: String,
    #[serde(deserialize_with = "null_as_default")]
    pub bytes: i64,
    #[serde(deserialize_with = "null_as_default")]
    pub sha256: String,
    #[serde(deserialize_with = "null_as_default")]
    pub codec: String,
    #[serde(deserialize_with = "null_as_default")]
    pub channels: i64,
    #[serde(deserialize_with = "null_as_default")]
    pub sample_rate: i64,
    #[serde(deserialize_with = "null_as_default")]
    pub pcm_samples: u64,
    #[serde(deserialize_with = "null_as_default")]
    pub duration_ms: u64,
}

/// Go：`catalog.go:49`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Track {
    #[serde(deserialize_with = "null_as_default")]
    pub track_id: String,
    #[serde(deserialize_with = "null_as_default")]
    pub resource_index: i64,
    #[serde(deserialize_with = "null_as_default")]
    pub renders: BTreeMap<String, Render>,
}

/// Go：`catalog.go:55`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Catalog {
    #[serde(deserialize_with = "null_as_default")]
    pub schema: String,
    #[serde(deserialize_with = "null_as_default")]
    pub schema_version: i64,
    #[serde(deserialize_with = "null_as_default")]
    pub source: Source,
    #[serde(rename = "loop", deserialize_with = "null_as_default")]
    pub loop_: Loop,
    #[serde(deserialize_with = "null_as_default")]
    pub profiles: BTreeMap<String, Profile>,
    #[serde(deserialize_with = "null_as_default")]
    pub tracks: Vec<Track>,
    #[serde(skip)]
    by_id: BTreeMap<String, Track>,
    #[serde(skip)]
    root: String,
}

/// Go：`catalog.go:66`。讀檔錯誤的內層文字是 Rust `std::io` 的（見模組文件）。
pub fn load(assets_root: &str) -> Result<Catalog> {
    let raw = std::fs::read(go_filepath::join(&[assets_root, "music_catalog.json"]))
        .map_err(|e| err(format!("music catalog: {e}")))?;
    let mut catalog: Catalog = decode_strict(&raw).map_err(|e| match e {
        StrictDecodeError::Decode(detail) => err(format!("music catalog decode: {detail}")),
        StrictDecodeError::Trailing | StrictDecodeError::TrailingInvalid(_) => err("music catalog trailing data"),
    })?;
    if catalog.schema != "fd2_music_catalog" || catalog.schema_version != 1 {
        return Err(err(format!(
            "music catalog schema {} version {}",
            go_quote(&catalog.schema),
            catalog.schema_version
        )));
    }
    let want_source = Source {
        file: "FDMUS.DAT".to_string(),
        size: 80367,
        md5: "4dfa214125edcc4658acbba2e1201a28".to_string(),
        sha256: "4105ebde543fe1c497e852728f6bc333bda80edeb7fb3671e487504bee74e998".to_string(),
    };
    if catalog.source != want_source {
        return Err(err("music catalog FDMUS identity mismatch"));
    }
    let lp = &catalog.loop_;
    if lp.mode != "whole_file_runtime_repeat"
        || lp.accepted_counts.len() != 2
        || lp.accepted_counts[0] != 0
        || lp.accepted_counts[1] != 1
        || lp.seam_evidence != "unknown"
        || lp.evidence_level != "strong_inference_e1"
    {
        return Err(err("music catalog loop contract mismatch"));
    }
    if catalog.profiles.len() != 2 {
        return Err(err(format!("music catalog profiles={}, want 2", catalog.profiles.len())));
    }
    for profile in ["fm", "mt32"] {
        let ok = catalog.profiles.get(profile).is_some_and(|meta| {
            !meta.render_pipeline.is_empty()
                && meta.provenance_status == "incomplete_legacy_render"
                && !meta.rights_note.is_empty()
        });
        if !ok {
            return Err(err(format!("music catalog profile {} invalid", go_quote(profile))));
        }
    }
    if catalog.tracks.len() != EXPECTED_TRACKS.len() {
        return Err(err(format!("music catalog tracks={}, want {}", catalog.tracks.len(), EXPECTED_TRACKS.len())));
    }
    catalog.by_id = BTreeMap::new();
    catalog.root = assets_root.to_string();
    for (position, &resource) in EXPECTED_TRACKS.iter().enumerate() {
        let track = catalog.tracks[position].clone();
        let expected_id = format!("FDMUS_{resource:03}");
        if track.track_id != expected_id || track.resource_index != resource {
            return Err(err(format!("music catalog track {position} identity mismatch")));
        }
        if catalog.by_id.contains_key(&track.track_id) {
            return Err(err(format!("music catalog duplicate track {}", go_quote(&track.track_id))));
        }
        if track.renders.len() != 2 {
            return Err(err(format!(
                "music catalog track {} renders={}",
                go_quote(&track.track_id),
                track.renders.len()
            )));
        }
        for profile in ["fm", "mt32"] {
            let Some(render) = track.renders.get(profile) else {
                return Err(err(format!("music catalog track {} lacks {profile}", go_quote(&track.track_id))));
            };
            let expected_path = format!("music_{profile}/{expected_id}.ogg");
            if render.path != expected_path || render.path.starts_with('/') || render.path.contains("..") {
                return Err(err(format!(
                    "music catalog track {} profile {profile} path {}",
                    go_quote(&track.track_id),
                    go_quote(&render.path)
                )));
            }
            validate_render(&go_filepath::join(&[assets_root, &render.path]), render).map_err(|e| {
                e.wrap(format!("music catalog track {} profile {profile}: ", go_quote(&track.track_id)))
            })?;
        }
        catalog.by_id.insert(track.track_id.clone(), track);
    }
    Ok(catalog)
}

impl Catalog {
    /// Go：`catalog.go:132`。Go 對 nil 接收者回 `music profile %q unavailable`；Rust 的 `self` 不會是
    /// null，那條分支只剩 profile 名稱的檢查。
    pub fn resolve(&self, profile: &str, track_id: &str) -> Result<String> {
        if profile != "fm" && profile != "mt32" {
            return Err(err(format!("music profile {} unavailable", go_quote(profile))));
        }
        let Some(track) = self.by_id.get(track_id) else {
            return Err(err(format!("music track {} unavailable", go_quote(track_id))));
        };
        // Go 以零值 Render 查不存在的 profile；Load 已保證兩個 profile 都在。
        let path = track.renders.get(profile).map(|r| r.path.as_str()).unwrap_or_default();
        Ok(go_filepath::join(&[&self.root, path]))
    }
}

/// Go：`catalog.go:143`
fn validate_render(path: &str, expected: &Render) -> Result<()> {
    let raw = std::fs::read(path).map_err(|e| err(format!("open {path}: {e}")))?;
    let sum = Sha256::digest(&raw);
    let hex: String = sum.iter().map(|b| format!("{b:02x}")).collect();
    if raw.len() as i64 != expected.bytes || hex != expected.sha256 {
        return Err(err("identity mismatch"));
    }
    let (channels, rate, samples) = inspect_vorbis(&raw)?;
    // Go 的 uint64 乘加會繞回；rate 為 0 時 Go 整數除以零會 panic，Rust 同樣 panic。
    let duration = samples.wrapping_mul(1000).wrapping_add(rate as u64 / 2) / rate as u64;
    if expected.codec != "vorbis"
        || channels != expected.channels
        || rate != expected.sample_rate
        || samples != expected.pcm_samples
        || duration != expected.duration_ms
        || channels != 2
    {
        return Err(err("Vorbis geometry mismatch"));
    }
    Ok(())
}

fn le_u64(b: &[u8]) -> u64 {
    u64::from_le_bytes(b[..8].try_into().expect("8 位元組"))
}

fn le_u32(b: &[u8]) -> u32 {
    u32::from_le_bytes(b[..4].try_into().expect("4 位元組"))
}

/// Go：`catalog.go:163`，回傳 `(channels, sampleRate int, pcmSamples uint64, err error)`。
pub(crate) fn inspect_vorbis(raw: &[u8]) -> Result<(i64, i64, u64)> {
    let mut offset = 0usize;
    let mut packet: Vec<u8> = Vec::with_capacity(64);
    let mut identification: Option<Vec<u8>> = None;
    let mut last_granule = 0u64;
    while offset < raw.len() {
        if offset + 27 > raw.len() || &raw[offset..offset + 4] != b"OggS" || raw[offset + 4] != 0 {
            return Err(err(format!("invalid Ogg page at {offset}")));
        }
        let segments = raw[offset + 26] as usize;
        let header_end = offset + 27 + segments;
        if header_end > raw.len() {
            return Err(err("truncated Ogg segment table"));
        }
        let body_length: usize = raw[offset + 27..header_end].iter().map(|&l| l as usize).sum();
        let body_end = header_end + body_length;
        if body_end > raw.len() {
            return Err(err("truncated Ogg body"));
        }
        let granule = le_u64(&raw[offset + 6..offset + 14]);
        if granule != u64::MAX {
            last_granule = granule;
        }
        let mut cursor = header_end;
        for &length_byte in &raw[offset + 27..header_end] {
            let length = length_byte as usize;
            packet.extend_from_slice(&raw[cursor..cursor + length]);
            cursor += length;
            if length < 255 {
                // Go `append([]byte(nil), packet...)` 對空封包回傳 nil，所以空的第一個封包不算，
                // 由下一個封包當識別封包。
                if identification.is_none() && !packet.is_empty() {
                    identification = Some(packet.clone());
                }
                packet.clear();
            }
        }
        offset = body_end;
    }
    let identification = identification.unwrap_or_default();
    if offset != raw.len()
        || !packet.is_empty()
        || identification.len() < 16
        || &identification[..7] != b"\x01vorbis"
        || last_granule == 0
    {
        return Err(err("invalid Vorbis stream"));
    }
    if le_u32(&identification[7..11]) != 0 {
        return Err(err("unsupported Vorbis version"));
    }
    Ok((identification[11] as i64, le_u32(&identification[12..16]) as i64, last_granule))
}

#[cfg(test)]
mod catalog_test;
