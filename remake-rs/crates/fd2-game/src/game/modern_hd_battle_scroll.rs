//! Go 來源：`remake/cmd/fd2/modern_hd_battle_scroll.go`（138 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Decode only the verified transparent FDOTHER blit. Two different initial
/// destinations distinguish an opaque index0 from a transparent write omission.
///
/// Go：`modern_hd_battle_scroll.go:14`，`func modernHDBattleSurfaceFrame(frame fdother.Frame) (figani.Frame, error)`
pub(crate) fn modern_hd_battle_surface_frame(frame: &fdother::Frame) -> Result<figani::Frame> {
    if frame.width <= 0 || frame.height <= 0 || frame.width > 320 || frame.height > 200 {
        return Err(game_err!("modern HD battle: invalid native surface"));
    }
    let mut a = vec![0u8; (frame.width * frame.height) as usize];
    let mut b = vec![255u8; a.len()];
    let mut local = frame.clone();
    (local.x, local.y) = (0, 0);
    local.blit(&mut a, frame.width, -1)?;
    local.blit(&mut b, frame.width, -1)?;
    let mut mask = vec![0u8; a.len()];
    for i in 0..a.len() {
        if a[i] == b[i] {
            mask[i] = 255;
        } else if !(a[i] == 0 && b[i] == 255) {
            return Err(game_err!("modern HD battle: native surface depends on destination"));
        }
    }
    Ok(figani::Frame {
        x: frame.x,
        y: frame.y,
        width: frame.width,
        height: frame.height,
        pixels: a,
        mask,
        ..Default::default()
    })
}
