//! 移植自 Go `indexed_transition.go`：`0x24618` 轉場中的一次 `0x22046` 編排。

use super::{
    CenteredRectLUTRemap, FdotherError, RadialLUTRemap, Result, apply_centered_rect_lut_remap, apply_radial_lut_remap,
};

/// `0x24618` 轉場中一次已驗證的 `0x22046` 編排。兩次放射狀 LUT 處理刻意分開：原生先呼叫
/// `0x219ad`，經 `0x127a9` 重繪相對鏡頭的物件，再呼叫一次 `0x219ad`，最後做一次置中的矩形 LUT
/// 處理。描述子與緩衝來源歸呼叫端，重繪 callback 也由呼叫端提供。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct IndexedTransitionPass {
    pub first_radial: RadialLUTRemap,
    pub second_radial: RadialLUTRemap,
    pub final_rect: CenteredRectLUTRemap,
}

/// 中段重繪 callback：收到正在處理的緩衝。Go 版是 `func([]byte) error`，nil 時拒收；Rust 以
/// `None` 保留那條分支。
pub type IndexedTransitionRedraw<'a> = &'a mut dyn FnMut(&mut [u8]) -> Result<()>;

/// 對驗證而言原子地執行一次 raw 索引處理：三組幾何都在第一個位元組改變前檢查。`redraw` 是必要的，
/// 因為原生中段重繪是會改寫緩衝的階段，不是可省略的簿記；沒有 callback 時失敗即關閉。
pub fn apply_indexed_transition_pass(
    dst: &mut [u8],
    stride: i64,
    lut: &[u8],
    pass: IndexedTransitionPass,
    redraw: Option<IndexedTransitionRedraw>,
) -> Result<()> {
    let Some(redraw) = redraw else {
        return Err(FdotherError::fixed("indexed transition: missing native middle redraw callback"));
    };
    apply_indexed_transition_pass_at(dst, 0, stride, lut, pass, redraw)
}

/// [`apply_indexed_transition_pass`] 的本體，處理 `full[offset..]`，重繪 callback 收到整個 `full`。
/// Go 的 `RunNativeUnitPresentLUTFrame` 把 `work[0x8088:]` 交給處理，再以閉包把完整的 `work` 交給
/// 重繪；兩個切片共用底層陣列。Rust 不能同時持有兩個可變借用，所以由這裡在各階段之間重新切片。
pub(super) fn apply_indexed_transition_pass_at(
    full: &mut [u8],
    offset: usize,
    stride: i64,
    lut: &[u8],
    pass: IndexedTransitionPass,
    redraw: &mut dyn FnMut(&mut [u8]) -> Result<()>,
) -> Result<()> {
    let n = full[offset..].len();
    validate_radial_geometry(n, stride, lut, pass.first_radial)?;
    validate_radial_geometry(n, stride, lut, pass.second_radial)?;
    validate_rect_geometry(n, stride, lut, pass.final_rect)?;
    apply_radial_lut_remap(&mut full[offset..], stride, lut, pass.first_radial)?;
    redraw(full)?;
    apply_radial_lut_remap(&mut full[offset..], stride, lut, pass.second_radial)?;
    apply_centered_rect_lut_remap(&mut full[offset..], stride, lut, pass.final_rect)
}

/// 重用正式運算的完整邊界檢查，但不改呼叫端緩衝。Go 版收 `dst []byte` 只取長度，這裡直接收長度。
pub(super) fn validate_radial_geometry(dst_len: usize, stride: i64, lut: &[u8], spec: RadialLUTRemap) -> Result<()> {
    if lut.len() != 256
        || stride <= 0
        || spec.clip_width <= 0
        || spec.clip_width > stride
        || spec.center_x < 0
        || spec.center_x >= spec.clip_width
        || spec.center_y < 0
        || spec.radius <= 0
        || spec.radius > 0x7fff
        || spec.scale < 0
        || spec.scale > 0x7fff
        || spec.start_y < 0
        || spec.end_y < spec.start_y
        || spec.end_y > dst_len as i64 / stride
    {
        return Err(FdotherError::fixed("indexed transition: invalid radial geometry"));
    }
    Ok(())
}

pub(super) fn validate_rect_geometry(
    dst_len: usize,
    stride: i64,
    lut: &[u8],
    spec: CenteredRectLUTRemap,
) -> Result<()> {
    if lut.len() != 256
        || stride <= 0
        || spec.clip_width <= 0
        || spec.clip_width > stride
        || spec.center_x < 0
        || spec.center_x >= spec.clip_width
        || spec.horizontal_radius < 0
        || spec.horizontal_radius > 0x7fff
        || spec.start_y < 0
        || spec.end_y < spec.start_y
        || spec.end_y > dst_len as i64 / stride
    {
        return Err(FdotherError::fixed("indexed transition: invalid centered geometry"));
    }
    Ok(())
}
