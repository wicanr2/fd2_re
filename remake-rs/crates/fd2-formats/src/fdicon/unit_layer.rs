//! 移植自 Go `unit_layer.go`。

use super::{
    Bank, NATIVE_MAP_STRIDE, NATIVE_SIZE, NativeSelectorCache, Result, Sprite, fixed, len64, native_frame_index,
    native_placement_offset,
};

/// `0x127a9`／`0x127e0` 消費的 raw 子集。`slot` 是 runtime unit+2（不是圖格組）；
/// `pose`、`motion_offset`、`force_base` 分別是 unit+3、+4、+0x26。`inactive` 代表前置的
/// `0x3453e` 名冊閘門。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeUnitLayerEntry {
    pub x: i64,
    pub y: i64,
    pub slot: i64,
    pub pose: i64,
    pub motion_offset: i64,
    pub flags: u8,
    pub force_base: bool,
    pub inactive: bool,
}

struct NativeUnitBlit<'a> {
    sprite: &'a Sprite,
    offset: i64,
    flags: u8,
}

impl Bank {
    /// 重現穩定的 `0x127a9` 名冊 pass。呼叫端提供 `0x127e0` 讀取的 raw 可視範圍，而不是由
    /// 轉接層賦予鏡頭尺寸語意：原版接受 X 在 `[camera_x-1, camera_x+visible_x_max]`、Y 在
    /// `[camera_y-1, camera_y+visible_y_max+1]`。修改 `dst` 前先預檢每個選中項目，格式錯誤的
    /// 可編輯選擇器狀態不會產生只畫一半的索引畫面。
    #[allow(clippy::too_many_arguments)]
    pub fn blit_native_unit_layer(
        &self,
        dst: &mut [u8],
        stride: i64,
        cache: &NativeSelectorCache,
        units: &[NativeUnitLayerEntry],
        camera_x: i64,
        camera_y: i64,
        visible_x_max: i64,
        visible_y_max: i64,
        idle_cycle: i64,
        moving_cycle: i64,
        pixel_shift: i64,
    ) -> Result<()> {
        self.blit_native_unit_layer_with_base_shift(
            dst,
            stride,
            cache,
            units,
            camera_x,
            camera_y,
            visible_x_max,
            visible_y_max,
            idle_cycle,
            moving_cycle,
            pixel_shift,
            0,
        )
    }

    /// 保留在呼叫 `0x127E0` 前暫時調整原版 `[0x53A49]` framebuffer 指標的呼叫端。`base_shift`
    /// 是加在 [`native_placement_offset`] 之後的有號位元組偏移；不改變單位座標、鏡頭範圍、
    /// 動畫狀態或選擇器槽位。
    #[allow(clippy::too_many_arguments)]
    pub fn blit_native_unit_layer_with_base_shift(
        &self,
        dst: &mut [u8],
        stride: i64,
        cache: &NativeSelectorCache,
        units: &[NativeUnitLayerEntry],
        camera_x: i64,
        camera_y: i64,
        visible_x_max: i64,
        visible_y_max: i64,
        idle_cycle: i64,
        moving_cycle: i64,
        pixel_shift: i64,
        base_shift: i64,
    ) -> Result<()> {
        if stride != NATIVE_MAP_STRIDE || visible_x_max < 0 || visible_y_max < 0 {
            return fixed("fdicon: invalid native unit layer");
        }
        let mut commands = Vec::with_capacity(units.len());
        for unit in units {
            if unit.inactive {
                continue;
            }
            if unit.x < camera_x.wrapping_sub(1)
                || unit.x > camera_x.wrapping_add(visible_x_max)
                || unit.y < camera_y.wrapping_sub(1)
                || unit.y > camera_y.wrapping_add(visible_y_max).wrapping_add(1)
            {
                continue;
            }
            let cycle = native_frame_index(unit.motion_offset, unit.force_base, idle_cycle, moving_cycle)?;
            let sprite = self.sprite_for_native_slot(cache, unit.slot, unit.pose, cycle)?;
            let mut offset = native_placement_offset(
                unit.x,
                unit.y,
                camera_x,
                camera_y,
                unit.pose,
                unit.motion_offset,
                pixel_shift,
                unit.force_base,
            )?;
            offset = offset.wrapping_add(base_shift);
            if offset < 0 {
                // 0x127e0 略過負的原版指標偏移。
                continue;
            }
            if offset.wrapping_add((NATIVE_SIZE - 1) * stride + NATIVE_SIZE) > len64(dst) {
                return fixed("fdicon: native unit layer exceeds destination");
            }
            commands.push(NativeUnitBlit { sprite, offset, flags: unit.flags });
        }
        for command in commands {
            command.sprite.blit_for_native_flags_at_offset(dst, stride, command.offset, command.flags)?;
        }
        Ok(())
    }
}
