//! Go 來源：`remake/internal/battle/native_battle_result.go`（31 行）。

use super::*;

/// NativeBattleResultCode205B4 reproduces the raw three-value result written by
/// the 0x205b4 function, whose shared inner entry at 0x205be is called directly
/// by chapter handlers. The numeric result is intentionally not renamed as a
/// victory, defeat, or story transition: chapter handlers may overwrite the
/// same global with the same values for other conditions.
///
/// The native routine starts from code 2, changes it to 0 when any record whose
/// raw +6 byte is zero also has raw +5 bit0 clear, then lets record zero's raw
/// +5 bit0 override the result with code 1.
///
/// Go：`native_battle_result.go:14`，`func NativeBattleResultCode205B4(records []byte, count int) (int, error)`
pub fn native_battle_result_code205_b4(records: &[u8], count: i64) -> Result<i64> {
    if count <= 0 || count > 0x100 || records.len() as i64 != count * NATIVE_RECORD_SIZE {
        return Err(battle_err!("native battle result 0x205b4 records are malformed"));
    }
    let mut result = 2;
    for index in 0..count {
        let record = &records[(index * NATIVE_RECORD_SIZE) as usize..];
        if record[6] == 0 && record[5] & 1 == 0 {
            result = 0;
        }
    }
    if records[5] & 1 != 0 {
        result = 1;
    }
    Ok(result)
}

#[cfg(test)]
mod native_battle_result_test;
