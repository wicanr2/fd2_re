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
    pub(crate) background: Option<Arc<Picture>>,
    pub(crate) ui: Option<Arc<Picture>>,
    pub(crate) range: Option<Arc<Picture>>,
    pub(crate) units: Vec<ModernHDMapUnitDraw>,
    pub(crate) foreground: Vec<ModernHDMapForegroundDraw>,
    pub(crate) terrain_seed: Vec<u8>,
    pub(crate) terrain: Vec<ModernHDTerrainDraw>,
    pub(crate) overlays: Vec<ModernHDMapOverlayDraw>,
    pub(crate) viewport: Rectangle,
    pub(crate) overlays_above_ui: bool,
    pub(crate) hud: Vec<ModernHDMapOverlayDraw>,
    pub(crate) palette_add6: u8,
}

/// Go：`modern_hd_map_runtime.go:40`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapOverlayDraw {
    pub(crate) entry: i64,
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) rect: Rectangle,
    pub(crate) scale_to_rect: bool,
}

#[derive(Clone, Copy)]
pub(crate) enum ModernHDMapMotion {
    Steady,
    Step,
    North { unit_y: i64, phase: i64 },
}

/// Probes the exact same native blitter with two backgrounds to retain writes of palette index zero.
fn overwrite_layer(
    size: usize,
    palette: &[ColorRgba],
    mut render: impl FnMut(&mut [u8]) -> Result<()>,
) -> Result<Vec<[u8; 4]>> {
    if size == 0 || palette.len() != 256 {
        return Err(game_err!("modern HD map: incomplete overwrite input"));
    }
    let mut first = vec![0; size];
    let mut second = vec![255; size];
    render(&mut first)?;
    render(&mut second)?;
    first
        .into_iter()
        .zip(second)
        .enumerate()
        .map(|(at, (a, b))| match (a, b) {
            (a, b) if a == b => {
                let c = palette[a as usize];
                Ok([c.r, c.g, c.b, 255])
            }
            (0, 255) => Ok([0; 4]),
            _ => Err(game_err!("modern HD map: destination-dependent write at {at}")),
        })
        .collect()
}

fn viewport_picture(mut source: impl FnMut(usize, usize) -> [u8; 4]) -> Picture {
    let mut pixels = vec![[0; 4]; 320 * 200];
    for y in 4..196 {
        for x in 4..316 {
            pixels[y * 320 + x] = source(x, y);
        }
    }
    let mut picture = Picture::new_image(320, 200);
    picture.write_pixels(pixels.as_flattened());
    picture
}

pub(crate) fn prepare_modern_hd_map_frame(
    input: &indexedmap::NativeFrameInput,
    seed: &[u8],
    palette: &[ColorRgba],
    sprites: &BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    terrain_source: Option<(&TerrainMap, &TerrainColourContext)>,
    motion: ModernHDMapMotion,
    banner: Option<&dyn Fn(&mut [u8]) -> bool>,
) -> Result<ModernHDMapFrame> {
    use ModernHDMapMotion::*;
    let work_size = indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize;
    if seed.len() != work_size || palette.len() != 256 {
        return Err(game_err!("modern HD map: incomplete seed or palette"));
    }
    let before = &input.frame;
    let mut background = before.clone();
    background.units.clear();
    background.foreground_units.clear();
    background.selection_overlay = None;
    let layout = if let North { unit_y, phase } = motion {
        indexedmap::native_up_step_layout(&background, unit_y, phase)?
    } else {
        (NATIVE_MAP_STEADY_BASE, before.camera_y, 8, NATIVE_MAP_STEADY_BASE)
    };
    let mut work = seed.to_vec();
    let mut vga = vec![0; 320 * 200];
    let mut terrain_seed = Vec::new();
    let mut observe = |stage, work: &mut [u8], _: &mut [u8]| {
        if stage == indexedmap::FRAME_STAGE_TERRAIN {
            terrain_seed.extend_from_slice(work);
        }
        Ok(())
    };
    match motion {
        Steady => indexedmap::compose_frame_observed(
            &mut work,
            &mut vga,
            &mut background,
            Some(&mut |_| Ok(())),
            Some(&mut observe),
        )?,
        Step => indexedmap::compose_native_step_frame_observed(&mut work, &mut vga, &background, Some(&mut observe))?,
        North { unit_y, phase } => indexedmap::compose_native_up_step_frame_observed(
            &mut work,
            &mut vga,
            &background,
            unit_y,
            phase,
            Some(&mut observe),
        )?,
    }
    if terrain_seed.len() != work_size {
        return Err(game_err!("modern HD map: missing terrain pass"));
    }
    let mut after = before.clone();
    after.cells = background.cells;
    let geometry = prepare_modern_hd_map_geometry(before, &after, layout, palette, sprites, terrain_source)?;
    let viewport_base = layout.3 as usize;
    let stride = fdicon::NATIVE_MAP_STRIDE as usize;
    let project = |source: &[[u8; 4]]| viewport_picture(|x, y| source[viewport_base + (y - 4) * stride + (x - 4)]);
    let range = if matches!(motion, Steady) && !geometry.terrain.is_empty() && !matches!(before.range_mode, 0 | 6) {
        let bank = before.range_bank.ok_or_else(|| game_err!("modern HD map: missing range bank"))?;
        let pixels = overwrite_layer(work_size, palette, |dst| {
            fdother::blit_native_range_overlay(
                bank,
                dst,
                before.camera_x,
                before.camera_y,
                13,
                8,
                before.range_mode,
                before.cursor_x,
                before.cursor_y,
            )?;
            Ok(())
        })?;
        Some(Arc::new(project(&pixels)))
    } else {
        None
    };
    let mut ui = if matches!(motion, Steady) {
        let pixels = overwrite_layer(work_size, palette, |dst| {
            if let Some(overlay) = before.selection_overlay {
                overlay(dst, fdicon::NATIVE_MAP_STRIDE, NATIVE_MAP_STEADY_BASE)?;
            }
            indexedmap::blit_native_map_hud(
                &input.frames,
                input.hud_terrain,
                input.hud_units,
                input.hud_cache,
                &mut dst[NATIVE_MAP_STEADY_BASE as usize..],
                &input.hud,
            )?;
            Ok(())
        })?;
        project(&pixels)
    } else {
        Picture::new_image(320, 200)
    };
    if let Some(banner) = banner {
        let pixels = overwrite_layer(320 * 200, palette, |dst| {
            if !banner(dst) {
                return Err(game_err!("modern HD map: invalid phase banner"));
            }
            Ok(())
        })?;
        let mut glyphs = Picture::new_image(320, 200);
        glyphs.write_pixels(pixels.as_flattened());
        ui.draw_image(&glyphs, &DrawImageOptions::default());
    }
    let mut background = viewport_picture(|x, y| {
        let c = palette[vga[y * 320 + x] as usize];
        [c.r, c.g, c.b, 255]
    });
    // The Go background has an opaque black border. Preserve the viewport colours.
    let mut pixels = background.read_pixels().to_vec();
    for rgba in pixels.as_chunks_mut::<4>().0 {
        rgba[3] = 255;
    }
    background.write_pixels(&pixels);
    Ok(ModernHDMapFrame {
        background: Some(Arc::new(background)),
        ui: Some(Arc::new(ui)),
        range,
        terrain: geometry.terrain,
        units: geometry.units,
        foreground: geometry.foreground,
        terrain_seed,
        ..Default::default()
    })
}

