//! 移植自 Go `foreground_layer.go`。

use super::terrain::{
    NativeTerrainCell, native_foreground_frame_index, native_foreground_redraw_cells, native_foreground_redraw_eligible,
};
use super::{Bank, NATIVE_MAP_STRIDE, NATIVE_SIZE, Result, Sprite, at, fixed, len64};

/// `0x129ec` 使用的 raw 單位子集。`unit7`、`race`、`class` 原樣傳遞、不賦予遊戲名稱，
/// 因為它們餵給另外還原的 `0x1f183` 資格閘門。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeForegroundLayerEntry {
    pub x: i64,
    pub y: i64,
    pub pose: u8,
    pub motion_offset: i64,
    pub inactive: bool,
    pub unit7: u8,
    pub race: u8,
    pub class: u8,
}

struct NativeForegroundBlit<'a> {
    sprite: &'a Sprite,
    offset: i64,
    raw: bool,
}

impl Bank {
    /// 合成單位之後穩定的 `0x129ec→0x12ac6` 前景 pass。鏡頭測試是已還原的被呼叫端 ABI。
    /// 原版不檢查記憶體；這裡在索引格子前先略過地圖外的排程座標，這是明確的失敗即關閉
    /// 安全邊界，不代表原版越界記憶體的內容。與單位圖層相同，所有選中的有效 blit 都在
    /// 修改 `dst` 之前先預檢。
    #[allow(clippy::too_many_arguments)]
    pub fn blit_native_foreground_layer(
        &self,
        dst: &mut [u8],
        stride: i64,
        units: &[NativeForegroundLayerEntry],
        map_width: i64,
        cells: &[NativeTerrainCell],
        controls: &[u8],
        camera_x: i64,
        camera_y: i64,
        visible_x_max: i64,
        visible_y_max: i64,
        flip: i64,
        lut: &[u8],
    ) -> Result<()> {
        if stride != NATIVE_MAP_STRIDE
            || map_width <= 0
            || len64(cells) % map_width != 0
            || visible_x_max < 0
            || visible_y_max < 0
            || (flip != 0 && flip != 1)
        {
            return fixed("fdicon: invalid native foreground layer");
        }
        let map_height = len64(cells) / map_width;
        let mut commands = Vec::new();
        for unit in units {
            if !native_foreground_redraw_eligible(unit.inactive, unit.unit7, unit.race, unit.class) {
                continue;
            }
            let (coords, count) = native_foreground_redraw_cells(unit.x, unit.y, unit.pose, unit.motion_offset);
            for coord in &coords[..count] {
                if coord.x < camera_x.wrapping_sub(1)
                    || coord.x > camera_x.wrapping_add(visible_x_max)
                    || coord.y < camera_y.wrapping_sub(1)
                    || coord.y > camera_y.wrapping_add(visible_y_max).wrapping_add(1)
                    || coord.y < 0
                    || coord.x < 0
                    || coord.x >= map_width
                    || coord.y >= map_height
                {
                    continue;
                }
                let cell = cells[at(coord.y * map_width + coord.x)];
                let tile = (cell.tile & 0x3ff) as i64;
                if tile * 4 >= len64(controls) {
                    return fixed("fdicon: foreground terrain control table is too short");
                }
                let (index, present) = native_foreground_frame_index(tile, controls[at(tile * 4)], flip)?;
                if !present {
                    continue;
                }
                let sprite = self.sprite_for(index / 12, (index % 12) / 3, index % 3)?;
                if cell.blit_mode != 0xff && lut.len() != 256 {
                    return fixed("fdicon: foreground LUT is invalid");
                }
                let offset = 0x8088i64
                    .wrapping_add(coord.y.wrapping_sub(camera_y).wrapping_mul(NATIVE_SIZE * stride))
                    .wrapping_add(coord.x.wrapping_sub(camera_x).wrapping_mul(NATIVE_SIZE));
                if offset < 0 || offset.wrapping_add((NATIVE_SIZE - 1) * stride + NATIVE_SIZE) > len64(dst) {
                    return fixed("fdicon: native foreground layer exceeds destination");
                }
                commands.push(NativeForegroundBlit { sprite, offset, raw: cell.blit_mode == 0xff });
            }
        }
        for command in commands {
            if command.raw {
                command.sprite.blit_for_native_flags_at_offset(dst, stride, command.offset, 0)?;
            } else {
                command.sprite.blit_lut_transparent_at_offset(dst, stride, command.offset, lut)?;
            }
        }
        Ok(())
    }
}
