//! 移植自 Go `separated_palette.go`：固定版本 FDOTHER.DAT 的具名 VGA DAC。

use super::go_compat::{nullable, nullable_struct, nullable_vec};
use super::separated_surface::read_document;
use super::{FdotherError, Palette, Result, parse_vga_palette};
use serde::Deserialize;
use std::path::Path;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub(super) struct SeparatedPaletteSource {
    #[serde(deserialize_with = "nullable")]
    pub file: String,
    #[serde(deserialize_with = "nullable")]
    pub resource: i64,
    #[serde(deserialize_with = "nullable")]
    pub size: i64,
    #[serde(deserialize_with = "nullable")]
    pub md5: String,
    #[serde(deserialize_with = "nullable")]
    pub sha256: String,
    #[serde(deserialize_with = "nullable")]
    pub raw_size: i64,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub(super) struct SeparatedPaletteDocument {
    #[serde(deserialize_with = "nullable")]
    pub schema_version: i64,
    #[serde(deserialize_with = "nullable")]
    pub asset_id: String,
    #[serde(deserialize_with = "nullable_struct")]
    pub source: SeparatedPaletteSource,
    #[serde(deserialize_with = "nullable_vec", rename = "dac_6bit_components")]
    pub components: Vec<i64>,
}

/// 載入固定版本 FDOTHER.DAT 的具名 VGA DAC。不讀取也不退回原始封存。
pub fn load_separated_fdother_palette(palette_root: impl AsRef<Path>, resource: i64) -> Result<(Vec<u8>, Palette)> {
    let palette_root = palette_root.as_ref();
    if palette_root.as_os_str().is_empty() || ![0, 8, 57, 76, 99, 101, 102].contains(&resource) {
        return Err(FdotherError::message(format!("fdother: unsupported separated palette request {resource}")));
    }
    let context = "fdother: separated palette metadata: ";
    let doc: SeparatedPaletteDocument =
        read_document(&palette_root.join(format!("fdother_{resource:03}.json")), context, context)?;
    if doc.schema_version != 1
        || doc.asset_id != format!("palette/fdother_{resource:03}")
        || doc.source.file != "FDOTHER.DAT"
        || doc.source.resource != resource
        || doc.source.size != 3382481
        || doc.source.md5 != "22f56e5027edc7c766ad34ca4e5aca93"
        || doc.source.sha256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
        || doc.source.raw_size != 768
        || doc.components.len() != 768
    {
        return Err(FdotherError::message(format!("fdother: separated palette {resource} contract mismatch")));
    }
    let mut dac = Vec::with_capacity(768);
    for (index, &component) in doc.components.iter().enumerate() {
        if !(0..=63).contains(&component) {
            return Err(FdotherError::message(format!(
                "fdother: separated palette {resource} component {index} outside 0..63"
            )));
        }
        dac.push(component as u8);
    }
    let palette = parse_vga_palette(&dac)?;
    Ok((dac, palette))
}
