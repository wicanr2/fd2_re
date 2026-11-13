//! 移植自 Go `preparation_gate.go`。

use super::native_flow_error::{NativeFlowResult, fixed, le_u16};

/// `0x1a866` 讀取的 raw 子集。欄位名稱刻意保留偏移：這個呼叫端沒有建立它們較高層的玩法意義。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePreparationRecord {
    pub raw25: u8,
    pub raw5: u8,
    pub raw6: u8,
    pub raw7: u8,
    pub word40: u16,
    pub word42: u16,
}

/// 只從一筆 0x50 位元組單位記錄取出 `0x1a866` 讀的位元組。過短的記錄拒收。
pub fn parse_native_preparation_record(raw: &[u8]) -> NativeFlowResult<NativePreparationRecord> {
    if raw.len() < 0x44 {
        return fixed("fdother: preparation record is shorter than 0x44 bytes");
    }
    Ok(NativePreparationRecord {
        raw25: raw[0x25],
        raw5: raw[5],
        raw6: raw[6],
        raw7: raw[7],
        word40: le_u16(raw, 0x40),
        word42: le_u16(raw, 0x42),
    })
}

/// 重現 `0x1a866` 對呼叫端 selector 的三個 raw 閘門。刻意不呼叫閘門之後的呈現函式。
pub fn native_preparation_eligible(record: NativePreparationRecord, selector: i64) -> bool {
    record.raw25 != 0 && record.raw6 as i64 == selector && record.raw5 & 1 == 0
}

/// 回傳閘門之後的 word+0x40 值，以及寫入全域狀態的 raw 除數。原版對零延伸的 word+0x42 做
/// 有號除法，減法在 0 夾住。
pub fn native_preparation_adjusted_word40(record: NativePreparationRecord) -> (u16, i64) {
    let divisor = record.word42 as i64 / 10;
    if record.word40 as i64 - divisor < 0 {
        return (0, divisor);
    }
    (record.word40 - divisor as u16, divisor)
}

#[cfg(test)]
mod tests;
