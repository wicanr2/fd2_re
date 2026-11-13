//! Go 來源：`remake/internal/battle/native_equipment.go`（132 行）。

use super::*;

pub(crate) const NATIVE_EQUIPMENT_RECORD_SIZE: i64 = 0x50;

pub(crate) const NATIVE_INVENTORY_OFFSET: i64 = 0xa;

pub(crate) const NATIVE_INVENTORY_CELLS: i64 = 8;

pub(crate) const NATIVE_EQUIPPED_MASK: i64 = 0x40;

/// Go：`native_equipment.go:16`，`func nativeSignedWord(buf []byte, off int) int16`
pub(crate) fn native_signed_word(buf: &[u8], off: i64) -> i16 {
    todo!("battle/native_equipment.go:nativeSignedWord")
}

/// Go：`native_equipment.go:20`，`func nativePutWord(buf []byte, off int, value int16)`
pub(crate) fn native_put_word(buf: &mut [u8], off: i64, value: i16) {
    todo!("battle/native_equipment.go:nativePutWord")
}

/// FD2.EXE 0x5018d stores the IEEE-754 double 1.15 exactly as
/// 2589569785738035 / 2251799813685248. 0x1b750 multiplies in x87 extended
/// precision and 0x377a4 rounds toward zero, so a host binary64 multiply is
/// not interchangeable (for example, native maps 20 to 22, not 23).
/// Go 原式：`uint64(2589569785738035)`
pub(crate) const NATIVE_SCALE115_NUMERATOR: u64 = 2589569785738035;

/// Go 原式：`uint64(2251799813685248)`
pub(crate) const NATIVE_SCALE115_DENOMINATOR: u64 = 2251799813685248;

/// Go：`native_equipment.go:33`，`func nativeScale115TowardZero(value int32) int32`
pub(crate) fn native_scale115_toward_zero(value: i32) -> i32 {
    todo!("battle/native_equipment.go:nativeScale115TowardZero")
}

/// Go：`native_equipment.go:47`，`func nativeEquipmentTotals(record, itemTable []byte, runtimeModifiers bool) ([4]int32, error)`
pub(crate) fn native_equipment_totals(record: &[u8], item_table: &[u8], runtime_modifiers: bool) -> Result<[i32; 4]> {
    todo!("battle/native_equipment.go:nativeEquipmentTotals")
}

/// ApplyNativeEquipmentRecalc reproduces the proven arithmetic tail of
/// 0x1145a. record is one native 0x50-byte persistent record; itemTable starts
/// at the relocated 0x602ad row table. The eight inventory cells are raw
/// [flag,item] pairs at +0x0a. Only flag bit 0x40 selects a row. Cross-checking
/// all 215 known raw rows against item.json identifies row +1/+3/+5/+7 as the
/// AP/HIT/DP/EV equipment words. Native accumulation order is AP/DP/HIT/EV into
/// record +0x48/+0x4a/+0x4c/+0x4e. All word writes wrap at 16 bits.
///
/// The operation is fail-closed and atomic: malformed record/table input is
/// rejected before any destination byte is changed.
///
/// Go：`native_equipment.go:107`，`func ApplyNativeEquipmentRecalc(record, itemTable []byte) error`
pub fn apply_native_equipment_recalc(record: &mut [u8], item_table: &[u8]) -> Result<()> {
    todo!("battle/native_equipment.go:ApplyNativeEquipmentRecalc")
}

/// ApplyNativeRuntimeEquipmentRecalc reproduces 0x1b750 for one 0x50-byte
/// runtime record rooted at [0x53a45]. It differs from persistent 0x1145a:
/// +0x22/+0x23 apply the exact x87 1.15 toward-zero transform to AP/DP, while
/// +0x24 adds 15 to both HIT/EV bases before equipped item words are added.
/// The operation validates every selected item row before changing the record.
///
/// Go：`native_equipment.go:123`，`func ApplyNativeRuntimeEquipmentRecalc(record, itemTable []byte) error`
pub fn apply_native_runtime_equipment_recalc(record: &mut [u8], item_table: &[u8]) -> Result<()> {
    todo!("battle/native_equipment.go:ApplyNativeRuntimeEquipmentRecalc")
}
