//! 移植自 Go `battlepresent/native_command24_background.go`。

use super::{GoResult, Result, at, err, len64};
use fd2_formats::fdother;
use fd2_formats::figani;

pub const NATIVE_COMMAND24_BACKGROUND_WIDTH: i64 = 320;
pub const NATIVE_COMMAND24_BACKGROUND_HEIGHT: i64 = 200;
pub(super) const NATIVE_COMMAND24_WORK_STRIDE: i64 = 640;

/// NativeCommand24BackgroundInputs is the caller-owned input to sub_29C90's
/// two ten-present viewport slides. Source and Target are already composed
/// 320x200 indexed bases; keeping that boundary explicit avoids guessing the
/// separate sub_2A289 status-panel compositor.
///
/// Go：`native_command24_background.go:19`
#[derive(Debug, Clone, Default)]
pub struct NativeCommand24BackgroundInputs<'a> {
    pub layers: [fdother::Frame; 3],
    pub source: &'a [u8],
    pub target: &'a [u8],
    pub target_idle: figani::Frame,
}

/// NativeCommand24BackgroundPresent retains the work-buffer composition at a
/// viewport boundary. BGLayers is the ordered write history in the other half;
/// transparent BG pixels can retain an earlier layer in that history.
///
/// Go：`native_command24_background.go:29`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand24BackgroundPresent {
    pub pixels: Vec<u8>,
    pub incoming: bool,
    pub content_x: i64,
    pub bgx: i64,
    pub viewport_x: i64,
    pub bg_layers: Vec<i64>,
}

/// Go：`native_command24_background.go:36`
fn background_pixels(presents: Vec<NativeCommand24BackgroundPresent>) -> Vec<Vec<u8>> {
    presents.into_iter().map(|p| p.pixels).collect()
}

/// BuildNativeCommand24BackgroundFrames reproduces the two sub_29C90 viewport
/// loops. It returns no partial frames when an input or transparent blit is
/// invalid, so a presentation owner can remain fail-closed before mutation.
///
/// Go：`native_command24_background.go:47`
pub fn build_native_command24_background_frames(in_: &NativeCommand24BackgroundInputs) -> Result<Vec<Vec<u8>>> {
    build_native_command24_background_presents(in_).map(background_pixels)
}

/// Go 以值收輸入，改寫 `in.Layers[i]` 的 X/Y 只影響副本；這裡先複製三層再改。
fn placed_layers(layers: &[fdother::Frame; 3], message: &str) -> Result<[fdother::Frame; 3]> {
    let mut out = layers.clone();
    for layer in &mut out {
        if layer.width != NATIVE_COMMAND24_BACKGROUND_WIDTH || layer.height != 100 {
            return Err(err(message));
        }
        (layer.x, layer.y) = (0, 50);
    }
    Ok(out)
}

/// Go：`native_command24_background.go:55`
pub fn build_native_command24_background_presents(
    in_: &NativeCommand24BackgroundInputs,
) -> Result<Vec<NativeCommand24BackgroundPresent>> {
    const FRAME_BYTES: i64 = NATIVE_COMMAND24_BACKGROUND_WIDTH * NATIVE_COMMAND24_BACKGROUND_HEIGHT;
    if len64(in_.source) != FRAME_BYTES || len64(in_.target) != FRAME_BYTES {
        return Err(err("battlepresent: command24 background bases must be 320x200"));
    }
    let layers = placed_layers(&in_.layers, "battlepresent: command24 BG layer geometry mismatch")?;
    let stride = at(NATIVE_COMMAND24_WORK_STRIDE);
    let mut work = vec![0u8; at(NATIVE_COMMAND24_WORK_STRIDE * NATIVE_COMMAND24_BACKGROUND_HEIGHT)];
    for y in 0..200usize {
        work[y * stride + 320..y * stride + 640].copy_from_slice(&in_.source[y * 320..(y + 1) * 320]);
    }
    let mut frames = Vec::with_capacity(20);
    let mut history: Vec<i64> = Vec::new();
    for i in (0..=9i64).rev() {
        layers[at(i % 3)].blit_at(&mut work, NATIVE_COMMAND24_WORK_STRIDE, 0, -1).go()?;
        history.push(i % 3);
        frames.push(NativeCommand24BackgroundPresent {
            pixels: native_command24_viewport(&work, 32 * i),
            content_x: 320,
            bgx: 0,
            viewport_x: 32 * i,
            bg_layers: history.clone(),
            ..Default::default()
        });
    }

    work.fill(0);
    for y in 0..200usize {
        work[y * stride..y * stride + 320].copy_from_slice(&in_.target[y * 320..(y + 1) * 320]);
    }
    in_.target_idle.blit_at(&mut work, NATIVE_COMMAND24_WORK_STRIDE).go()?;
    history.clear();
    for j in (0..=9i64).rev() {
        layers[at((j + 2) % 3)].blit_at(&mut work, NATIVE_COMMAND24_WORK_STRIDE, 320, -1).go()?;
        history.push((j + 2) % 3);
        frames.push(NativeCommand24BackgroundPresent {
            pixels: native_command24_viewport(&work, 32 * j),
            incoming: true,
            content_x: 0,
            bgx: 320,
            viewport_x: 32 * j,
            bg_layers: history.clone(),
        });
    }
    Ok(frames)
}

