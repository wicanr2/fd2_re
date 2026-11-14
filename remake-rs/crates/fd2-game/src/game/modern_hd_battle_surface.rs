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
    todo!("modern_hd_battle_surface.go:modernPhysicalBaseScenery")
}

/// Go：`modern_hd_battle_surface.go:199`，`func modernPhysicalPlatformSource(scene *nativePhysicalScene) (modernHDBattleSource, error)`
pub(crate) fn modern_physical_platform_source(scene: &NativePhysicalScene) -> Result<ModernHDBattleSource> {
    todo!("modern_hd_battle_surface.go:modernPhysicalPlatformSource")
}

/// Go：`modern_hd_battle_surface.go:203`，`func modernBattlePlatformSource(platform fdother.Frame, resource int) (modernHDBattleSource, error)`
pub(crate) fn modern_battle_platform_source(platform: &fdother::Frame, resource: i64) -> Result<ModernHDBattleSource> {
    todo!("modern_hd_battle_surface.go:modernBattlePlatformSource")
}

/// Go：`modern_hd_battle_surface.go:216`，`func modernPhysicalScenery(scene *nativePhysicalScene, platform bool) ([]modernHDBattleSource, error)`
pub(crate) fn modern_physical_scenery(
    scene: Option<&NativePhysicalScene>,
    platform: bool,
) -> Result<Vec<ModernHDBattleSource>> {
    todo!("modern_hd_battle_surface.go:modernPhysicalScenery")
}

/// Go：`modern_hd_battle_surface.go:223`，`func modernBattleBackdropGeometry(frame fdother.Frame) bool`
pub(crate) fn modern_battle_backdrop_geometry(frame: &fdother::Frame) -> bool {
    todo!("modern_hd_battle_surface.go:modernBattleBackdropGeometry")
}

/// Go：`modern_hd_battle_surface.go:228`，`func modernPhysicalBackdropScenery(scene *nativePhysicalScene, background fdother.Frame, index byte, platform bool) ([]modernHDBattleSource, error)`
pub(crate) fn modern_physical_backdrop_scenery(
    scene: &NativePhysicalScene,
    background: &fdother::Frame,
    index: u8,
    platform: bool,
) -> Result<Vec<ModernHDBattleSource>> {
    todo!("modern_hd_battle_surface.go:modernPhysicalBackdropScenery")
}

/// 只還原已證實由memset供應、且不與BG／面板交會的台座底層。
/// 不改job的原生base；原生及HD-native fallback分別重組驗證。
///
/// Go：`modern_hd_battle_surface.go:252`，`func bindModernPhysicalScenery(scene *nativePhysicalScene, sources *modernHDBattleSources) (err error)`
pub(crate) fn bind_modern_physical_scenery(
    scene: &mut NativePhysicalScene,
    sources: Option<&mut ModernHDBattleSources>,
) -> Result<()> {
    todo!("modern_hd_battle_surface.go:bindModernPhysicalScenery")
}

/// Go：`modern_hd_battle_surface.go:268`，`func bindModernPhysicalBaseScenery(sources *modernHDBattleSources, scenery []modernHDBattleSource) error`
pub(crate) fn bind_modern_physical_base_scenery(
    sources: Option<&mut ModernHDBattleSources>,
    scenery: &mut [ModernHDBattleSource],
) -> Result<()> {
    todo!("modern_hd_battle_surface.go:bindModernPhysicalBaseScenery")
}
