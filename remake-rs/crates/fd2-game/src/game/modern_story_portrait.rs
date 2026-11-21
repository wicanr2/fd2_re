//! Go 來源：`remake/cmd/fd2/modern_story_portrait.go`（515 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use std::path::{Component, Path};

use serde::{Deserialize, de::DeserializeOwned};
use sha2::{Digest as _, Sha256};

use super::*;

pub(crate) const MODERN_HANDPAINTED_THEME_ID: &str = "modern-handpainted-a";

#[derive(Deserialize)]
struct ThemeCatalog<T> {
    schema_version: u32,
    theme_id: String,
    assets: Vec<T>,
}

pub(crate) fn read_theme_catalog<T: DeserializeOwned>(path: &Path) -> Result<Vec<T>> {
    let catalog: ThemeCatalog<T> =
        serde_json::from_slice(&read_theme_file(path)?).map_err(|err| game_err!("modern theme catalog: {err}"))?;
    if catalog.schema_version != 1 || catalog.theme_id != MODERN_HANDPAINTED_THEME_ID {
        return Err(game_err!("modern theme catalog: unsupported identity"));
    }
    Ok(catalog.assets)
}

pub(crate) fn theme_file_extension(name: &str) -> Option<&str> {
    let path = Path::new(name);
    let mut parts = path.components();
    match (parts.next(), parts.next()) {
        (Some(Component::Normal(base)), None) if base == name && !name.contains('\\') => path.extension()?.to_str(),
        _ => None,
    }
}

pub(crate) fn read_theme_file(path: &Path) -> Result<Vec<u8>> {
    std::fs::read(path).map_err(|err| game_err!("modern theme: {}: {err}", path.display()))
}

pub(crate) fn read_verified_theme_file(path: &Path, sha256: &str) -> Result<Vec<u8>> {
    let bytes = read_theme_file(path)?;
    if format!("{:x}", Sha256::digest(&bytes)) != sha256 {
        return Err(game_err!("modern theme: SHA-256 differs for {}", path.display()));
    }
    Ok(bytes)
}

/// Go：`modern_story_portrait.go:21`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernStoryPortraitFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) hd_image: Option<Arc<Picture>>,
}

/// Go：`modern_story_portrait.go:27`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernStoryPortraitSet {
    pub(crate) has_hd: bool,
    pub(crate) portraits: BTreeMap<i64, ModernStoryPortraitFrame>,
    pub(crate) map_sprites: BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    pub(crate) map_sprites_hd: BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    pub(crate) map_tilesets: BTreeMap<i64, Option<Arc<Picture>>>,
    pub(crate) battle_action_icons: Vec<Option<Arc<Picture>>>,
    pub(crate) battle_animations: BTreeMap<i64, ModernBattleHDResource>,
    pub(crate) battle_surfaces: BTreeMap<String, BTreeMap<i64, ModernBattleSurfaceHD>>,
    pub(crate) map_overlays: Option<ModernMapOverlayHDResource>,
    pub(crate) physical_sfx: BTreeMap<[i64; 2], Vec<u8>>,
    pub(crate) battle_presentation: Option<ModernBattlePresentationProfile>,
}
