//! Go `indexedmap` 測試檔之間共用的輔助函式（`frame_test.go` 的 `solid`／`bank`／`transparentBank`，
//! `hud_test.go` 的 `frame`／`hudFrames`）。

use super::NativeMapHUDFrames;
use fd2_formats::fdicon::{Bank, Sprite};
use fd2_formats::fdother::Frame;

/// Go `solid(v)`。
pub(super) fn solid(v: u8) -> Sprite {
    Sprite { pixels: vec![v; 24 * 24], mask: vec![1; 24 * 24], remap_mask: vec![0; 24 * 24] }
}

/// Go `bank(n, v)`。
pub(super) fn bank(n: usize, v: u8) -> Bank {
    Bank { sprites: (0..n).map(|_| solid(v)).collect() }
}

/// Go `transparentBank(n)`。
pub(super) fn transparent_bank(n: usize) -> Bank {
    Bank { sprites: vec![Sprite::default(); n] }
}

/// Go `identity`：0..255 的恆等 LUT。
pub(super) fn identity() -> Vec<u8> {
    (0..=255u8).collect()
}

/// Go `frame(width, height, pixel)`：以 0x4E63D 的 RLE 編碼成單色圖格。
pub(super) fn frame(width: i64, height: i64, pixel: u8) -> Frame {
    let mut raw = Vec::with_capacity(4);
    raw.extend_from_slice(&(width as u16).to_le_bytes());
    raw.extend_from_slice(&(height as u16).to_le_bytes());
    for _ in 0..height {
        let mut remaining = width;
        while remaining > 0 {
            let count = remaining.min(64);
            raw.push((count - 1) as u8);
            raw.push(pixel);
            remaining -= count;
        }
    }
    Frame::raw(0, 0, width, height, raw)
}

/// Go `hudFrames()`。
pub(super) fn hud_frames() -> NativeMapHUDFrames {
    let mut frames = NativeMapHUDFrames {
        panel: frame(69, 34, 0x5a),
        positive_sign: frame(6, 7, 0x31),
        negative_sign: frame(6, 5, 0x42),
        ..Default::default()
    };
    for digit in 0..10 {
        frames.digits[digit] = frame(6, 8, 0x50 + digit as u8);
        frames.hp_mismatch_digits[digit] = frame(6, 8, 0x70 + digit as u8);
    }
    frames.digits[1] = frame(5, 8, 0x51); // FDOTHER #5 entry #0x20 is 5x8.
    frames.hp_mismatch_digits[1] = frame(5, 8, 0x71);
    frames.hp_equal_overflow = frame(18, 8, 0x7a);
    frames.hp_mismatch_overflow = frame(18, 8, 0x7b);
    frames
}
