//! 移植自 Go `indexed_transition_buffer.go`。

use super::native_flow_error::{NativeFlowResult, at, fixed};

// 由 0x24618／0x11eb0 還原的原版索引轉場幾何。暫存影像是 128K 工作緩衝區中位元組偏移
// 32904、stride 456 的 312×192 視窗；呈現時複製到 320 位元組 stride 的 VGA 表面。
pub const NATIVE_TRANSITION_STAGE_OFFSET: i64 = 32904;
pub const NATIVE_TRANSITION_STAGE_STRIDE: i64 = 456;
pub const NATIVE_TRANSITION_STAGE_WIDTH: i64 = 312;
pub const NATIVE_TRANSITION_STAGE_HEIGHT: i64 = 192;
pub const NATIVE_TRANSITION_PRESENT_STRIDE: i64 = 320;

/// 每次 `0x22046` descriptor pass 之前原版 memmove 做的整塊種子複製。刻意不解碼
/// descriptor，也不套 LUT。
pub fn copy_native_transition_buffer(dst: &mut [u8], src: &[u8]) -> NativeFlowResult<()> {
    if src.is_empty() || dst.len() < src.len() {
        return fixed("indexed transition: destination buffer is too small");
    }
    dst[..src.len()].copy_from_slice(src);
    Ok(())
}

/// 對應 `0x11eb0(a1=VGA,a2=320,a3=stage,a4=456,a5=312,a6=192)`。只做 raw 索引複製；調色盤與
/// descriptor 語意由呼叫端負責。
///
/// 範圍檢查與 Go 一樣以 64 位元繞回計算；繞回騙過檢查時，之後的切片越界會 panic，
/// 與 Go 執行期的 panic 對應（panic 之前已複製的列保留，順序與 Go 相同）。
pub fn copy_native_transition_viewport(
    dst: &mut [u8],
    dst_stride: i64,
    src: &[u8],
    src_offset: i64,
    src_stride: i64,
    width: i64,
    height: i64,
) -> NativeFlowResult<()> {
    if dst_stride <= 0
        || src_stride <= 0
        || width <= 0
        || height <= 0
        || width > dst_stride
        || width > src_stride
        || src_offset < 0
        || (src_offset % src_stride).wrapping_add(width) > src_stride
    {
        return fixed("indexed transition: invalid viewport geometry");
    }
    if src_offset.wrapping_add(height.wrapping_mul(src_stride)) > src.len() as i64
        || (height - 1).wrapping_mul(dst_stride).wrapping_add(width) > dst.len() as i64
    {
        return fixed("indexed transition: viewport exceeds buffer");
    }
    for row in 0..height {
        let d = row.wrapping_mul(dst_stride);
        let s = src_offset.wrapping_add(row.wrapping_mul(src_stride));
        dst[at(d)..at(d.wrapping_add(width))].copy_from_slice(&src[at(s)..at(s.wrapping_add(width))]);
    }
    Ok(())
}

#[cfg(test)]
mod tests;
