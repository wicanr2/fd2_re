//! 移植自 Go `separated_range_overlay.go`：FDOTHER #1 戰術範圍覆蓋層的分離描述子庫。

use super::go_compat::{GoImage, decode_png_file, nullable, nullable_struct, nullable_struct_vec};
use super::range_overlay::NATIVE_RANGE_OVERLAY_TILES;
use super::separated_item_panel::PngLoadError;
use super::separated_surface::read_document;
use super::{FdotherError, Result};
use crate::fdicon::{Bank, NATIVE_SIZE, Sprite};
use serde::Deserialize;
use std::path::Path;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedRangeOverlaySource {
    #[serde(deserialize_with = "nullable")]
    file: String,
    #[serde(deserialize_with = "nullable")]
    resource: i64,
    #[serde(deserialize_with = "nullable")]
    size: i64,
    #[serde(deserialize_with = "nullable")]
    md5: String,
    #[serde(deserialize_with = "nullable")]
    sha256: String,
    #[serde(deserialize_with = "nullable")]
    raw_size: i64,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedRangeOverlaySprite {
    #[serde(deserialize_with = "nullable")]
    index: i64,
    #[serde(deserialize_with = "nullable")]
    width: i64,
    #[serde(deserialize_with = "nullable")]
    height: i64,
    #[serde(deserialize_with = "nullable")]
    frame: String,
    #[serde(deserialize_with = "nullable")]
    mask: String,
    #[serde(deserialize_with = "nullable")]
    remap_mask: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedRangeOverlayDocument {
    #[serde(deserialize_with = "nullable")]
    schema_version: i64,
    #[serde(deserialize_with = "nullable")]
    kind: String,
    #[serde(deserialize_with = "nullable")]
    asset_id: String,
    #[serde(deserialize_with = "nullable")]
    status: String,
    #[serde(deserialize_with = "nullable")]
    evidence: String,
    #[serde(deserialize_with = "nullable_struct")]
    source: SeparatedRangeOverlaySource,
    #[serde(deserialize_with = "nullable_struct_vec")]
    sprites: Vec<SeparatedRangeOverlaySprite>,
}

/// 載入戰術範圍覆蓋層與整備游標共用的完整 FDOTHER #1 描述子庫。
///
/// 項目數恰為 20 且索引唯一，所以 `is missing` 錯誤在 Go 與這裡都不會發生；照搬 Go 的檢查。
pub fn load_separated_range_overlay_bank(root: impl AsRef<Path>) -> Result<Bank> {
    let root = root.as_ref();
    let document: SeparatedRangeOverlayDocument = read_document(
        &root.join("bank.json"),
        "fdother: separated range-overlay metadata: ",
        "fdother: separated range-overlay JSON: ",
    )?;
    let tiles = NATIVE_RANGE_OVERLAY_TILES;
    if document.schema_version != 1
        || document.kind != "fdother_sprite_bank"
        || document.asset_id != "sprites/FDOTHER_001/range_overlay"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != 1
        || document.source.size != 3382481
        || document.source.md5 != "22f56e5027edc7c766ad34ca4e5aca93"
        || document.source.sha256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
        || document.source.raw_size != 2235
        || document.sprites.len() as i64 != tiles
    {
        return Err(FdotherError::fixed("fdother: separated range-overlay contract mismatch"));
    }
    let mut bank = Bank { sprites: vec![Sprite::default(); tiles as usize] };
    let mut seen = vec![false; tiles as usize];
    for entry in &document.sprites {
        let index = entry.index;
        if index < 0
            || index >= tiles
            || seen[index as usize]
            || entry.width != NATIVE_SIZE
            || entry.height != NATIVE_SIZE
        {
            return Err(FdotherError::message(format!("fdother: invalid separated range-overlay sprite {index}")));
        }
        let prefix = format!("sprite_{index:04}/");
        if entry.frame != format!("{prefix}frame.png")
            || entry.mask != format!("{prefix}mask.png")
            || entry.remap_mask != format!("{prefix}remap_mask.png")
        {
            return Err(FdotherError::message(format!("fdother: range-overlay sprite {index} path mismatch")));
        }
        let pixels = load_range_overlay_indexed(&root.join(&entry.frame))
            .map_err(|e| e.wrap(format!("fdother: range-overlay sprite {index} frame: ")))?;
        let mask = load_range_overlay_mask(&root.join(&entry.mask))
            .map_err(|e| e.wrap(format!("fdother: range-overlay sprite {index} mask: ")))?;
        let remap_mask = load_range_overlay_mask(&root.join(&entry.remap_mask))
            .map_err(|e| e.wrap(format!("fdother: range-overlay sprite {index} remap mask: ")))?;
        bank.sprites[index as usize] = Sprite { pixels, mask, remap_mask };
        seen[index as usize] = true;
    }
    if let Some(index) = seen.iter().position(|&present| !present) {
        return Err(FdotherError::message(format!("fdother: range-overlay sprite {index} is missing")));
    }
    Ok(bank)
}

/// Go 以 `image.Decode` 讀（先比對檔頭）；要求 24×24 且調色盤恰好 256 項。
fn load_range_overlay_indexed(path: &Path) -> std::result::Result<Vec<u8>, PngLoadError> {
    match decode_png_file(path, true).map_err(PngLoadError::External)? {
        GoImage::Paletted { width: 24, height: 24, pix, palette_len: 256 } => Ok(pix),
        _ => Err(PngLoadError::Contract("indexed PNG must be 24x24 with 256 palette entries")),
    }
}

/// 24×24 Gray PNG，非 0 值寫成 1；值只能是 0／0xff。
fn load_range_overlay_mask(path: &Path) -> std::result::Result<Vec<u8>, PngLoadError> {
    let GoImage::Gray { width: 24, height: 24, pix } = decode_png_file(path, true).map_err(PngLoadError::External)?
    else {
        return Err(PngLoadError::Contract("mask PNG must be 24x24 grayscale"));
    };
    let mut mask = vec![0u8; pix.len()];
    for (i, &value) in pix.iter().enumerate() {
        if value != 0 && value != 0xff {
            return Err(PngLoadError::Contract("mask PNG contains a non-binary value"));
        }
        if value != 0 {
            mask[i] = 1;
        }
    }
    Ok(mask)
}
