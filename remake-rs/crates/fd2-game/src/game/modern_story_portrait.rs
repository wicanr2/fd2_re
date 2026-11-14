//! Go 來源：`remake/cmd/fd2/modern_story_portrait.go`（515 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

pub(crate) const MODERN_HANDPAINTED_THEME_ID: &str = "modern-handpainted-a";

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
