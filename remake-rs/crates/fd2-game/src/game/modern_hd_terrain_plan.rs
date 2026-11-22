//! 依原生地形工作位址投影 HD 替換畫格，對應 Go `modern_hd_terrain_plan.go`。

use super::modern_hd_terrain::{NATIVE_SIDE, SCALE, TerrainColourContext, TerrainMap, TerrainRole};
use super::*;

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct ModernHDTerrainDraw {
    pub(crate) image: Arc<Picture>,
    pub(crate) rect: Rectangle,
    pub(crate) frame: i64,
    pub(crate) work_offset: i64,
}

pub(crate) fn modern_hd_map_viewport() -> Rectangle {
    let scale = SCALE as i64;
    rect(
        indexedmap::NATIVE_MAP_VIEWPORT_X0 * scale,
        indexedmap::NATIVE_MAP_VIEWPORT_Y0 * scale,
        (indexedmap::NATIVE_MAP_VIEWPORT_X1 + 1) * scale,
        (indexedmap::NATIVE_MAP_VIEWPORT_Y1 + 1) * scale,
    )
}

pub(crate) struct TerrainRegion {
    pub(crate) terrain_base: i64,
    pub(crate) camera_y: i64,
    pub(crate) cols: i64,
    pub(crate) rows: i64,
    pub(crate) viewport_base: i64,
}

/// Layout is the native work address tuple also returned by `native_up_step_layout`.
pub(crate) fn build_modern_hd_terrain_plan(
    input: &indexedmap::FrameInput,
    (terrain_base, camera_y, rows, viewport_base): (i64, i64, i64, i64),
    map: &TerrainMap,
    colours: &TerrainColourContext,
) -> Result<Vec<ModernHDTerrainDraw>> {
    let view = modern_hd_map_viewport();
    let cols = (view.max.x - view.min.x) / (SCALE * NATIVE_SIDE) as i64;
    let region = TerrainRegion { terrain_base, camera_y, cols, rows, viewport_base };
    build_modern_hd_terrain_region_plan(input, region, view, map, colours)
}

pub(crate) fn build_modern_hd_terrain_region_plan(
    input: &indexedmap::FrameInput,
    region: TerrainRegion,
    clip: Rectangle,
    map: &TerrainMap,
    colours: &TerrainColourContext,
) -> Result<Vec<ModernHDTerrainDraw>> {
    if map.frames.is_empty() {
        return Ok(Vec::new());
    }
    let TerrainRegion { terrain_base, camera_y, cols, rows, viewport_base } = region;
    let width = input.map_width;
    let count = input.cells.len() as i64;
    let limit = indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE;
    if width <= 0
        || count % width != 0
        || input.camera_x < 0
        || camera_y < 0
        || cols <= 0
        || rows <= 0
        || cols > width - input.camera_x
        || rows > count / width - camera_y
        || !(0..limit).contains(&terrain_base)
        || !(0..limit).contains(&viewport_base)
    {
        return Err(game_err!("modern HD terrain: invalid native region"));
    }
    let stride = fdicon::NATIVE_MAP_STRIDE;
    let side = NATIVE_SIDE as i64;
    let mut draws = Vec::new();
    for row in 0..rows {
        for col in 0..cols {
            let cell = input.cells[((camera_y + row) * width + input.camera_x + col) as usize];
            let tile = i64::from(cell.tile & 0x3ff);
            let flags = *input
                .controls
                .get((tile * 4) as usize)
                .ok_or_else(|| game_err!("modern HD terrain: control record missing"))?;
            let frame = fdicon::native_terrain_frame_index(tile, flags, input.flip, input.terrain_cycle)
                .map_err(|err| game_err!("{err}"))?;
            let Ok(id) = u16::try_from(frame) else { continue };
            if !map.frames.get(&id).is_some_and(|asset| asset.role.supports(TerrainRole::Terrain)) {
                continue;
            }
            let work_offset = row
                .checked_mul(side * stride)
                .and_then(|offset| offset.checked_add(terrain_base))
                .and_then(|offset| col.checked_mul(side).and_then(|x| offset.checked_add(x)))
                .filter(|&offset| hd_map_tile_fits_work(offset))
                .ok_or_else(|| game_err!("modern HD terrain: work destination out of bounds"))?;
            let area = hd_map_tile_rect(work_offset, viewport_base);
            if rect_empty(&rect_intersect(area, clip)) {
                continue;
            }
            if let Some(image) = map.shade(id, colours, cell.blit_mode == 0xff)? {
                draws.push(ModernHDTerrainDraw { image, rect: area, frame, work_offset });
            }
        }
    }
    Ok(draws)
}
