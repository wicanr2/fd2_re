//! 移植自 Go `indexedmap/native_command_heal_tail.go`。

use super::frame::{WORK_BASE, WORK_STRIDE};
use super::{
    NATIVE_MAP_VGA_SIZE, NATIVE_UNIT_PRESENT_WORK_SIZE, Result, at, copy_native_unit_present_viewport, err, go_sharp_x,
    len64, wrap,
};
use fd2_formats::fdicon::{Bank, NativeSelectorCache};
use fd2_formats::fdother::LMI1Entry;
use fd2_rules::battle::NativePresentationDigit;
use std::collections::BTreeMap;

/// NativeCommandHealTailTarget preserves the runtime-record selector and map
/// bytes consumed by 0x1c4cc, 0x1c2da and 0x1df58.
///
/// Go：`native_command_heal_tail.go:14`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandHealTailTarget {
    pub record_index: i64,
    pub x: i64,
    pub y: i64,
    pub selector_slot: i64,
}

/// NativeCommandTailLayer records the exact successful LMI write. X/Y are
/// relative to the copied work viewport, before its VGA destination is chosen.
///
/// Go：`native_command_heal_tail.go:20`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandTailLayer {
    pub resource: i64,
    pub entry: i64,
    pub x: i64,
    pub y: i64,
}

/// Go：`native_command_heal_tail.go:24`
fn native_command_tail_layer(resource: i64, index: i64, x: i64, y: i64) -> NativeCommandTailLayer {
    NativeCommandTailLayer { resource, entry: index, x: x - WORK_BASE % WORK_STRIDE, y: y - WORK_BASE / WORK_STRIDE }
}

/// Go：`native_command_heal_tail.go:29`，回傳 `(x, y int, err error)`。
fn native_command_tail_origin(
    x: i64,
    y: i64,
    camera_x: i64,
    camera_y: i64,
    extra_x: i64,
    extra_rows: i64,
) -> Result<(i64, i64)> {
    let offset = WORK_BASE
        .wrapping_add(24i64.wrapping_mul(x.wrapping_sub(camera_x)))
        .wrapping_add((24 * WORK_STRIDE).wrapping_mul(y.wrapping_sub(camera_y)))
        .wrapping_add(extra_x)
        .wrapping_add(extra_rows.wrapping_mul(WORK_STRIDE));
    if offset < 0 {
        return Err(err("indexedmap: native command tail origin before work buffer"));
    }
    Ok((offset % WORK_STRIDE, offset / WORK_STRIDE))
}

/// ComposeNativeCommandHealEffectFrame reproduces one 0x1c4cc descriptor pass.
/// The caller supplies the exact steady-frame snapshot and visible final target
/// array; rejected input leaves both work and VGA unchanged.
///
/// Go：`native_command_heal_tail.go:39`
#[allow(clippy::too_many_arguments)]
pub fn compose_native_command_heal_effect_frame(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    entry: &LMI1Entry,
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
) -> Result<()> {
    compose_native_command_heal_effect_frame_inner(work, vga, snapshot, entry, targets, camera_x, camera_y, -1, -1)
        .map(|_| ())
}

/// Go：`native_command_heal_tail.go:44`
#[allow(clippy::too_many_arguments)]
pub fn compose_native_command_heal_effect_frame_with_sources(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    entry: &LMI1Entry,
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
    resource: i64,
    index: i64,
) -> Result<Vec<NativeCommandTailLayer>> {
    if resource < 0 || index < 0 {
        return Err(err("indexedmap: command tail source unavailable"));
    }
    compose_native_command_heal_effect_frame_inner(
        work, vga, snapshot, entry, targets, camera_x, camera_y, resource, index,
    )
}

