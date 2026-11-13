//! 移植自 Go `indexed_copy.go`。

use super::{Result, at, fixed, len64};

/// 重現 `0x11eb0` 逐列 memmove 的契約。呼叫端提供已加上偏移的索引緩衝區與明確 stride；
/// 一般地圖重繪用來源 buffer+0x8088、stride 456、寬 320、高 192，目的端 VGA+0x504、
/// stride 320。這個基本操作不擁有 VGA 緩衝區，也不負責呈現像素。
///
/// 範圍檢查沿用 Go 的 64 位元繞回；檢查被繞回騙過時，切片越界一樣會 panic。
pub fn copy_native_indexed_region(
    dst: &mut [u8],
    dst_stride: i64,
    src: &[u8],
    src_stride: i64,
    width: i64,
    height: i64,
) -> Result<()> {
    if dst_stride < 0 || src_stride < 0 || width < 0 || height < 0 || dst_stride < width || src_stride < width {
        return fixed("fdicon: invalid native indexed copy region");
    }
    if height == 0 {
        return Ok(());
    }
    let last = (height - 1).wrapping_mul(dst_stride).wrapping_add(width);
    if last > len64(dst) || (height - 1).wrapping_mul(src_stride).wrapping_add(width) > len64(src) {
        return fixed("fdicon: native indexed copy region exceeds buffer");
    }
    for row in 0..height {
        let d = row.wrapping_mul(dst_stride);
        let s = row.wrapping_mul(src_stride);
        dst[at(d)..at(d.wrapping_add(width))].copy_from_slice(&src[at(s)..at(s.wrapping_add(width))]);
    }
    Ok(())
}
