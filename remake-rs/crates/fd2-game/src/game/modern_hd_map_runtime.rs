//! Go 來源：`remake/cmd/fd2/modern_hd_map_runtime.go`（349 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::modern_hd_terrain::{SCALE, TerrainColourContext, TerrainMap, TerrainRole};
use super::*;

const HD_MAP_TILE_SIDE: i64 = fdicon::NATIVE_SIZE * SCALE as i64;

pub(crate) const NATIVE_MAP_STEADY_BASE: i64 = 0x8088;

pub(crate) fn hd_map_tile_rect(offset: i64, viewport_base: i64) -> Rectangle {
    let stride = fdicon::NATIVE_MAP_STRIDE;
    let view = modern_hd_map_viewport();
    let x = view.min.x + (offset % stride - viewport_base % stride) * SCALE as i64;
    let y = view.min.y + (offset / stride - viewport_base / stride) * SCALE as i64;
    rect(x, y, x + HD_MAP_TILE_SIDE, y + HD_MAP_TILE_SIDE)
}

pub(crate) fn hd_map_viewport_fits_work(base: i64) -> bool {
    let stride = fdicon::NATIVE_MAP_STRIDE;
    let view = modern_hd_map_viewport();
    let width = (view.max.x - view.min.x) / SCALE as i64;
    let height = (view.max.y - view.min.y) / SCALE as i64;
    let last_base = indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE - (height - 1) * stride - width;
    (0..=last_base).contains(&base) && base % stride + width <= stride
}

pub(crate) fn hd_map_tile_fits_work(offset: i64) -> bool {
    let last = indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
        - (fdicon::NATIVE_SIZE - 1) * fdicon::NATIVE_MAP_STRIDE
        - fdicon::NATIVE_SIZE;
    (0..=last).contains(&offset)
}

pub(crate) struct ModernHDMapGeometry {
    pub(crate) terrain: Vec<ModernHDTerrainDraw>,
    pub(crate) units: Vec<ModernHDMapUnitDraw>,
    pub(crate) foreground: Vec<ModernHDMapForegroundDraw>,
}

/// 地形使用範圍模式改寫前的格子；前景沿原生合成後的格子重畫。
pub(crate) fn prepare_modern_hd_map_geometry(
    before: &indexedmap::FrameInput,
    after: &indexedmap::FrameInput,
    layout: (i64, i64, i64, i64),
    palette: &[ColorRgba],
    sprites: &BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    terrain_source: Option<(&TerrainMap, &TerrainColourContext)>,
) -> Result<ModernHDMapGeometry> {
    let viewport_base = layout.3;
    let units = build_modern_hd_map_unit_plan(before, viewport_base, sprites)?;
    let mut foreground = build_modern_hd_map_foreground_plan(after, viewport_base, palette)?;
    let mut terrain = Vec::new();
    if let Some((map, colours)) = terrain_source {
        terrain = build_modern_hd_terrain_plan(before, layout, map, colours)?;
        for command in &mut foreground {
            let Ok(id) = u16::try_from(command.source_frame) else { continue };
            if map.frames.get(&id).is_some_and(|frame| frame.role.supports(TerrainRole::Foreground)) {
                command.hd_image = map.shade(id, colours, command.blit_mode == 0xff)?;
            }
        }
    }
    Ok(ModernHDMapGeometry { terrain, units, foreground })
}

/// A frame is prepared inside the original timing transaction and published
/// only with its native work/VGA/state. TerrainSeed never contains objects/UI.
///
/// Go：`modern_hd_map_runtime.go:18`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapFrame {
    pub(crate) units: Vec<ModernHDMapUnitDraw>,
    pub(crate) foreground: Vec<ModernHDMapForegroundDraw>,
    pub(crate) terrain_seed: Vec<u8>,
    pub(crate) terrain: Vec<ModernHDTerrainDraw>,
    pub(crate) overlays: Vec<ModernHDMapOverlayDraw>,
}

/// Go：`modern_hd_map_runtime.go:40`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapOverlayDraw {
    pub(crate) entry: i64,
    pub(crate) image: Option<Rgba>,
    pub(crate) rect: Rectangle,
}

#[cfg(test)]
#[path = "modern_hd_map_runtime_test.rs"]
mod tests;
