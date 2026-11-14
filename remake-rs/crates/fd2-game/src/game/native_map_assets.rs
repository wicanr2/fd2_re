//! Go 來源：`remake/cmd/fd2/native_map_assets.go`（155 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeMapAssets is an all-or-nothing original-resource bundle. It remains
/// separate from the PNG renderer until indexed presentation is ready.
///
/// Go：`native_map_assets.go:14`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeMapAssets {
    pub(crate) map_index: i64,
    pub(crate) frames: indexedmap::NativeMapHUDFrames,
    pub(crate) terrain: Option<fdicon::Bank>,
    pub(crate) range: Option<fdicon::Bank>,
    pub(crate) units: Option<fdicon::Bank>,
    pub(crate) controls: Vec<u8>,
    /// LUTs is FDOTHER#3's raw 256-byte remap bank. Entries 1..9 are the
    /// verified 0x24618 transition selectors; loading them here does not by
    /// itself authorize scene presentation.
    pub(crate) luts: Vec<Vec<u8>>,
    pub(crate) palette: ColorPalette,
    /// PaletteDAC retains FDOTHER#0's exact 768-byte, six-bit DAC baseline.
    /// Native 0x11df2 effects require mutable raw components; color.Palette is
    /// presentation-only and must not be used as the effect state.
    pub(crate) palette_dac: Vec<u8>,
    /// SpawnIntro is FDOTHER #9's exact twelve-entry LMI1 bank consumed by
    /// sub_32999. Map initialization preflights it with the shared battle bundle
    /// so an otherwise valid map cannot fail halfway through a roster event.
    pub(crate) spawn_intro: Vec<fdother::LMI1Entry>,
    /// CommandHealDigits is the complete FDOTHER #5 LMI1 bank. The historical
    /// field name is retained for existing command-tail callers; ch28 post also
    /// consumes exact entries 0x44..0x4f without assigning them semantic names.
    /// FDOTHER6 is the complete shared #6 bank consumed by command 13..16 and
    /// 0x22253; neither path may rename the archive itself after one effect.
    pub(crate) command_heal_digits: Vec<fdother::LMI1Entry>,
    pub(crate) fdother6: Vec<fdother::LMI1Entry>,
    pub(crate) chapter_aux: Option<fdother::NativeChapterAuxSurface>,
    pub(crate) parallax: Option<fdother::NativeMapParallaxSurface>,
}

/// Go：`native_map_assets.go:45`，`func loadNativeMapAssets(mapDir string) (*nativeMapAssets, error)`
pub(crate) fn load_native_map_assets(map_dir: &str) -> Result<Option<NativeMapAssets>> {
    todo!("native_map_assets.go:loadNativeMapAssets")
}

/// Go：`native_map_assets.go:125`，`func nativeMapAssetsAvailable(a *nativeMapAssets) bool`
pub(crate) fn native_map_assets_available(a: Option<&NativeMapAssets>) -> bool {
    todo!("native_map_assets.go:nativeMapAssetsAvailable")
}

/// Go：`native_map_assets.go:145`，`func nativeSpawnIntroAssetsAvailable(a *nativeMapAssets) bool`
pub(crate) fn native_spawn_intro_assets_available(a: &NativeMapAssets) -> bool {
    todo!("native_map_assets.go:nativeSpawnIntroAssetsAvailable")
}