/// Go：`native_command_heal_tail.go:51`
#[allow(clippy::too_many_arguments)]
fn compose_native_command_heal_effect_frame_inner(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    entry: &LMI1Entry,
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
    resource: i64,
    index: i64,
) -> Result<Vec<NativeCommandTailLayer>> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) != NATIVE_MAP_VGA_SIZE
    {
        return Err(err("indexedmap: incomplete native command heal effect buffers"));
    }
    let mut frame = snapshot.to_vec();
    let mut layers = Vec::with_capacity(targets.len());
    for target in targets {
        let (x, y) = native_command_tail_origin(target.x, target.y, camera_x, camera_y, 0, -6)?;
        entry
            .blit_at(&mut frame, WORK_STRIDE, x, y, false)
            .map_err(wrap(format!("indexedmap: command heal effect target {}: ", target.record_index)))?;
        layers.push(native_command_tail_layer(resource, index, x, y));
    }
    let mut viewport = vga.to_vec();
    copy_native_unit_present_viewport(&mut viewport, &frame)?;
    work.copy_from_slice(&frame);
    vga.copy_from_slice(&viewport);
    Ok(layers)
}

/// ComposeNativeCommandHealMaskFrame reproduces 0x1c2da's 0x4ddd7 calls.
///
/// Go：`native_command_heal_tail.go:76`
#[allow(clippy::too_many_arguments)]
pub fn compose_native_command_heal_mask_frame(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    units: Option<&Bank>,
    cache: Option<&NativeSelectorCache>,
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
    cycle: i64,
    raw_index: u8,
) -> Result<()> {
    let (Some(units), Some(cache)) = (units, cache) else {
        return Err(err("indexedmap: incomplete native command heal mask input"));
    };
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) != NATIVE_MAP_VGA_SIZE
        || !(0..=3).contains(&cycle)
    {
        return Err(err("indexedmap: incomplete native command heal mask input"));
    }
    let cycle = if cycle == 3 { 2 } else { cycle };
    let mut frame = snapshot.to_vec();
    for target in targets {
        let sprite = units
            .sprite_for_native_slot(cache, target.selector_slot, 0, cycle)
            .map_err(wrap(format!("indexedmap: command heal mask target {} selector: ", target.record_index)))?;
        let (x, y) = native_command_tail_origin(target.x, target.y, camera_x, camera_y, 0, -6)?;
        sprite
            .blit_constant_mask_at(&mut frame, WORK_STRIDE, x, y, raw_index)
            .map_err(wrap(format!("indexedmap: command heal mask target {}: ", target.record_index)))?;
    }
    let mut viewport = vga.to_vec();
    copy_native_unit_present_viewport(&mut viewport, &frame)?;
    work.copy_from_slice(&frame);
    vga.copy_from_slice(&viewport);
    Ok(())
}

/// ComposeNativeAIItemDamageBlendFrame reproduces one 0x1cd17 snapshot frame.
///
/// Go：`native_command_heal_tail.go:105`
#[allow(clippy::too_many_arguments)]
pub fn compose_native_ai_item_damage_blend_frame(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    units: Option<&Bank>,
    cache: Option<&NativeSelectorCache>,
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
    idle_cycle: i64,
    blend: i64,
    raw_base: u8,
) -> Result<()> {
    let (Some(units), Some(cache)) = (units, cache) else {
        return Err(err("indexedmap: incomplete native AI item damage blend input"));
    };
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) != NATIVE_MAP_VGA_SIZE
        || !(0..=3).contains(&idle_cycle)
        || !(0..=7).contains(&blend)
    {
        return Err(err("indexedmap: incomplete native AI item damage blend input"));
    }
    let idle_cycle = if idle_cycle == 3 { 2 } else { idle_cycle };
    let mut frame = snapshot.to_vec();
    for target in targets {
        let sprite = units
            .sprite_for_native_slot(cache, target.selector_slot, 0, idle_cycle)
            .map_err(wrap(format!("indexedmap: AI item damage target {} selector: ", target.record_index)))?;
        let (x, y) = native_command_tail_origin(target.x, target.y, camera_x, camera_y, 0, -6)?;
        sprite
            .blit_native_damage_blend_at(&mut frame, WORK_STRIDE, x, y, blend, raw_base)
            .map_err(wrap(format!("indexedmap: AI item damage target {}: ", target.record_index)))?;
    }
    let mut viewport = vga.to_vec();
    copy_native_unit_present_viewport(&mut viewport, &frame)?;
    work.copy_from_slice(&frame);
    vga.copy_from_slice(&viewport);
    Ok(())
}

