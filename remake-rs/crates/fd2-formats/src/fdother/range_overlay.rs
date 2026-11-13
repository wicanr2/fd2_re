//! 移植自 Go `range_overlay.go`：`0x122dc`／`0x126f7` 的戰術範圍覆蓋層與 mode 6 的格子改寫。

use super::{FdotherError, Result, read_resource_i64};
use crate::fdicon::{self, Bank, NATIVE_SIZE};
use std::path::Path;

const NATIVE_RANGE_OVERLAY_RESOURCE: i64 = 1;
pub(super) const NATIVE_RANGE_OVERLAY_STRIDE: i64 = 0x1c8;
const NATIVE_RANGE_OVERLAY_BASE: i64 = 0x8088;
pub(super) const NATIVE_RANGE_OVERLAY_TILES: i64 = 20;

/// `0x122dc` 發出的一次直接 `0x126f7` 呼叫。`descriptor` 是呼叫端的原始描述子庫索引；
/// 不是玩法上的範圍類別，也不是 GUI 的高亮顏色。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeRangeOverlayPlacement {
    pub x: i64,
    pub y: i64,
    pub descriptor: i64,
}

/// 下面的常值依呼叫順序抄自 `0x122dc..0x126f6`。保持為呼叫表，不要正規化成推論出的形狀。
fn native_range_overlay_mode(mode: i64) -> Option<&'static [(i64, i64, i64)]> {
    Some(match mode {
        1 => &[(0, 0, 0)],
        2 => &[(0, 0, 1)],
        3 => &[(0, 0, 14), (0, -1, 2), (-1, 0, 3), (1, 0, 4), (0, 1, 5)],
        4 => &[
            (0, 0, 1),
            (0, -2, 2),
            (-2, 0, 3),
            (2, 0, 4),
            (0, 2, 5),
            (-1, -1, 6),
            (1, -1, 7),
            (-1, 1, 8),
            (1, 1, 9),
            (0, -1, 10),
            (-1, 0, 11),
            (1, 0, 12),
            (0, 1, 13),
        ],
        5 => &[
            (0, 0, 1),
            (0, -3, 2),
            (-3, 0, 3),
            (3, 0, 4),
            (0, 3, 5),
            (-1, -2, 6),
            (-2, -1, 6),
            (1, -2, 7),
            (-2, 2, 7),
            (1, 2, 8),
            (1, -2, 8),
            (2, 2, 9),
            (1, 2, 9),
            (0, -2, 10),
            (-2, 0, 11),
            (2, 0, 12),
            (0, 2, 13),
            (-1, -1, 15),
            (1, -1, 16),
            (-1, 1, 17),
            (1, 1, 18),
        ],
        _ => return None,
    })
}

/// 實體化原始 mode 1..5 在 cursorX/cursorY 周圍的有序 `0x122dc` 呼叫。同一座標的多次呼叫
/// 都保留，因為原生在那裡用不同的描述子。
///
/// Mode 6 沒有 `0x126f7` 呼叫：它改為清掉 `[0x53a51 + 4*(cursorX + cursorY*[0x53ac1]) + 7]`
/// 的一個位元組。這裡刻意拒收，讓繪製端不會把那個格子改寫誤當成繪圖。
pub fn native_range_overlay_placements(
    mode: i64,
    cursor_x: i64,
    cursor_y: i64,
) -> Result<Vec<NativeRangeOverlayPlacement>> {
    let Some(offsets) = native_range_overlay_mode(mode) else {
        return Err(FdotherError::message(format!(
            "fdother: native range overlay mode {mode} has no drawable placement table"
        )));
    };
    Ok(offsets
        .iter()
        .map(|&(dx, dy, descriptor)| NativeRangeOverlayPlacement {
            x: cursor_x.wrapping_add(dx),
            y: cursor_y.wrapping_add(dy),
            descriptor,
        })
        .collect())
}

/// Mode 6 的原始位址運算式。刻意不含格子基底與紀錄儲存：只回傳相對 `[0x53a51]` 的位元組
/// 偏移，寫入前呼叫端必須先建立那個原生記憶體配置。width 必須為正；原生本身不保護座標運算。
pub fn native_range_overlay_mode6_byte_address(width: i64, cursor_x: i64, cursor_y: i64) -> Result<i64> {
    if width <= 0 {
        return Err(FdotherError::fixed("fdother: native range overlay mode 6 width must be positive"));
    }
    Ok(cursor_x.wrapping_add(cursor_y.wrapping_mul(width)).wrapping_mul(4).wrapping_add(7))
}

