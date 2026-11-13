//! 移植自 Go `native_compound_palette.go`。

use super::native_flow_error::{NativeFlowResult, at, fixed};

/// 重現 `sub_286BD` 由呼叫端指定 `[start,end)` 的 DAC 內插。除法是有號、向零截斷
/// （Rust 的 `i64 /` 同樣向零截斷），結果與 Go 的 `byte(...)` 一樣取低 8 位元。
pub fn interpolate_native_compound_dac(
    baseline: &[u8],
    start: i64,
    end: i64,
    delta: i64,
    raw: [u8; 3],
) -> NativeFlowResult<Vec<u8>> {
    if baseline.len() != 256 * 3 || start < 0 || end > 256 || start >= end || !(0..=40).contains(&delta) {
        return fixed("fdother: native compound DAC interpolation unavailable");
    }
    let mut out = baseline.to_vec();
    for index in start..end {
        for component in 0..3 {
            let base = baseline[at(3 * index + component)] as i64;
            let origin = raw[at(component)] as i64;
            out[at(3 * index + component)] = (origin + delta * (base - origin) / 40) as u8;
        }
    }
    Ok(out)
}

#[cfg(test)]
mod tests;