/// ComposeNativeCommandHealDigitFrame consumes one typed 0x1df58 queue frame.
///
/// Go：`native_command_heal_tail.go:137`
#[allow(clippy::too_many_arguments)]
pub fn compose_native_command_heal_digit_frame(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    digits: &[LMI1Entry],
    queue: &[NativePresentationDigit],
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
    vertical: [i64; 25],
    frame_index: i64,
) -> Result<()> {
    compose_native_command_heal_digit_frame_inner(
        work,
        vga,
        snapshot,
        digits,
        queue,
        targets,
        camera_x,
        camera_y,
        vertical,
        frame_index,
    )
    .map(|_| ())
}

/// Go：`native_command_heal_tail.go:142`
#[allow(clippy::too_many_arguments)]
pub fn compose_native_command_heal_digit_frame_with_sources(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    digits: &[LMI1Entry],
    queue: &[NativePresentationDigit],
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
    vertical: [i64; 25],
    frame_index: i64,
) -> Result<Vec<NativeCommandTailLayer>> {
    compose_native_command_heal_digit_frame_inner(
        work,
        vga,
        snapshot,
        digits,
        queue,
        targets,
        camera_x,
        camera_y,
        vertical,
        frame_index,
    )
}

/// Go：`native_command_heal_tail.go:146`。Go 以 `map[int]` 依 RecordIndex 建索引，重複時後者覆蓋前者；
/// `BTreeMap::insert` 相同，查詢與迭代順序無關。
#[allow(clippy::too_many_arguments)]
fn compose_native_command_heal_digit_frame_inner(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    digits: &[LMI1Entry],
    queue: &[NativePresentationDigit],
    targets: &[NativeCommandHealTailTarget],
    camera_x: i64,
    camera_y: i64,
    vertical: [i64; 25],
    frame_index: i64,
) -> Result<Vec<NativeCommandTailLayer>> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) != NATIVE_MAP_VGA_SIZE
        || !(0..22).contains(&frame_index)
    {
        return Err(err("indexedmap: incomplete native command heal digit input"));
    }
    let by_record: BTreeMap<i64, NativeCommandHealTailTarget> = targets.iter().map(|t| (t.record_index, *t)).collect();
    let mut frame = snapshot.to_vec();
    let mut layers = Vec::with_capacity(queue.len());
    for (index, item) in queue.iter().enumerate() {
        if item.digit == 0 {
            continue;
        }
        if item.digit < 0 || item.digit >= len64(digits) {
            return Err(err(format!(
                "indexedmap: command heal digit descriptor {} unavailable",
                go_sharp_x(item.digit)
            )));
        }
        let Some(target) = by_record.get(&item.target) else {
            return Err(err(format!("indexedmap: command heal digit target {} unavailable", item.target)));
        };
        let phase = frame_index + index as i64 % 4;
        if phase < 0 || phase >= vertical.len() as i64 {
            return Err(err("indexedmap: command heal digit phase outside vertical table"));
        }
        let (x, y) = native_command_tail_origin(
            target.x,
            target.y,
            camera_x,
            camera_y,
            item.position_code,
            vertical[at(phase)].wrapping_sub(3),
        )?;
        digits[at(item.digit)]
            .blit_at(&mut frame, WORK_STRIDE, x, y, false)
            .map_err(wrap(format!("indexedmap: command heal digit queue {index}: ")))?;
        layers.push(native_command_tail_layer(5, item.digit, x, y));
    }
    let mut viewport = vga.to_vec();
    copy_native_unit_present_viewport(&mut viewport, &frame)?;
    work.copy_from_slice(&frame);
    vga.copy_from_slice(&viewport);
    Ok(layers)
}

#[cfg(test)]
mod native_command_heal_tail_test;
