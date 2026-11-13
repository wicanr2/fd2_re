//! 移植自 Go `terrain.go`。

use super::{Bank, NATIVE_MAP_STRIDE, NATIVE_SIZE, Result, at, fixed, len64};

/// 一格已解碼的 FDFIELD 合成格。`tile` 是 raw 地形 word；`blit_mode` 是合成項目
/// byte+3（事件 word 的高位元組）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTerrainCell {
    pub tile: u16,
    pub blit_mode: u8,
}

/// `0x12e38` 對一格 FDFIELD 合成格產生的八位元組結果。`tile` 是合成 word 遮成十位元，
/// `event_low` 是下一個位元組遮成五位元，`control` 是該 tile 選中的 FDSHAP 四位元組
/// 地形紀錄原值。原版地圖 HUD 先消費這些值，再視需要查游標下的單位。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTerrainCursorInfo {
    pub tile: u16,
    pub event_low: u8,
    pub control: [u8; 4],
}

/// `0x1acf3` 在 456 位元組 stride 工作緩衝區內的位元組偏移契約。只描述已證實的合成
/// 目的端；資源選擇與十進位字形繪製由各自的原版基本操作負責。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeMapHUDLayout {
    pub frame: i64,
    pub terrain: i64,
    pub ap: i64,
    pub dp: i64,
    pub unit: i64,
    pub hp: i64,
}

/// `0x2b5e1` 對每個最終目標消費的 raw 輸入。`gate` 是未命名的 `0x1f183` 結果；
/// `control` 是 `0x12e38` 選出的 FDSHAP 四位元組紀錄。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandBackgroundTarget {
    pub gate: bool,
    pub control: [u8; 4],
}

/// 重現 `0x2b5e1` 由後往前掃描最終目標。目標通過 raw gate 且選擇器非 0 時保留目前值；
/// 否則換成解碼後的 control byte+2。這個位元組的場景／地形意義刻意不推論。
pub fn native_command_background_selector(initial: u8, targets: &[NativeCommandBackgroundTarget]) -> u8 {
    let mut selector = initial;
    for target in targets.iter().rev() {
        if !target.gate || selector == 0 {
            selector = target.control[2];
        }
    }
    selector
}

/// 重現 `0x1acf3` 選定 raw 水平錨點後使用的固定偏移。原版呼叫端一律傳 456 位元組的
/// 地圖工作緩衝區 stride；接受其他 stride 會掩蓋轉接錯誤，所以失敗即關閉。
/// `anchor_x+69` 與 Go 一樣以 64 位元繞回比較。
pub fn native_map_hud_layout_for(anchor_x: i64, stride: i64) -> Result<NativeMapHUDLayout> {
    if stride != NATIVE_MAP_STRIDE || anchor_x < 0 || anchor_x.wrapping_add(69) > 320 {
        return fixed("fdicon: invalid native map HUD layout");
    }
    let base = (stride * 157).wrapping_add(anchor_x);
    Ok(NativeMapHUDLayout {
        frame: base,
        // 0x1adbf 對地形圖示計算 ebp + 5*stride + 6，與選用的單位圖示共用第 5 列位置。
        terrain: base.wrapping_add(NATIVE_MAP_STRIDE * 5 + 6),
        ap: base.wrapping_add(stride * 8 + 0x2b),
        dp: base.wrapping_add(stride * 19 + 0x2b),
        unit: base.wrapping_add(stride * 5 + 6),
        hp: base.wrapping_add(stride * 21 + 9),
    })
}

/// 重現 `0x12e38` 對一組 raw FDFIELD tile／event 的解碼。`controls` 必須是選定的 FDSHAP
/// 控制表（每筆四位元組）；格式錯誤即拒收，不猜測。
pub fn native_terrain_cursor_info_for_cell(
    tile_word: u16,
    event_word: u16,
    controls: &[u8],
) -> Result<NativeTerrainCursorInfo> {
    let tile = tile_word & 0x3ff;
    let off = tile as usize * 4;
    if off + 4 > controls.len() {
        return fixed("fdicon: terrain cursor control table is too short");
    }
    let mut info = NativeTerrainCursorInfo { tile, event_low: (event_word as u8) & 0x1f, control: [0; 4] };
    info.control.copy_from_slice(&controls[off..off + 4]);
    Ok(info)
}