/// BuildNativePhysicalRightBackgroundFrames 實作 0x29DED 的右向捲動。
/// Source 是 caller 留在 work640 左半的完整畫面。Target 已按 BG→TAI→panel
/// 建立；0x29DED 的明確 memset 後才複製到右半。兩段共 9+10 presents。
/// 證據與 READY 契約：docs/data/ida/fd2_physical_background_selection_20261004.json。
///
/// Go：`native_command24_background.go:104`
pub fn build_native_physical_right_background_frames(in_: &NativeCommand24BackgroundInputs) -> Result<Vec<Vec<u8>>> {
    build_native_physical_right_background_presents(in_).map(background_pixels)
}

/// Go：`native_command24_background.go:112`
pub fn build_native_physical_right_background_presents(
    in_: &NativeCommand24BackgroundInputs,
) -> Result<Vec<NativeCommand24BackgroundPresent>> {
    const FRAME_BYTES: i64 = NATIVE_COMMAND24_BACKGROUND_WIDTH * NATIVE_COMMAND24_BACKGROUND_HEIGHT;
    if len64(in_.source) != FRAME_BYTES || len64(in_.target) != FRAME_BYTES {
        return Err(err("battlepresent: physical background bases must be 320x200"));
    }
    let layers = placed_layers(&in_.layers, "battlepresent: physical BG layer geometry mismatch")?;
    let stride = at(NATIVE_COMMAND24_WORK_STRIDE);
    let mut work = vec![0u8; at(NATIVE_COMMAND24_WORK_STRIDE * NATIVE_COMMAND24_BACKGROUND_HEIGHT)];
    for y in 0..200usize {
        work[y * stride..y * stride + 320].copy_from_slice(&in_.source[y * 320..(y + 1) * 320]);
    }
    let mut frames = Vec::with_capacity(19);
    let mut history: Vec<i64> = Vec::new();
    for i in 1..10i64 {
        layers[at(i % 3)].blit_at(&mut work, NATIVE_COMMAND24_WORK_STRIDE, 320, -1).go()?;
        history.push(i % 3);
        frames.push(NativeCommand24BackgroundPresent {
            pixels: native_command24_viewport(&work, 32 * i),
            content_x: 0,
            bgx: 320,
            viewport_x: 32 * i,
            bg_layers: history.clone(),
            ..Default::default()
        });
    }
    work.fill(0);
    for y in 0..200usize {
        work[y * stride + 320..y * stride + 640].copy_from_slice(&in_.target[y * 320..(y + 1) * 320]);
    }
    in_.target_idle.blit_at_base(&mut work, NATIVE_COMMAND24_WORK_STRIDE, 320).go()?;
    history.clear();
    for i in 1..=10i64 {
        layers[at((i + 1) % 3)].blit_at(&mut work, NATIVE_COMMAND24_WORK_STRIDE, 0, -1).go()?;
        history.push((i + 1) % 3);
        frames.push(NativeCommand24BackgroundPresent {
            pixels: native_command24_viewport(&work, 32 * i),
            incoming: true,
            content_x: 320,
            bgx: 0,
            viewport_x: 32 * i,
            bg_layers: history.clone(),
        });
    }
    Ok(frames)
}

/// Go：`native_command24_background.go:160`
pub(super) fn native_command24_viewport(work: &[u8], base: i64) -> Vec<u8> {
    let (w, stride) = (at(NATIVE_COMMAND24_BACKGROUND_WIDTH), at(NATIVE_COMMAND24_WORK_STRIDE));
    let mut out = vec![0u8; at(NATIVE_COMMAND24_BACKGROUND_WIDTH * NATIVE_COMMAND24_BACKGROUND_HEIGHT)];
    for y in 0..at(NATIVE_COMMAND24_BACKGROUND_HEIGHT) {
        let start = y * stride + at(base);
        out[y * w..(y + 1) * w].copy_from_slice(&work[start..start + w]);
    }
    out
}

#[cfg(test)]
pub(super) mod native_command24_background_test;
