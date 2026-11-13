//! 移植自 Go `indexed_transition_lut.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult, at};

/// 把 `0x24618`／`0x22046` 使用的 raw descriptor 索引解析成 FDOTHER#3 的一張 256 位元組重映射表。
/// 原版迴圈走訪 9..1；索引 0 與其他 bank 項目屬於別的呼叫端，這裡拒收而不悄悄沿用。
/// 回傳複本，不共用 bank 的儲存。
pub fn native_indexed_transition_lut(bank: &[Vec<u8>], descriptor: i64) -> NativeFlowResult<Vec<u8>> {
    if !(1..=9).contains(&descriptor) || descriptor >= bank.len() as i64 {
        return Err(NativeFlowError::TransitionDescriptorRange(descriptor));
    }
    let entry = &bank[at(descriptor)];
    if entry.len() != 256 {
        return Err(NativeFlowError::TransitionDescriptorLength { descriptor, len: entry.len() });
    }
    Ok(entry.clone())
}

#[cfg(test)]
mod tests;
