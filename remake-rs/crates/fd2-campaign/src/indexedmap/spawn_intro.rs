//! 移植自 Go `indexedmap/spawn_intro.go`。

use super::frame::WORK_STRIDE;
use super::{
    NATIVE_MAP_VGA_SIZE, NATIVE_UNIT_PRESENT_WORK_SIZE, NativeTransitionFrameInput, Result, at,
    compose_native_unit_present_terrain_snapshot, copy_native_unit_present_viewport, err, len64,
    redraw_native_unit_present_objects, wrap,
};
use fd2_formats::fdother::{
    self, LMI1Entry, NATIVE_SPAWN_INTRO_FULL_FRAME, NATIVE_SPAWN_INTRO_KEEP_SNAPSHOT, NATIVE_SPAWN_INTRO_SPLIT_UNITS,
    NativeSpawnIntroStep,
};

/// NativeSpawnIntroFrameInput binds the complete post-constructor roster to
/// the old/new slot boundary saved by 0x32999 before its 0x10B4E call.
///
/// Go：`spawn_intro.go:12`
#[derive(Debug, Clone, Default)]
pub struct NativeSpawnIntroFrameInput<'a> {
    pub frame: NativeTransitionFrameInput<'a>,
    pub old_unit_count: i64,
}

/// Go：`spawn_intro.go:17`
fn validate_native_spawn_intro_frame_input(in_: &NativeSpawnIntroFrameInput) -> Result<()> {
    let f = &in_.frame;
    if in_.old_unit_count < 0
        || in_.old_unit_count > len64(&f.units)
        || f.units.len() != f.foreground_units.len()
        || f.unit_bank.is_none()
        || f.foreground_bank.is_none()
        || f.selector_cache.is_none()
    {
        return Err(err("indexedmap: incomplete native spawn-intro roster boundary"));
    }
    Ok(())
}

/// Go：`spawn_intro.go:27`。呼叫端都先通過 `validate_native_spawn_intro_frame_input`，三個資源都存在。
fn redraw_native_spawn_intro_split(
    dst: &mut [u8],
    in_: &NativeSpawnIntroFrameInput,
    new_unit_pixel_lift: i64,
) -> Result<()> {
    if len64(dst) != NATIVE_UNIT_PRESENT_WORK_SIZE || new_unit_pixel_lift > 0 {
        return Err(err("indexedmap: invalid native spawn-intro split target"));
    }
    let f = &in_.frame;
    let (unit_bank, foreground_bank, cache) =
        (f.unit_bank.expect("已驗證"), f.foreground_bank.expect("已驗證"), f.selector_cache.expect("已驗證"));
    let old = at(in_.old_unit_count);
    let mut frame = dst.to_vec();
    unit_bank
        .blit_native_unit_layer(
            &mut frame,
            WORK_STRIDE,
            cache,
            &f.units[..old],
            f.camera_x,
            f.camera_y,
            12,
            7,
            f.idle_cycle,
            f.moving_cycle,
            f.pixel_shift,
        )
        .map_err(wrap("indexedmap: spawn-intro old units: "))?;
    unit_bank
        .blit_native_unit_layer_with_base_shift(
            &mut frame,
            WORK_STRIDE,
            cache,
            &f.units[old..],
            f.camera_x,
            f.camera_y,
            12,
            7,
            f.idle_cycle,
            f.moving_cycle,
            f.pixel_shift,
            new_unit_pixel_lift.wrapping_mul(WORK_STRIDE),
        )
        .map_err(wrap("indexedmap: spawn-intro new units: "))?;
    foreground_bank
        .blit_native_foreground_layer(
            &mut frame,
            WORK_STRIDE,
            &f.foreground_units,
            f.map_width,
            &f.cells,
            f.controls,
            f.camera_x,
            f.camera_y,
            12,
            7,
            f.flip,
            f.terrain_lut,
        )
        .map_err(wrap("indexedmap: spawn-intro foreground: "))?;
    dst.copy_from_slice(&frame);
    Ok(())
}

