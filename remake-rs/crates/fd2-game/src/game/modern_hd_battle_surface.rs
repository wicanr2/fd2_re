//! Go 來源：`remake/cmd/fd2/modern_hd_battle_surface.go`（292 行）。
//!
//! 骨架：Go 檔的 14 個函式中產生 8 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// 靜態場景圖完整解碼後才發布。原始像素保留供同次 producer 身分核對。
///
/// Go：`modern_hd_battle_surface.go:18`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernBattleSurfaceHD {}

/// Go：`modern_hd_battle_surface.go:185`，`func modernPhysicalBaseScenery(scene *nativePhysicalScene) ([]modernHDBattleSource, error)`
pub(crate) fn modern_physical_base_scenery(scene: Option<&NativePhysicalScene>) -> Result<Vec<ModernHDBattleSource>> {
    match scene {
        Some(scene) if !scene.selection.has_separate_backgrounds => modern_physical_scenery(Some(scene), true),
        _ => Ok(Vec::new()),
    }
}

/// Go：`modern_hd_battle_surface.go:192`，`func modernPhysicalTargetBGIndex(scene *nativePhysicalScene) byte`
/// （`r8`，隨呼叫端 `attachNativePhysicalBody` 補移植）。
pub(crate) fn modern_physical_target_bg_index(scene: &NativePhysicalScene) -> u8 {
    if scene.raw_side == 0 {
        return scene.selection.secondary_bg;
    }
    scene.selection.primary_bg
}

/// Go：`modern_hd_battle_surface.go:199`，`func modernPhysicalPlatformSource(scene *nativePhysicalScene) (modernHDBattleSource, error)`
pub(crate) fn modern_physical_platform_source(scene: &NativePhysicalScene) -> Result<ModernHDBattleSource> {
    modern_battle_platform_source(&scene.platform_frame, scene.selection.tai as i64)
}

/// Go：`modern_hd_battle_surface.go:203`，`func modernBattlePlatformSource(platform fdother.Frame, resource int) (modernHDBattleSource, error)`
pub(crate) fn modern_battle_platform_source(platform: &fdother::Frame, resource: i64) -> Result<ModernHDBattleSource> {
    let mut frame = modern_hd_battle_surface_frame(platform)?;
    (frame.x, frame.y) = (164, 157);
    let rect = (frame.x, frame.y, frame.x + frame.width, frame.y + frame.height);
    // Go `image.Rectangle.In`／`Overlaps`（非空矩形；寬高已由 modernHDBattleSurfaceFrame 保證為正）。
    let inside = rect.0 >= 0 && rect.1 >= 0 && rect.2 <= 320 && rect.3 <= 200;
    let overlaps = |x0: i64, y0: i64, x1: i64, y1: i64| rect.0 < x1 && x0 < rect.2 && rect.1 < y1 && y0 < rect.3;
    if !inside || overlaps(0, 154, 149, 196) || overlaps(171, 4, 320, 46) || overlaps(0, 50, 320, 150) {
        return Err(game_err!("modern HD physical platform overlaps a native base component"));
    }
    Ok(ModernHDBattleSource { archive: String::from("TAI.DAT"), resource, frame, fill: -1, ..Default::default() })
}

/// Go：`modern_hd_battle_surface.go:216`，`func modernPhysicalScenery(scene *nativePhysicalScene, platform bool) ([]modernHDBattleSource, error)`
pub(crate) fn modern_physical_scenery(
    scene: Option<&NativePhysicalScene>,
    platform: bool,
) -> Result<Vec<ModernHDBattleSource>> {
    let Some(scene) = scene else {
        return Ok(Vec::new());
    };
    modern_physical_backdrop_scenery(scene, &scene.background_frame, scene.selection.base_bg, platform)
}

/// Go：`modern_hd_battle_surface.go:223`，`func modernBattleBackdropGeometry(frame fdother.Frame) bool`
pub(crate) fn modern_battle_backdrop_geometry(frame: &fdother::Frame) -> bool {
    (frame.width == 320 && (frame.height == 100 || frame.height == 101)) || (frame.width == 318 && frame.height == 100)
}

/// Go：`modern_hd_battle_surface.go:228`，`func modernPhysicalBackdropScenery(scene *nativePhysicalScene, background fdother.Frame, index byte, platform bool) ([]modernHDBattleSource, error)`
pub(crate) fn modern_physical_backdrop_scenery(
    scene: &NativePhysicalScene,
    background: &fdother::Frame,
    index: u8,
    platform: bool,
) -> Result<Vec<ModernHDBattleSource>> {
    // 三種原框均落於y50..151，不與面板y4..46及y154..196交會。
    // 固定來源及READY契約見61「其餘可見戰鬥背景與台座HD」。
    if !modern_battle_backdrop_geometry(background) {
        return Ok(Vec::new());
    }
    let mut frame = modern_hd_battle_surface_frame(background)?;
    (frame.x, frame.y) = (0, 50);
    let mut sources = vec![ModernHDBattleSource {
        archive: String::from("BG.DAT"),
        resource: index as i64,
        frame,
        fill: -1,
        ..Default::default()
    }];
    if platform {
        sources.push(modern_physical_platform_source(scene)?);
    }
    Ok(sources)
}

/// 只還原已證實由memset供應、且不與BG／面板交會的台座底層。
/// 不改job的原生base；原生及HD-native fallback分別重組驗證。
///
/// Go：`modern_hd_battle_surface.go:252`，`func bindModernPhysicalScenery(scene *nativePhysicalScene, sources *modernHDBattleSources) (err error)`
pub(crate) fn bind_modern_physical_scenery(
    scene: &mut NativePhysicalScene,
    sources: Option<&mut ModernHDBattleSources>,
) -> Result<()> {
    // Go 以 defer 在成功時接著 bindModernPhysicalPanelText。
    let Some(sources) = sources else {
        modern_physical_base_scenery(Some(scene))?;
        return Ok(());
    };
    let mut scenery = modern_physical_base_scenery(Some(scene))?;
    if !scenery.is_empty() {
        bind_modern_physical_base_scenery(Some(sources), &mut scenery)?;
    }
    bind_modern_physical_panel_text(Some(scene), Some(sources));
    Ok(())
}

/// Go：`modern_hd_battle_surface.go:268`，`func bindModernPhysicalBaseScenery(sources *modernHDBattleSources, scenery []modernHDBattleSource) error`
pub(crate) fn bind_modern_physical_base_scenery(
    sources: Option<&mut ModernHDBattleSources>,
    scenery: &mut [ModernHDBattleSource],
) -> Result<()> {
    let Some(sources) = sources else {
        return Ok(());
    };
    if scenery.is_empty() {
        return Ok(());
    }
    if sources.base.len() != 64000 {
        return Err(game_err!("modern HD physical clean base unavailable"));
    }
    let mut base = sources.base.clone();
    for source in scenery.iter() {
        if source.archive != "TAI.DAT" {
            continue;
        }
        let platform = &source.frame;
        for y in platform.y..platform.y + platform.height {
            let at = (y * 320 + platform.x) as usize;
            base[at..at + platform.width as usize].fill(0);
        }
    }
    let mut scenery = scenery.to_vec();
    for source in &sources.base_scenery {
        if source.panel_text.is_some() {
            scenery.push(source.clone());
        }
    }
    (sources.base, sources.base_scenery) = (base, scenery);
    Ok(())
}