/// 重現 `0x1ae4d` 選用的游標單位圖示選擇器：以 unit+2 × 12 加 raw 動畫狀態索引 runtime
/// FDICON 指標表，但狀態 3 映成狀態 1。raw 狀態不命名，因為這個 HUD 用途沒有建立遊戲標籤。
pub fn native_map_hud_unit_frame_index(sprite_group: i64, raw_state: i64) -> Result<i64> {
    if sprite_group < 0 || !(0..=3).contains(&raw_state) {
        return fixed("fdicon: invalid native map HUD unit selector");
    }
    let state = if raw_state == 3 { 1 } else { raw_state };
    Ok(sprite_group.wrapping_mul(12).wrapping_add(state))
}

/// 原版戰場地圖的世界格座標。刻意允許可視範圍或地圖範圍之外的座標：`0x129ec` 把這項
/// 檢查交給 `0x12ac6`，而不是裁切自己的呼叫排程。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCellCoordinate {
    pub x: i64,
    pub y: i64,
}

/// 重現 `0x129ec` 的兩道名冊閘門。槽位必須先是作用中（`0x3453e` 的 unit+5 bit0 查詢），
/// 再通過 raw `0x1f183` 判斷式：unit+7!=0x1c 且 class==0x13 或 race 為 4、5 時不做前景重繪。
/// 這些都是 raw 欄位值，刻意不賦予遊戲或視覺標籤。
pub fn native_foreground_redraw_eligible(inactive: bool, unit7: u8, race: u8, class: u8) -> bool {
    if inactive || unit7 == 0x1c {
        return !inactive;
    }
    class != 0x13 && race != 4 && race != 5
}

/// 重現單位圖格畫完後 `0x129ec` 的確切呼叫順序。一律先重繪單位所在格，再重繪上方一格。
/// unit+4 移動偏移非 0 時再加一格依 pose 而定的鄰格：下、左、上（兩格）、其他 pose 值一律右。
///
/// 地圖／鏡頭範圍檢查由原版被呼叫端負責，所以這個純排程不能丟掉負值或畫面外的座標。
/// 靜止時回傳數量 2，移動中為 3。座標加減沿用 Go 的 64 位元繞回。
pub fn native_foreground_redraw_cells(
    x: i64,
    y: i64,
    pose: u8,
    movement_offset: i64,
) -> ([NativeCellCoordinate; 3], usize) {
    let mut cells = [
        NativeCellCoordinate { x, y },
        NativeCellCoordinate { x, y: y.wrapping_sub(1) },
        NativeCellCoordinate::default(),
    ];
    if movement_offset == 0 {
        return (cells, 2);
    }
    cells[2] = match pose {
        0 => NativeCellCoordinate { x, y: y.wrapping_add(1) },
        1 => NativeCellCoordinate { x: x.wrapping_sub(1), y },
        2 => NativeCellCoordinate { x, y: y.wrapping_sub(2) },
        _ => NativeCellCoordinate { x: x.wrapping_add(1), y },
    };
    (cells, 3)
}

/// 重現 `0x11eee` 的 FDSHAP 描述子選擇器。`tile` 是合成 word 的低 10 位元；`flags` 是選定
/// 地形控制位元組。`flip` 是原版 `0x53a40`（0 或 1），`cycle` 是 `0x53c0b`。
/// 旗標優先序：0x08、0x10、0x04，最後是基底 tile。
pub fn native_terrain_frame_index(tile: i64, flags: u8, flip: i64, cycle: i64) -> Result<i64> {
    if !(0..=0x3ff).contains(&tile) || (flip != 0 && flip != 1) {
        return fixed("fdicon: invalid native terrain frame selector");
    }
    if flags & 0x08 != 0 {
        return Ok(tile + 2 * flip);
    }
    if flags & 0x10 != 0 {
        return Ok(tile.wrapping_add(trunc_div2(cycle)));
    }
    if flags & 0x04 != 0 {
        return Ok(tile + flip);
    }
    Ok(tile)
}

/// Go 版的 `truncDiv2`：負值先取負再除。`-v` 對 `i64::MIN` 繞回成自己，所以
/// `trunc_div2(i64::MIN)` 是 `+2^62`，與 `i64::MIN / 2` 不同；照 Go 原樣保留。
fn trunc_div2(v: i64) -> i64 {
    if v < 0 {
        return (v.wrapping_neg() / 2).wrapping_neg();
    }
    v / 2
}

