//! Go 來源：`remake/cmd/fd2/modern_hd_map_unit_plan.go`（89 行）。

use super::*;

/// Go：`modern_hd_map_unit_plan.go:12`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapUnitDraw {
    pub(crate) group: i64,
    pub(crate) frame: i64,
    pub(crate) rect: Rectangle,
    pub(crate) work_offset: i64,
    pub(crate) palette_band: bool,
}

pub(crate) fn build_modern_hd_map_unit_plan(
    input: &indexedmap::FrameInput,
    viewport_base: i64,
    sprites: &BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
) -> Result<Vec<ModernHDMapUnitDraw>> {
    build_modern_hd_map_unit_region_plan(input, viewport_base, sprites, 12, 8)
}

pub(crate) fn build_modern_hd_map_unit_region_plan(
    input: &indexedmap::FrameInput,
    viewport_base: i64,
    sprites: &BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    xmax: i64,
    ymax: i64,
) -> Result<Vec<ModernHDMapUnitDraw>> {
    let (Some(bank), Some(cache)) = (input.unit_bank, input.selector_cache) else {
        return Err(game_err!("modern HD units: incomplete native input"));
    };
    if !hd_map_viewport_fits_work(viewport_base) {
        return Err(game_err!("modern HD units: invalid viewport"));
    }
    let x_range = input.camera_x.saturating_sub(1)..=input.camera_x.saturating_add(xmax);
    let y_range = input.camera_y.saturating_sub(1)..=input.camera_y.saturating_add(ymax);
    let area = (fdicon::NATIVE_SIZE * fdicon::NATIVE_SIZE) as usize;
    let side = fdicon::NATIVE_SIZE * super::modern_hd_terrain::SCALE as i64;
    let mut commands = Vec::with_capacity(input.units.len());
    for unit in &input.units {
        if unit.inactive || !x_range.contains(&unit.x) || !y_range.contains(&unit.y) {
            continue;
        }
        let cycle =
            fdicon::native_frame_index(unit.motion_offset, unit.force_base, input.idle_cycle, input.moving_cycle)
                .map_err(|err| game_err!("{err}"))?;
        let group = cache.key_for_slot(unit.slot).map_err(|err| game_err!("{err}"))?;
        let sprite = bank.sprite_for(group, unit.pose, cycle).map_err(|err| game_err!("{err}"))?;
        let work_offset = fdicon::native_placement_offset(
            unit.x,
            unit.y,
            input.camera_x,
            input.camera_y,
            unit.pose,
            unit.motion_offset,
            input.pixel_shift,
            unit.force_base,
        )
        .map_err(|err| game_err!("{err}"))?;
        if work_offset < 0 {
            continue;
        }
        if !hd_map_tile_fits_work(work_offset) {
            return Err(game_err!("modern HD units: native unit exceeds work buffer"));
        }
        if sprite.pixels.len() != area || sprite.mask.len() != area {
            return Err(game_err!("modern HD units: incomplete original sprite"));
        }
        if !sprite.mask.iter().any(|&value| value != 0) {
            continue;
        }
        let frame = unit.pose * 3 + cycle;
        let picture = sprites.get(&group).and_then(|frames| frames.get(frame as usize)).and_then(Option::as_ref);
        if !picture.is_some_and(|picture| (picture.width, picture.height) == (side, side)) {
            return Err(game_err!("modern HD units: invalid group {group} frame {frame}"));
        }
        commands.push(ModernHDMapUnitDraw {
            group,
            frame,
            rect: hd_map_tile_rect(work_offset, viewport_base),
            work_offset,
            palette_band: unit.flags & 0x80 != 0,
        });
    }
    Ok(commands)
}
