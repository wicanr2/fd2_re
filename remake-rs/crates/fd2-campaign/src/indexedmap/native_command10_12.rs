//! 移植自 Go `indexedmap/native_command10_12.go`。

use super::frame::{STEADY_VIEWPORT_OFFSET, STEADY_VIEWPORT_WIDTH, VIEW_HEIGHT, VIEW_WIDTH, WORK_BASE, WORK_STRIDE};
use super::{
    NATIVE_MAP_VGA_SIZE, NATIVE_UNIT_PRESENT_WORK_SIZE, NativeTransitionFrameInput, Result, at,
    copy_native_unit_present_viewport, err, len64, redraw_native_unit_present_objects,
};
use fd2_formats::fdicon::{self, NATIVE_SIZE};

const NATIVE_COMMAND_SAMPLING_UNIT: i64 = 0xc00;

/// Go：`native_command10_12.go:12`。Go 的 `/`、`%` 向零截斷，Rust 相同。
fn floor_div_mod(value: i64, divisor: i64) -> (i64, i64) {
    let (mut q, mut r) = (value / divisor, value % divisor);
    if r < 0 {
        r += divisor;
        q -= 1;
    }
    (q, r)
}

/// ComposeNativeCommand10To12Surface reproduces one 0x1F558 sampling pass,
/// the following 0x127E0 object redraw, and the final 312x192 viewport copy.
/// xOrigin/yOrigin/increment are the untouched signed table values from
/// 0x52096/0x520A2/0x520AE. Input rejection leaves work and VGA unchanged.
///
/// Go：`native_command10_12.go:25`
pub fn compose_native_command10_to12_surface(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &NativeTransitionFrameInput,
    x_origin: i64,
    y_origin: i64,
    increment: i64,
) -> Result<()> {
    let Some(terrain_bank) = in_.terrain_bank else {
        return Err(err("indexedmap: incomplete native command10-12 sampling input"));
    };
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) != NATIVE_MAP_VGA_SIZE
        || in_.map_width <= 0
        || len64(&in_.cells) % in_.map_width != 0
        || !in_.controls.len().is_multiple_of(4)
        || increment <= 0
    {
        return Err(err("indexedmap: incomplete native command10-12 sampling input"));
    }
    let map_height = len64(&in_.cells) / in_.map_width;
    let mut terrain = vec![0u8; at(NATIVE_MAP_VGA_SIZE)];
    let start_x = in_
        .camera_x
        .wrapping_mul(NATIVE_COMMAND_SAMPLING_UNIT)
        .wrapping_add(13 * (NATIVE_COMMAND_SAMPLING_UNIT / 2))
        .wrapping_add(x_origin)
        .wrapping_sub(156i64.wrapping_mul(increment));
    let start_y = in_
        .camera_y
        .wrapping_mul(NATIVE_COMMAND_SAMPLING_UNIT)
        .wrapping_add(8 * (NATIVE_COMMAND_SAMPLING_UNIT / 2))
        .wrapping_add(y_origin)
        .wrapping_sub(96i64.wrapping_mul(increment));
    for y in 0..VIEW_HEIGHT {
        let (tile_y, rem_y) =
            floor_div_mod(start_y.wrapping_add(y.wrapping_mul(increment)), NATIVE_COMMAND_SAMPLING_UNIT);
        let pixel_y = rem_y / 128;
        for x in 0..STEADY_VIEWPORT_WIDTH {
            let (tile_x, rem_x) =
                floor_div_mod(start_x.wrapping_add(x.wrapping_mul(increment)), NATIVE_COMMAND_SAMPLING_UNIT);
            let pixel_x = rem_x / 128;
            if tile_x < 0 || tile_x >= in_.map_width || tile_y < 0 || tile_y >= map_height {
                continue;
            }
            let cell = in_.cells[at(tile_y * in_.map_width + tile_x)];
            let tile = (cell.tile & 0x3ff) as i64;
            if tile * 4 >= len64(in_.controls) {
                return Err(err(format!("indexedmap: native command10-12 terrain control {tile} unavailable")));
            }
            let frame =
                fdicon::native_terrain_frame_index(tile, in_.controls[at(tile * 4)], in_.flip, in_.terrain_cycle)?;
            let sprite = match terrain_bank.sprite_for(frame / 12, (frame % 12) / 3, frame % 3) {
                Ok(sprite) if len64(&sprite.pixels) == NATIVE_SIZE * NATIVE_SIZE => sprite,
                _ => {
                    return Err(err(format!("indexedmap: native command10-12 terrain frame {frame} unavailable")));
                }
            };
            terrain[at(STEADY_VIEWPORT_OFFSET + y * VIEW_WIDTH + x)] =
                sprite.pixels[at(pixel_y * NATIVE_SIZE + pixel_x)];
        }
    }
    let (mut next_work, mut next_vga) = (vec![0u8; work.len()], vga.to_vec());
    let width = at(STEADY_VIEWPORT_WIDTH);
    for row in 0..VIEW_HEIGHT {
        let dst = at(WORK_BASE + row * WORK_STRIDE);
        let src = at(STEADY_VIEWPORT_OFFSET + row * VIEW_WIDTH);
        next_work[dst..dst + width].copy_from_slice(&terrain[src..src + width]);
    }
    redraw_native_unit_present_objects(&mut next_work, in_)?;
    copy_native_unit_present_viewport(&mut next_vga, &next_work)?;
    work.copy_from_slice(&next_work);
    vga.copy_from_slice(&next_vga);
    Ok(())
}

#[cfg(test)]
mod native_command10_12_test;
