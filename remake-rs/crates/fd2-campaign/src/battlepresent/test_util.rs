//! Go `battlepresent` 測試之間共用的小工具（Go 測試以結構字面值建立 `figani.Frame`、`fdother.Frame`）。

use fd2_formats::fdother;
use fd2_formats::figani::{Animation, Frame};

/// Go `figani.Frame{X: x, Y: y, Width: w, Height: h, Pixels: p, Mask: m, Delay: d}`。
pub(super) fn fig(x: i64, y: i64, w: i64, h: i64, pixels: Vec<u8>, mask: Vec<u8>, delay: i64) -> Frame {
    Frame { x, y, width: w, height: h, pixels, mask, delay, ..Default::default() }
}

/// 1×1 不透明的 figani 幀。
pub(super) fn px1(x: i64, y: i64, value: u8, delay: i64) -> Frame {
    fig(x, y, 1, 1, vec![value], vec![1], delay)
}

/// Go `&figani.Animation{Frames: frames}`。
pub(super) fn anim(frames: Vec<Frame>) -> Animation {
    Animation { frames, ..Default::default() }
}

/// Go `fdother.Frame{X: x, Y: y, Width: w, Height: h, Pixels: p}`（`on_blit` 是私有欄位，只能經建構子）。
pub(super) fn raw_frame(x: i64, y: i64, w: i64, h: i64, pixels: Vec<u8>) -> fdother::Frame {
    fdother::Frame::raw(x, y, w, h, pixels)
}

/// Go 測試的 `make([]byte, 320*200)`。
pub(super) fn surface() -> Vec<u8> {
    vec![0u8; 320 * 200]
}