/// 重現 `0x12ac6` 的前景選擇器。只有地形控制位元 0x80 會畫前景格。位元 0x08 加上兩倍的
/// 原版 flip，而 FDSHAP 偏移查詢比該索引多一項（base+0x0a，地形 pass 是 base+0x06）。
/// 回傳 `(索引, 是否存在)`；不存在時索引為 0，與 Go 版相同。
pub fn native_foreground_frame_index(tile: i64, flags: u8, flip: i64) -> Result<(i64, bool)> {
    if !(0..=0x3ff).contains(&tile) || (flip != 0 && flip != 1) {
        return fixed("fdicon: invalid native foreground selector");
    }
    if flags & 0x80 == 0 {
        return Ok((0, false));
    }
    let mut tile = tile;
    if flags & 0x08 != 0 {
        tile += 2 * flip;
    }
    Ok((tile + 1, true))
}

impl Bank {
    /// 依 `0x11eee` 合成一格已選定的 FDFIELD 地形格：先選 FDSHAP 圖格，合成項目
    /// byte+3 == 0xff 時才用 raw `0x4deda`，其餘用 LUT 版 `0x4dcc6`。鏡頭走訪、LUT 相位
    /// 選擇與前景重繪由呼叫端負責。
    #[allow(clippy::too_many_arguments)]
    pub fn blit_native_terrain_cell(
        &self,
        dst: &mut [u8],
        stride: i64,
        x: i64,
        y: i64,
        tile: i64,
        flags: u8,
        blit_mode: u8,
        flip: i64,
        cycle: i64,
        lut: &[u8],
    ) -> Result<()> {
        let index = native_terrain_frame_index(tile, flags, flip, cycle)?;
        let sprite = self.sprite_for(index / 12, (index % 12) / 3, index % 3)?;
        if blit_mode == 0xff {
            return sprite.blit_at(dst, stride, x, y);
        }
        sprite.blit_lut(dst, stride, x, y, lut)
    }

    /// 重現單位繪製後的 `0x12ac6`。沒有前景旗標時不做事；另一分支用 `0x4dd52` 語意，
    /// 其 mode-3 區段保留目的端像素。
    #[allow(clippy::too_many_arguments)]
    pub fn blit_native_foreground_cell(
        &self,
        dst: &mut [u8],
        stride: i64,
        x: i64,
        y: i64,
        tile: i64,
        flags: u8,
        blit_mode: u8,
        flip: i64,
        lut: &[u8],
    ) -> Result<()> {
        let (index, present) = native_foreground_frame_index(tile, flags, flip)?;
        if !present {
            return Ok(());
        }
        let sprite = self.sprite_for(index / 12, (index % 12) / 3, index % 3)?;
        if blit_mode == 0xff {
            return sprite.blit_at(dst, stride, x, y);
        }
        sprite.blit_lut_transparent(dst, stride, x, y, lut)
    }

    /// 重現 `0x11eee` 的可視格迴圈。`controls` 是選定的 raw FDSHAP 地形控制表（每個基底
    /// tile 四位元組）；這裡只消費 byte 0，與原版相同。時序由呼叫端負責，LUT 也由呼叫端
    /// 明確選定，這裡不推進。中途某格失敗時，之前的格已經寫入目的端，與 Go 版相同。
    #[allow(clippy::too_many_arguments)]
    pub fn blit_native_terrain_region(
        &self,
        dst: &mut [u8],
        stride: i64,
        dst_x: i64,
        dst_y: i64,
        map_width: i64,
        cells: &[NativeTerrainCell],
        controls: &[u8],
        map_x: i64,
        map_y: i64,
        width: i64,
        height: i64,
        flip: i64,
        cycle: i64,
        lut: &[u8],
    ) -> Result<()> {
        if map_width <= 0
            || width < 0
            || height < 0
            || map_x < 0
            || map_y < 0
            || len64(cells) % map_width != 0
            || map_x.wrapping_add(width) > map_width
            || map_y.wrapping_add(height) > len64(cells) / map_width
        {
            return fixed("fdicon: invalid native terrain region");
        }
        for row in 0..height {
            for col in 0..width {
                let cell =
                    cells[at(map_y.wrapping_add(row).wrapping_mul(map_width).wrapping_add(map_x).wrapping_add(col))];
                let tile = (cell.tile & 0x3ff) as i64;
                if tile * 4 >= len64(controls) {
                    return fixed("fdicon: terrain control table is too short");
                }
                self.blit_native_terrain_cell(
                    dst,
                    stride,
                    dst_x.wrapping_add(col.wrapping_mul(NATIVE_SIZE)),
                    dst_y.wrapping_add(row.wrapping_mul(NATIVE_SIZE)),
                    tile,
                    controls[at(tile * 4)],
                    cell.blit_mode,
                    flip,
                    cycle,
                    lut,
                )?;
            }
        }
        Ok(())
    }
}