impl Game {
    pub(crate) fn modern_hd_map_frame_admission(&self, legacy_viewport: bool, battle_view: bool) -> bool {
        !legacy_viewport
            && battle_view
            && self.modern_story_portraits.as_ref().is_some_and(|set| set.map_sprites_hd.len() == 125)
            && native_map_assets_available(self.native_map_assets.as_ref())
            && self.st.as_ref().is_some_and(|state| state.has_native_map_view_state)
            && !self.spell_open
            && !self.item_open
            && self.cast_sp.is_none()
            && (self.sel.is_none()
                || self.native_command0_targeting
                || self.native_item_targeting
                || self.native_item_relocating
                || self.ring
                || self.native_command_open
                || self.native_move_plan.is_some())
    }

    pub(crate) fn prepare_hd_battle_map(
        &self,
        input: &indexedmap::NativeFrameInput,
        stage: Option<&NativeCh23AdapterState>,
        dac: &[u8],
        motion: ModernHDMapMotion,
    ) -> Result<ModernHDMapFrame> {
        let theme = self.modern_story_portraits.as_ref().ok_or_else(|| game_err!("modern HD map: missing theme"))?;
        let assets =
            self.native_map_assets.as_ref().ok_or_else(|| game_err!("modern HD map: missing native assets"))?;
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(dac)?);
        let tiles = match (&theme.terrain_hd, &assets.terrain, u8::try_from(assets.map_index)) {
            (Some(pack), Some(bank), Ok(id)) => pack.map(id, bank)?,
            _ => None,
        };
        let colours = if tiles.is_some() {
            Some(TerrainColourContext::new(assets.map_index as u8, &assets.palette, &palette, input.frame.lut)?)
        } else {
            None
        };
        let tiles = tiles.as_deref().zip(colours.as_ref());
        let size = indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize;
        let mut seed =
            if self.modern_hd_map_terrain.len() == size { self.modern_hd_map_terrain.clone() } else { vec![0; size] };
        let stage = stage.or(if assets.map_index == 23 { self.native_ch23_state.as_ref() } else { None });
        if let Some(stage) = stage {
            indexedmap::seed_native_ch23_staging(&mut seed, &stage.staging)?;
        }
        let (offset, visible) = self.phase_banner_slide_offset();
        let glyphs = |dst: &mut [u8]| self.blit_phase_banner_glyphs(dst, offset);
        let glyphs = (visible && self.phase_banner_glyphs_available()).then_some(&glyphs as &dyn Fn(&mut [u8]) -> bool);
        prepare_modern_hd_map_frame(input, &seed, &palette, &theme.map_sprites_hd, tiles, motion, glyphs)
    }
}

#[cfg(test)]
#[path = "modern_hd_map_runtime_test.rs"]
mod tests;