/// 對一份原始 FDFIELD 合成資源套用 mode 6。`0x108f0..0x10932` 把那份資源載到 `[0x53a51]`，
/// 它的有號 u16 標頭是 width/height，後面每格四位元組；`0x4dbfc` 把第四個位元組（事件高位
/// 元組／原始 blit 模式位元組）初始化為 0xff。所以 mode 6 的 `[base + 4*(x+y*width) + 7]`
/// 寫入恰好是所選格子的第四個位元組，不是範圍覆蓋層的圖格操作。不替清掉那個位元組指定
/// 更高的玩法意義。
pub fn clear_native_range_overlay_mode6_field_byte(field: &mut [u8], cursor_x: i64, cursor_y: i64) -> Result<()> {
    if field.len() < 4 {
        return Err(FdotherError::fixed("fdother: native range overlay mode 6 field is too short"));
    }
    let width = i16::from_le_bytes([field[0], field[1]]) as i64;
    let height = i16::from_le_bytes([field[2], field[3]]) as i64;
    if width <= 0
        || height <= 0
        || cursor_x < 0
        || cursor_x >= width
        || cursor_y < 0
        || cursor_y >= height
        || width > (field.len() as i64 - 4) / 4 / height
    {
        return Err(FdotherError::fixed("fdother: invalid native range overlay mode 6 field coordinate"));
    }
    let offset = native_range_overlay_mode6_byte_address(width, cursor_x, cursor_y)?;
    if offset >= field.len() as i64 {
        return Err(FdotherError::fixed("fdother: native range overlay mode 6 field is truncated"));
    }
    field[offset as usize] = 0;
    Ok(())
}

/// 讀 `0x25c7d..0x25c92` 載到 `[0x53a4d]` 的那一筆 FDOTHER #1 資源。它的 {24,24,20,u32 offsets[]}
/// 標頭與 FDICON.B24 是同一種四模式 RLE 圖庫 ABI；`0x122dc` 的 mode 1..5 只用描述子 0..18。
/// 要求觀察到的 20 筆，可避免部分可編輯素材悄悄改變原生描述子表。
pub fn decode_native_range_overlay_bank(dat_path: impl AsRef<Path>) -> Result<Bank> {
    let raw = read_resource_i64(dat_path.as_ref(), NATIVE_RANGE_OVERLAY_RESOURCE)?;
    let bank =
        fdicon::parse(&raw).map_err(|e| FdotherError::message(format!("fdother: native range overlay bank: {e}")))?;
    if bank.sprites.len() as i64 != NATIVE_RANGE_OVERLAY_TILES {
        return Err(FdotherError::message(format!(
            "fdother: native range overlay has {} descriptors, want {NATIVE_RANGE_OVERLAY_TILES}",
            bank.sprites.len()
        )));
    }
    Ok(bank)
}

/// 對原生 456 stride 工作緩衝執行 `0x122dc` 與 `0x126f7` 已驗證的繪圖部分。描述子圖格是
/// 直接的 `0x4deda` RLE 輸入，所以 `Sprite::blit_at` 刻意保留 mode-3 區段。鏡頭裁切與
/// `0x126f7` 一樣在查描述子之前。原始 mode 0 照 `0x122dc` 的預設分支不做事。Mode 6 仍拒收，
/// 因為它是另一種原始格子改寫，不是繪圖。
///
/// 選到的可見圖格與目的端在寫入前全部先檢查。這是可編輯輸入的安全邊界，不是宣稱原版對
/// 畫面指標做過範圍檢查。
///
/// Go 版接受 nil bank 並拒收；Rust 以參考型別排除 nil，該分支不存在。
#[allow(clippy::too_many_arguments)]
pub fn blit_native_range_overlay(
    bank: &Bank,
    dst: &mut [u8],
    camera_x: i64,
    camera_y: i64,
    visible_width: i64,
    visible_height: i64,
    mode: i64,
    cursor_x: i64,
    cursor_y: i64,
) -> Result<()> {
    if bank.sprites.len() as i64 != NATIVE_RANGE_OVERLAY_TILES {
        return Err(FdotherError::fixed("fdother: incomplete native range overlay descriptor bank"));
    }
    let stride = NATIVE_RANGE_OVERLAY_STRIDE;
    if visible_width <= 0 || visible_height <= 0 || dst.len() as i64 % stride != 0 {
        return Err(FdotherError::fixed("fdother: invalid native range overlay framebuffer"));
    }
    if mode == 0 {
        return Ok(());
    }
    let placements = native_range_overlay_placements(mode, cursor_x, cursor_y)?;
    let mut visible = Vec::with_capacity(placements.len());
    for p in placements {
        if p.x < camera_x
            || p.x >= camera_x.wrapping_add(visible_width)
            || p.y < camera_y
            || p.y >= camera_y.wrapping_add(visible_height)
        {
            continue;
        }
        if p.descriptor < 0 || p.descriptor >= bank.sprites.len() as i64 {
            return Err(FdotherError::fixed("fdother: native range overlay descriptor is outside bank"));
        }
        let x = (NATIVE_RANGE_OVERLAY_BASE % stride).wrapping_add(p.x.wrapping_sub(camera_x).wrapping_mul(NATIVE_SIZE));
        let y = (NATIVE_RANGE_OVERLAY_BASE / stride).wrapping_add(p.y.wrapping_sub(camera_y).wrapping_mul(NATIVE_SIZE));
        if x < 0
            || y < 0
            || x.wrapping_add(NATIVE_SIZE) > stride
            || y.wrapping_add(NATIVE_SIZE) > dst.len() as i64 / stride
        {
            return Err(FdotherError::fixed("fdother: native range overlay destination is outside framebuffer"));
        }
        visible.push((x, y, p.descriptor));
    }
    for (x, y, descriptor) in visible {
        bank.sprites[descriptor as usize].blit_at(dst, stride, x, y).map_err(|e| {
            FdotherError::message(format!("fdother: native range overlay descriptor {descriptor}: {e}"))
        })?;
    }
    Ok(())
}
