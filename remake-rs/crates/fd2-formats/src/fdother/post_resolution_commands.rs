//! 移植自 Go `post_resolution_commands.go`。

use super::native_flow_error::{NativeFlowResult, at, fixed, le_u16};

/// 保存 `0x1ac62` 讀取的三位元組記錄：一個 kind 位元組，接著一個 little-endian word
/// payload。呼叫端是結算後／互動路徑，不是整備 schema。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePostResolutionCommand {
    pub kind: u8,
    pub payload: u16,
}

/// 以原版的 `base+3*i` 定址解碼 `count` 筆記錄。未知 kind 保持為資料；由呼叫端決定已驗證
/// 的分支能否、何時執行。
///
/// `count*3` 與 Go 一樣以 64 位元繞回；繞回騙過長度檢查時 Go 在 `make` 就 panic，Rust 則在
/// 讀超出 `raw` 時 panic，兩邊都不回傳結果。
pub fn parse_native_post_resolution_commands(
    raw: &[u8],
    count: i64,
) -> NativeFlowResult<Vec<NativePostResolutionCommand>> {
    if count < 0 || (raw.len() as i64) < count.wrapping_mul(3) {
        return fixed("fdother: post-resolution command stream is truncated");
    }
    let mut commands = Vec::new();
    for i in 0..count {
        let offset = at(i * 3);
        commands.push(NativePostResolutionCommand { kind: raw[offset], payload: le_u16(raw, offset + 1) });
    }
    Ok(commands)
}

#[cfg(test)]
mod tests;