/// ComposeNativeSpawnIntroPass executes one complete 0x32999 iteration. The
/// visible VGA frame is captured before pass 6/7/8 rebuild the private snapshot,
/// matching the original call order. All three caller buffers remain unchanged
/// if any asset, roster selector, geometry, or layer operation fails.
///
/// Go：`spawn_intro.go:70`
pub fn compose_native_spawn_intro_pass(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &mut [u8],
    in_: &NativeSpawnIntroFrameInput,
    entry: &LMI1Entry,
    step: NativeSpawnIntroStep,
) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) < NATIVE_MAP_VGA_SIZE
    {
        return Err(err("indexedmap: incomplete native spawn-intro buffers"));
    }
    validate_native_spawn_intro_frame_input(in_)?;
    let schedule = fdother::native_spawn_intro_schedule();
    if step.pass < 0 || step.pass >= len64(&schedule) || step != schedule[at(step.pass)] {
        return Err(err("indexedmap: native spawn-intro step differs from 0x32999"));
    }

    let mut frame = snapshot.to_vec();
    let f = &in_.frame;
    for unit in &f.units[at(in_.old_unit_count)..] {
        if !fdother::native_spawn_intro_visible(unit.x, unit.y, f.camera_x, f.camera_y, 12, 7) {
            continue;
        }
        fdother::blit_native_spawn_intro_lmi(entry, &mut frame, unit.x, unit.y, f.camera_x, f.camera_y)
            .map_err(wrap(format!("indexedmap: spawn-intro pass {} LMI: ", step.pass)))?;
    }
    let mut viewport = vga.to_vec();
    copy_native_unit_present_viewport(&mut viewport, &frame)
        .map_err(wrap(format!("indexedmap: spawn-intro pass {} present: ", step.pass)))?;

    let mut next_snapshot = snapshot.to_vec();
    match step.snapshot_mode {
        NATIVE_SPAWN_INTRO_KEEP_SNAPSHOT => {
            // The next pass restores the unchanged pre-existing snapshot.
        }
        NATIVE_SPAWN_INTRO_SPLIT_UNITS => {
            if step.redraw_terrain {
                compose_native_unit_present_terrain_snapshot(&mut next_snapshot, f)
                    .map_err(wrap(format!("indexedmap: spawn-intro pass {} terrain: ", step.pass)))?;
            }
            redraw_native_spawn_intro_split(&mut next_snapshot, in_, step.new_unit_pixel_lift)
                .map_err(wrap(format!("indexedmap: spawn-intro pass {} split: ", step.pass)))?;
        }
        NATIVE_SPAWN_INTRO_FULL_FRAME => {
            if !step.redraw_terrain || step.new_unit_pixel_lift != 0 {
                return Err(err("indexedmap: invalid full spawn-intro snapshot step"));
            }
            compose_native_unit_present_terrain_snapshot(&mut next_snapshot, f)
                .map_err(wrap(format!("indexedmap: spawn-intro pass {} terrain: ", step.pass)))?;
            redraw_native_unit_present_objects(&mut next_snapshot, f)
                .map_err(wrap(format!("indexedmap: spawn-intro pass {} objects: ", step.pass)))?;
        }
        _ => return Err(err("indexedmap: unknown native spawn-intro snapshot mode")),
    }

    vga.copy_from_slice(&viewport);
    if step.snapshot_mode == NATIVE_SPAWN_INTRO_KEEP_SNAPSHOT {
        work.copy_from_slice(&frame);
    } else {
        snapshot.copy_from_slice(&next_snapshot);
        work.copy_from_slice(&next_snapshot);
    }
    Ok(())
}

/// SeedNativeSpawnIntroSnapshot captures the exact pre-constructor work buffer.
/// It is deliberately a full 0x25680-byte copy, not a cropped visible frame.
///
/// Go：`spawn_intro.go:140`
pub fn seed_native_spawn_intro_snapshot(work: &[u8]) -> Result<Vec<u8>> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE {
        return Err(err("indexedmap: native spawn-intro seed must be exactly 0x25680 bytes"));
    }
    Ok(work.to_vec())
}

#[cfg(test)]
mod spawn_intro_test;
