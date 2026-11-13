//! 移植自 Go `preparation_dispatch.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

/// 一次 `0x1a813` 表格掃描的 raw 結果。表格的重疊位元組配置是刻意的：原版算出
/// `base + 3*i`，再讀偏移 +3、+5 與 +4。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePreparationDispatchMatch {
    pub slot: i64,
    pub entry_start: i64,
    pub function_index: u8,
}

/// 掃描原版正好十六個候選，依來源順序回傳所有符合項目；呼叫端的函式表呼叫刻意不在
/// 這個資料原語之內。
pub fn find_native_preparation_dispatch(
    table: &[u8],
    gate3: u8,
    selector: u8,
) -> NativeFlowResult<Vec<NativePreparationDispatchMatch>> {
    const SLOTS: usize = 16;
    const STRIDE: usize = 3;
    const LAST_READ: usize = 5;
    let need = (SLOTS - 1) * STRIDE + LAST_READ + 1;
    if table.len() < need {
        return Err(NativeFlowError::PreparationDispatchTable { have: table.len(), need });
    }
    let mut matches = Vec::new();
    for slot in 0..SLOTS {
        let start = slot * STRIDE;
        if table[start + 3] != gate3 || table[start + 5] != selector {
            continue;
        }
        matches.push(NativePreparationDispatchMatch {
            slot: slot as i64,
            entry_start: start as i64,
            function_index: table[start + 4],
        });
    }
    Ok(matches)
}

#[cfg(test)]
mod tests;
