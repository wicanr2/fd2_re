//! 畫格呈現（R5 階段 4，r5w）：640×400 CPU 畫格以整數倍最近鄰放大、置中，視窗不是整數倍時留黑邊
//! （docs/goal/121「架構」與範圍表）。
//!
//! Go 的對應是 `modernHDPresentation.drawDefaultFinalScreen`（`ebiten_screen_filter.go:52`）：倍率是整數時以最近鄰貼上，
//! 不是整數時 Ebiten 用自己的 screen shader 混色。121 定案只用整數倍，所以這裡取不超過視窗的最大整數倍（至少 1），
//! 差異列為 KNOWN_DIFFERENCES `window-integer-scale-letterbox`。

/// 邏輯畫布寬（Go `logicalW`）。
pub const LOGICAL_W: u32 = 640;
/// 邏輯畫布高（Go `logicalH`）。
pub const LOGICAL_H: u32 = 400;

/// 視窗裡畫格的位置（實體像素）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Letterbox {
    /// 整數倍率。
    pub scale: u32,
    /// 畫格左上角在視窗中的位置。
    pub ox: u32,
    pub oy: u32,
}

/// 視窗實體大小 `w×h` 時的整數倍置中位置：倍率 `max(1, min(w/640, h/400))`，餘下的寬高平分成黑邊（奇數時左、上少 1）。
pub fn letterbox(w: u32, h: u32) -> Letterbox {
    centered_game(w, h, 0)
}

/// 兩側加寬時，縮放依完整畫布，輸入原點仍在中央遊戲左上角。
pub fn centered_game(w: u32, h: u32, side: u32) -> Letterbox {
    let scale = (w / (LOGICAL_W + 2 * side)).min(h / LOGICAL_H).max(1);
    Letterbox { scale, ox: w.saturating_sub(LOGICAL_W * scale) / 2, oy: h.saturating_sub(LOGICAL_H * scale) / 2 }
}

impl Letterbox {
    pub fn game_x(self, physical: f32, logical: i64) -> i64 {
        if physical < self.ox as f32 { -1 } else { logical }
    }

    /// 視窗實體座標換成 Ebiten 邏輯座標：Ebiten `clientPositionToLogicalPosition`（`internal/ui/context.go:271`）的
    /// `(x - ox) / s`，再照 `input_mobile.go:57` 的 `int(x)` 向零截斷。`s` 與 `ox`／`oy` 用本模組的整數倍位置。
    pub fn to_logical(self, x: f32, y: f32) -> (i64, i64) {
        let s = self.scale as f64;
        (((x as f64 - self.ox as f64) / s) as i64, ((y as f64 - self.oy as f64) / s) as i64)
    }
}

/// CPU 畫格（預乘 alpha 的 RGBA）轉成上傳的貼圖：Ebiten 把畫面貼到每幀清成透明黑的最終畫面，視窗顯示的是
/// 預乘後的 RGB，所以 RGB 照抄、alpha 一律 255。
pub fn opaque_rgba(pix: &[u8]) -> Vec<u8> {
    let mut out = pix.to_vec();
    for px in out.as_chunks_mut::<4>().0 {
        px[3] = 255;
    }
    out
}

/// 預期的視窗畫面（RGBA，列距 `w*4`）：畫格區每個邏輯像素放大成 `scale×scale`，其餘是黑色。
pub fn expected_window(frame: &[u8], w: u32, h: u32) -> Vec<u8> {
    let lb = letterbox(w, h);
    let mut out = vec![0u8; (w * h * 4) as usize];
    for px in out.as_chunks_mut::<4>().0 {
        px[3] = 255;
    }
    for y in 0..h {
        if y < lb.oy || y >= lb.oy + LOGICAL_H * lb.scale {
            continue;
        }
        let sy = (y - lb.oy) / lb.scale;
        for x in 0..w {
            if x < lb.ox || x >= lb.ox + LOGICAL_W * lb.scale {
                continue;
            }
            let sx = (x - lb.ox) / lb.scale;
            let src = ((sy * LOGICAL_W + sx) * 4) as usize;
            let dst = ((y * w + x) * 4) as usize;
            out[dst..dst + 3].copy_from_slice(&frame[src..src + 3]);
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn letterbox_integer_scale_and_centering() {
        assert_eq!(letterbox(1280, 800), Letterbox { scale: 2, ox: 0, oy: 0 });
        assert_eq!(letterbox(1500, 850), Letterbox { scale: 2, ox: 110, oy: 25 });
        assert_eq!(letterbox(1921, 1200), Letterbox { scale: 3, ox: 0, oy: 0 });
        assert_eq!(letterbox(1279, 800), Letterbox { scale: 1, ox: 319, oy: 200 });
        assert_eq!(letterbox(300, 200), Letterbox { scale: 1, ox: 0, oy: 0 });
    }

    #[test]
    fn to_logical_truncates_toward_zero() {
        let lb = letterbox(1500, 850);
        assert_eq!(lb.to_logical(110.0, 25.0), (0, 0));
        assert_eq!(lb.to_logical(111.9, 26.9), (0, 0));
        assert_eq!(lb.to_logical(109.0, 24.0), (0, 0), "-0.5 向零截斷成 0（Go int()）");
        assert_eq!(lb.to_logical(107.0, 22.0), (-1, -1));
        assert_eq!(lb.to_logical(1389.0, 824.0), (639, 399));
    }

    #[test]
    fn expected_window_replicates_pixels() {
        let mut frame = vec![0u8; (LOGICAL_W * LOGICAL_H * 4) as usize];
        frame[0..4].copy_from_slice(&[10, 20, 30, 40]);
        let last = ((LOGICAL_W * LOGICAL_H - 1) * 4) as usize;
        frame[last..last + 4].copy_from_slice(&[1, 2, 3, 0]);
        let out = expected_window(&frame, 1500, 850);
        let at = |x: u32, y: u32| &out[((y * 1500 + x) * 4) as usize..((y * 1500 + x) * 4 + 4) as usize];
        assert_eq!(at(0, 0), [0, 0, 0, 255]);
        assert_eq!(at(110, 25), [10, 20, 30, 255]);
        assert_eq!(at(111, 26), [10, 20, 30, 255]);
        assert_eq!(at(112, 25), [0, 0, 0, 255]);
        assert_eq!(at(1389, 824), [1, 2, 3, 255]);
        assert_eq!(at(1390, 824), [0, 0, 0, 255]);
    }
}
