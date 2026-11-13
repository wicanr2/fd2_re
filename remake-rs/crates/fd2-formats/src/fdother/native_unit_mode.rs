//! 移植自 Go `native_unit_mode.go`。

use super::native_flow_error::{NativeFlowError, NativeFlowResult};

/// 保存 `0x13a9f` 派發到各 mode callee 之前選出的輸入。欄位刻意保持 raw：callee 的效果與
/// +0x35／+0x36／+0x3d 的意義都沒有推論。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeUnitModePlan {
    pub unit_index: i64,
    pub caller_arg: u8,
    pub mode: u8,
    pub raw_arg35: u8,
    pub raw_arg36: u8,
    pub raw_counter: u8,
}

/// 重現 `0x13a9f` 的入選閘門與欄位讀取。被拒的記錄回傳 `Ok(None)`（Go 的 `(nil, nil)`），
/// 不寫入也不觸發 callback。
pub fn plan_native_unit_mode(
    record: &[u8],
    unit_index: i64,
    caller_arg: u8,
) -> NativeFlowResult<Option<NativeUnitModePlan>> {
    if record.len() < 0x3e {
        return Err(NativeFlowError::UnitModeRecord(record.len()));
    }
    if record[5] & 5 != 0 {
        return Ok(None);
    }
    Ok(Some(NativeUnitModePlan {
        unit_index,
        caller_arg,
        mode: record[0x34] & 0x0f,
        raw_arg35: record[0x35],
        raw_arg36: record[0x36],
        raw_counter: record[0x3d],
    }))
}

#[cfg(test)]
mod tests;
