//! Go 來源：`remake/cmd/fd2/modern_hd_map_foreground_plan.go`（92 行）。

use super::*;

/// Go：`modern_hd_map_foreground_plan.go:12`
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct ModernHDMapForegroundDraw {
    pub(crate) image: Arc<Picture>,
    pub(crate) rect: Rectangle,
    pub(crate) source_frame: i64,
    pub(crate) work_offset: i64,
    pub(crate) blit_mode: u8,
    pub(crate) hd_image: Option<Arc<Picture>>,
}

pub(crate) fn build_modern_hd_map_foreground_plan(
    input: &indexedmap::FrameInput,
    viewport_base: i64,
    palette: &[ColorRgba],
) -> Result<Vec<ModernHDMapForegroundDraw>> {
    build_modern_hd_map_foreground_region_plan(input, viewport_base, palette, 12, 8)
}

pub(crate) fn build_modern_hd_map_foreground_region_plan(
    input: &indexedmap::FrameInput,
    viewport_base: i64,
    palette: &[ColorRgba],
    xmax: i64,
    ymax: i64,
) -> Result<Vec<ModernHDMapForegroundDraw>> {
    let Some(bank) = input.foreground_bank else {
        return Err(game_err!("modern HD foreground: missing native bank"));
    };
    let width = input.map_width;
    let count = input.cells.len() as i64;
    if width <= 0
        || count % width != 0
        || !(0..=1).contains(&input.flip)
        || palette.len() != 256
        || !hd_map_viewport_fits_work(viewport_base)
    {
        return Err(game_err!("modern HD foreground: incomplete native input or viewport"));
    }
    let x_range = input.camera_x.saturating_sub(1).max(0)..=input.camera_x.saturating_add(xmax).min(width - 1);
    let y_range = input.camera_y.saturating_sub(1).max(0)..=input.camera_y.saturating_add(ymax).min(count / width - 1);
    let stride = fdicon::NATIVE_MAP_STRIDE;
    let side = fdicon::NATIVE_SIZE;
    let area = (side * side) as usize;
    let mut commands = Vec::new();
    let mut pictures = BTreeMap::new();
    for unit in &input.foreground_units {
        if !fdicon::native_foreground_redraw_eligible(unit.inactive, unit.unit7, unit.race, unit.class) {
            continue;
        }
        let (coords, len) = fdicon::native_foreground_redraw_cells(unit.x, unit.y, unit.pose, unit.motion_offset);
        for coord in &coords[..len] {
            if !x_range.contains(&coord.x) || !y_range.contains(&coord.y) {
                continue;
            }
            let cell = input.cells[(coord.y * width + coord.x) as usize];
            let tile = i64::from(cell.tile & 0x3ff);
            let flags = *input
                .controls
                .get((tile * 4) as usize)
                .ok_or_else(|| game_err!("modern HD foreground: incomplete native terrain controls"))?;
            let (source_frame, present) =
                fdicon::native_foreground_frame_index(tile, flags, input.flip).map_err(|err| game_err!("{err}"))?;
            if !present {
                continue;
            }
            let sprite = bank
                .sprite_for(source_frame / 12, (source_frame % 12) / 3, source_frame % 3)
                .map_err(|err| game_err!("{err}"))?;
            let raw = cell.blit_mode == 0xff;
            if sprite.pixels.len() != area || sprite.mask.len() != area || (!raw && input.lut.len() != 256) {
                return Err(game_err!("modern HD foreground: incomplete source or LUT"));
            }
            let work_offset = coord
                .y
                .checked_sub(input.camera_y)
                .and_then(|dy| dy.checked_mul(side * stride))
                .and_then(|row| {
                    let col = coord.x.checked_sub(input.camera_x)?.checked_mul(side)?;
                    NATIVE_MAP_STEADY_BASE.checked_add(row)?.checked_add(col)
                })
                .filter(|&offset| hd_map_tile_fits_work(offset))
                .ok_or_else(|| game_err!("modern HD foreground: native destination exceeds work buffer"))?;
            let image = pictures.entry((source_frame, raw)).or_insert_with(|| {
                let mut pixels = [0; (fdicon::NATIVE_SIZE * fdicon::NATIVE_SIZE * 4) as usize];
                for ((&index, &mask), pixel) in
                    sprite.pixels.iter().zip(&sprite.mask).zip(pixels.as_chunks_mut::<4>().0)
                {
                    if mask != 0 {
                        let index = if raw { index } else { input.lut[index as usize] };
                        let colour = palette[index as usize];
                        *pixel = [colour.r, colour.g, colour.b, 255];
                    }
                }
                let mut image = Picture::new_image(side, side);
                image.write_pixels(&pixels);
                Arc::new(image)
            });
            commands.push(ModernHDMapForegroundDraw {
                image: image.clone(),
                rect: hd_map_tile_rect(work_offset, viewport_base),
                source_frame,
                work_offset,
                blit_mode: cell.blit_mode,
                hd_image: None,
            });
        }
    }
    Ok(commands)
}
