//! 移植自 Go `separated_sound.go`：固定 FDOTHER 巢狀 PCM 音效庫的 OGG 分離素材。

use super::go_compat::{decode_hex, nullable, nullable_struct, nullable_struct_vec, read_file};
use super::separated_surface::read_document;
use super::{FdotherError, Result};
use serde::Deserialize;
use std::collections::BTreeMap;
use std::path::Path;

pub(super) const SEPARATED_FDOTHER_SIZE: i64 = 3382481;
pub(super) const SEPARATED_FDOTHER_MD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";
pub(super) const SEPARATED_FDOTHER_SHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";

/// 已納入契約的音效庫與其樣本數（Go `separatedSoundCounts`）。
pub(super) fn separated_sound_count(resource: i64) -> Option<i64> {
    Some(match resource {
        31 => 13,
        48 => 6,
        49 => 7,
        50 | 51 => 5,
        52 => 6,
        53 => 4,
        77 => 4,
        78 => 1,
        80 => 16,
        82 => 2,
        83 => 4,
        84 => 3,
        85 | 86 => 2,
        87 => 4,
        88 => 2,
        90 | 91 => 3,
        92 | 93 => 2,
        94 => 3,
        95 => 1,
        _ => return None,
    })
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedSoundSource {
    #[serde(deserialize_with = "nullable")]
    name: String,
    #[serde(deserialize_with = "nullable")]
    size: i64,
    #[serde(deserialize_with = "nullable")]
    md5: String,
    #[serde(deserialize_with = "nullable")]
    sha256: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedSoundSample {
    #[serde(deserialize_with = "nullable")]
    subresource: i64,
    #[serde(deserialize_with = "nullable")]
    source_byte_count: i64,
    #[serde(deserialize_with = "nullable", rename = "source_pcm_sha256")]
    source_pcm_sha256: String,
    #[serde(deserialize_with = "nullable")]
    path: String,
    #[serde(deserialize_with = "nullable")]
    cue_evidence: String,
    #[serde(deserialize_with = "nullable")]
    classification_evidence: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedSoundDocument {
    #[serde(deserialize_with = "nullable")]
    schema_version: i64,
    #[serde(deserialize_with = "nullable")]
    kind: String,
    #[serde(deserialize_with = "nullable")]
    asset_id: String,
    #[serde(deserialize_with = "nullable")]
    status: String,
    #[serde(deserialize_with = "nullable_struct")]
    source: SeparatedSoundSource,
    #[serde(deserialize_with = "nullable")]
    resource: i64,
    #[serde(deserialize_with = "nullable")]
    container_count: i64,
    #[serde(deserialize_with = "nullable")]
    zero_length_tail_index: i64,
    #[serde(deserialize_with = "nullable")]
    sample_rate: i64,
    #[serde(deserialize_with = "nullable")]
    channels: i64,
    #[serde(deserialize_with = "nullable")]
    sample_format: String,
    #[serde(deserialize_with = "nullable")]
    timing_evidence: String,
    #[serde(deserialize_with = "nullable_struct_vec")]
    samples: Vec<SeparatedSoundSample>,
}

/// 保存固定 FDOTHER 巢狀 PCM 音效庫的 OGG 編碼內容。正式執行期只讀分離素材，不會退回
/// 原始封存。Go 以 `map[int][]byte` 保存，這裡用 `BTreeMap`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SeparatedSoundBank {
    pub resource: i64,
    pub encoded: BTreeMap<i64, Vec<u8>>,
}

/// 嚴格驗證一個已納入契約的 FDOTHER 音效庫及其所有 OGG 檔。
///
/// 路徑必須等於 `sample_%03d.ogg`，所以 Go 的 `filepath.Base` 與 `..` 檢查不會再擋下任何
/// 東西；樣本至少 4 位元組，所以 `missing subresource` 只發生在真的缺號時。兩者都照搬。
pub fn load_separated_sound_bank(sfx_root: impl AsRef<Path>, resource: i64) -> Result<SeparatedSoundBank> {
    let sfx_root = sfx_root.as_ref();
    let want_count = match separated_sound_count(resource) {
        Some(count) if !sfx_root.as_os_str().is_empty() => count,
        _ => return Err(FdotherError::message(format!("fdother: unsupported separated sound request {resource}"))),
    };
    let bank_root = sfx_root.join(format!("FDOTHER_{resource:03}"));
    let context = format!("fdother: separated sound metadata {resource}: ");
    let doc: SeparatedSoundDocument = read_document(&bank_root.join("resource.json"), &context, &context)?;
    if doc.schema_version != 2
        || doc.kind != "fd2_pcm_sound_bank"
        || doc.asset_id != format!("sfx/FDOTHER_{resource:03}")
        || doc.status != "converted"
        || doc.source.name != "FDOTHER.DAT"
        || doc.source.size != SEPARATED_FDOTHER_SIZE
        || doc.source.md5 != SEPARATED_FDOTHER_MD5
        || doc.source.sha256 != SEPARATED_FDOTHER_SHA256
        || doc.resource != resource
        || doc.container_count != want_count + 1
        || doc.zero_length_tail_index != want_count
        || doc.sample_rate != 11025
        || doc.channels != 1
        || doc.sample_format != "unsigned_u8"
        || doc.timing_evidence != "hardware-spec_approximation"
        || doc.samples.len() as i64 != want_count
    {
        return Err(FdotherError::message(format!("fdother: separated sound {resource} contract mismatch")));
    }
    let mut bank = SeparatedSoundBank { resource, encoded: BTreeMap::new() };
    for sample in &doc.samples {
        let sub = sample.subresource;
        let digest = decode_hex(&sample.source_pcm_sha256);
        if sub < 0
            || sub >= want_count
            || sample.source_byte_count <= 0
            || digest.as_ref().is_none_or(|d| d.len() != 32)
            || sample.cue_evidence.is_empty()
            || (sample.classification_evidence != "confirmed" && sample.classification_evidence != "strong_inference")
        {
            return Err(FdotherError::message(format!("fdother: separated sound {resource} invalid sample metadata")));
        }
        if bank.encoded.contains_key(&sub) {
            return Err(FdotherError::message(format!(
                "fdother: separated sound {resource} duplicate subresource {sub}"
            )));
        }
        let want_path = format!("sample_{sub:03}.ogg");
        if sample.path != want_path || go_base(&sample.path) != sample.path || sample.path.contains("..") {
            return Err(FdotherError::message(format!("fdother: separated sound {resource} unsafe sample path")));
        }
        let encoded = read_file(&bank_root.join(&sample.path)).map_err(|e| {
            FdotherError::external(format!("fdother: separated sound {resource} subresource {sub}: "), e)
        })?;
        if encoded.len() < 4 || &encoded[..4] != b"OggS" {
            return Err(FdotherError::message(format!(
                "fdother: separated sound {resource} subresource {sub} is not OGG"
            )));
        }
        bank.encoded.insert(sub, encoded);
    }
    for index in 0..want_count {
        if bank.encoded.get(&index).is_none_or(Vec::is_empty) {
            return Err(FdotherError::message(format!(
                "fdother: separated sound {resource} missing subresource {index}"
            )));
        }
    }
    Ok(bank)
}

/// Go `filepath.Base`（Unix）：去掉結尾斜線後取最後一段；空字串是 `.`，全是斜線是 `/`。
fn go_base(path: &str) -> &str {
    if path.is_empty() {
        return ".";
    }
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        return "/";
    }
    trimmed.rsplit('/').next().unwrap_or(trimmed)
}
