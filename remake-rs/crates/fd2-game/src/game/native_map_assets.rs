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
    let map_index = fdother::map_index_from_asset_path(map_dir)?;
    let frames = indexedmap::load_separated_native_map_hud_frames(separated_asset_path("ui"))?;
    let (terrain, controls) = fdicon::load_separated_fdshap_bank(separated_asset_path("tilesets/fdshap"), map_index)?;
    let units = fdicon::load_separated_bank(separated_asset_path("sprites/fdicon"))?;
    let range_bank =
        fdother::load_separated_range_overlay_bank(separated_asset_path("sprites/fdother_001_range_overlay"))?;
    let luts = fdother::load_separated_lut_bank(separated_asset_path("palette"))?;
    if luts.len() <= 9 {
        return Err(game_err!("native map assets: FDOTHER#3 LUT bank lacks transition entries 1..9"));
    }
    let (palette_raw, palette) = load_native_battle_palette()?;
    let spawn_intro = fdother::load_separated_lmi1_bank(
        separated_asset_path("animations/fdother_009_spawn_intro"),
        fdother::NATIVE_SPAWN_INTRO_FRAME_RESOURCE,
    )?;
    let command_heal_digits = fdother::load_separated_lmi1_bank(
        separated_asset_path("ui/fdother_005_lmi1_opaque"),
        fdother::NATIVE_COMMAND_HEAL_TAIL_DIGIT_RESOURCE,
    )?;
    let fdother6 = fdother::load_separated_lmi1_bank(
        separated_asset_path("effects/fdother_006_lmi1_opaque"),
        fdother::NATIVE_COMMAND_HEAL_TAIL_EFFECT_RESOURCE,
    )?;
    let mut chapter_aux = None;
    if let Some(contract) = fdother::native_chapter_aux_surface_for(map_index) {
        chapter_aux =
            Some(fdother::load_separated_chapter_aux_surface_resource(separated_asset_path("surfaces"), &contract)?);
    }
    let mut parallax = None;
    if map_index == 17 {
        parallax = Some(fdother::load_separated_native_map_parallax(separated_asset_path("surfaces"))?);
    }
    Ok(Some(NativeMapAssets {
        map_index,
        frames,
        terrain: Some(terrain),
        range: Some(range_bank),
        units: Some(units),
        controls,
        luts,
        palette,
        palette_dac: palette_raw,
        spawn_intro,
        command_heal_digits,
        fdother6,
        chapter_aux,
        parallax,
    }))
}

/// Go：`native_map_assets.go:125`，`func nativeMapAssetsAvailable(a *nativeMapAssets) bool`
pub(crate) fn native_map_assets_available(a: Option<&NativeMapAssets>) -> bool {
    let Some(a) = a else {
        return false;
    };
    if a.terrain.is_none()
        || a.range.is_none()
        || a.units.is_none()
        || a.controls.is_empty()
        || a.luts.len() <= 9
        || a.palette.len() != 256
        || a.palette_dac.len() != 256 * 3
    {
        return false;
    }
    for i in 1..=9 {
        if a.luts[i].len() != 256 {
            return false;
        }
    }
    if fdother::native_chapter_aux_surface_for(a.map_index).is_some()
        && !a.chapter_aux.as_ref().is_some_and(|aux| aux.pixels.len() == 320 * 200)
    {
        return false;
    }
    if a.map_index == 17
        && !a.parallax.as_ref().is_some_and(|p| {
            p.pixels.len() as i64 == fdother::NATIVE_MAP_PARALLAX_WIDTH * fdother::NATIVE_MAP_PARALLAX_HEIGHT
        })
    {
        return false;
    }
    true
}

/// Go：`native_map_assets.go:145`，`func nativeSpawnIntroAssetsAvailable(a *nativeMapAssets) bool`
pub(crate) fn native_spawn_intro_assets_available(a: &NativeMapAssets) -> bool {
    if !native_map_assets_available(Some(a)) || a.spawn_intro.len() as i64 != fdother::NATIVE_SPAWN_INTRO_PASS_COUNT {
        return false;
    }
    for entry in &a.spawn_intro {
        if entry.width <= 0 || entry.height <= 0 || entry.pixels.len() as i64 != entry.width * entry.height {
            return false;
        }
    }
    true
}
