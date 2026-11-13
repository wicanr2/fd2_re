//! Go 來源：`remake/internal/battle/native_record_flags.go`（28 行）。

use super::*;

pub(crate) const NATIVE_RECORD_SIZE: i64 = 0x50;

/// SetNativeRecordBit7 reproduces 0x13512(index): record[index*0x50+5] |=
/// 0x80. The byte remains an offset-preserving raw flag; callers may project
/// it to Unit.Acted only when their higher-level state has provenance.
///
/// Go：`native_record_flags.go:10`，`func SetNativeRecordBit7(records []byte, index int) error`
pub fn set_native_record_bit7(records: &mut [u8], index: i64) -> Result<()> {
    todo!("battle/native_record_flags.go:SetNativeRecordBit7")
}

/// ClearNativeRecordBit7All reproduces 0x13536's full record scan.
///
/// Go：`native_record_flags.go:20`，`func ClearNativeRecordBit7All(records []byte, count int) error`
pub fn clear_native_record_bit7_all(records: &mut [u8], count: i64) -> Result<()> {
    todo!("battle/native_record_flags.go:ClearNativeRecordBit7All")
}
